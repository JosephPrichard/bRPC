package internal

import (
	"strings"
	"unicode/utf8"
)

type TokType int

const (
	TokErr TokType = iota
	TokEof

	TokIden
	TokNumber
	TokTerm
	TokSep
	TokLBrace
	TokRBrace
	TokLParen
	TokRParen
	TokArrow
	TokEqual

	TokMessage
	TokService
	TokRequired
	TokOptional
	TokStruct
	TokUnion
	TokEnum
)

type Token struct {
	t        TokType
	val      string
	startPos int
	endPos   int
}

type Lexer struct {
	input  string
	curr   int
	start  int
	width  int
	tokens chan Token
}

const eof = 0

func newLexer(input string) Lexer {
	return Lexer{input: input, tokens: make(chan Token)}
}

func (l *Lexer) emit(tok TokType) {
	l.tokens <- Token{t: tok, val: l.input[l.start:l.curr]}
	l.skip()
}

func (l *Lexer) emitString() {
	str := l.input[l.start:l.curr]
	tok := TokIden
	switch str {
	case "struct":
		tok = TokStruct
	case "union":
		tok = TokUnion
	case "enum":
		tok = TokEnum
	case "message":
		tok = TokMessage
	case "service":
		tok = TokService
	case "required":
		tok = TokRequired
	case "optional":
		tok = TokOptional
	}
	l.tokens <- Token{t: tok, val: str}
	l.skip()
}

func (l *Lexer) next() (r rune) {
	if l.curr >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.curr:])
	l.curr += l.width
	return r
}

func (l *Lexer) peek() rune {
	r := l.next()
	l.unwind()
	return r
}

func (l *Lexer) skip() {
	l.start = l.curr
}

func (l *Lexer) unwind() {
	l.curr -= l.width
}

func (l *Lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.unwind()
	return false
}

func (l *Lexer) acceptWhile(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.unwind()
}

func (l *Lexer) acceptUntil(invalid string) {
	for strings.IndexRune(invalid, l.next()) < 0 {
	}
	l.unwind()
}

func (l *Lexer) assert(valid string) bool {
	if !l.accept(valid) {
		l.emit(TokErr)
		return false
	}
	return true
}

const numeric = "1234567890"
const control = "=(){};,->/"
const whitespace = " \t\r\n\f"
const newline = "\r\n"

func (l *Lexer) loop() {
	defer close(l.tokens)
	for state := lex; state != nil; {
		state = state(l)
	}
}

type LexFn func(*Lexer) LexFn

func lex(l *Lexer) LexFn {
	r := l.next()
	switch r {
	case eof:
		l.emit(TokEof)
		return nil
	// one character cases are simple, just emit the corresponding token
	case '=':
		l.emit(TokEqual)
	case '{':
		l.emit(TokLBrace)
	case '}':
		l.emit(TokRBrace)
	case '(':
		l.emit(TokLParen)
	case ')':
		l.emit(TokRParen)
	case ';':
		l.emit(TokTerm)
	case ',':
		l.emit(TokSep)
	case '-':
		// dash is not valid anywhere else, so it must be an arrow
		if !l.assert("<") {
			return nil
		}
		l.next()
		l.emit(TokArrow)
	case '/':
		// slash is not valid anywhere else, so it must be a comment
		if !l.assert("/") {
			return nil
		}
		l.acceptUntil(newline)
		l.skip()
	}
	if l.accept(numeric) {
		// lex while numeric, a numeric must end in either a control character or a whitespace
		l.acceptWhile(numeric)
		if !l.assert(whitespace + control) {
			return nil
		}
		l.emit(TokNumber)
	} else {
		// lex an iden until whitespace, an iden can contain anything until a whitespace marker
		l.acceptUntil(whitespace)
		l.emitString()
	}
	return lex
}
