package internal

import (
	"strings"
	"unicode/utf8"
)

type TokType int

const (
	TokErr TokType = iota

	TokIden
	TokOrder
	TokTerm
	TokSep
	TokLBrace
	TokRBrace
	TokLParen
	TokLBrack
	TokRBrack
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

type TokVal struct {
	T     TokType
	Value string
}

func (t TokVal) String() string {
	return t.Value
}

type Token struct {
	TokVal
	StartPos int
	EndPost  int
}

func (t Token) String() string {
	return t.TokVal.String()
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

func (l *Lexer) Emit(tok TokType) {
	val := l.input[l.start:l.curr]
	l.tokens <- Token{TokVal: TokVal{T: tok, Value: val}, StartPos: l.start, EndPost: l.curr}
	l.Skip()
}

func (l *Lexer) EmitErr() LexFn {
	l.Emit(TokErr)
	return nil
}

func (l *Lexer) EmitStr() {
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
	l.tokens <- Token{TokVal: TokVal{T: tok, Value: str}, StartPos: l.start, EndPost: l.curr}
	l.Skip()
}

func (l *Lexer) Consume(tok TokType) {
	l.Next()
	l.Emit(tok)
}

func (l *Lexer) Next() (r rune) {
	if l.curr >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.curr:])
	l.curr += l.width
	return r
}

func (l *Lexer) Peek() rune {
	r := l.Next()
	l.Unwind()
	return r
}

func (l *Lexer) Skip() {
	l.start = l.curr
}

func (l *Lexer) Unwind() {
	l.curr -= l.width
}

func (l *Lexer) Accept(valid string) bool {
	if strings.IndexRune(valid, l.Next()) >= 0 {
		return true
	}
	l.Unwind()
	return false
}

func (l *Lexer) Take(valid string) bool {
	if strings.IndexRune(valid, l.Next()) >= 0 {
		return true
	}
	return false
}

func (l *Lexer) Assert(valid string) bool {
	if strings.IndexRune(valid, l.Peek()) >= 0 {
		return true
	}
	l.Next()
	return false
}

func (l *Lexer) AcceptWhile(valid string) {
	for strings.IndexRune(valid, l.Next()) >= 0 {
	}
	l.Unwind()
}

func (l *Lexer) AcceptUntil(invalid string) {
	for strings.IndexRune(invalid, l.Next()) < 0 {
	}
	l.Unwind()
}

const numeric = "1234567890"
const control = "=()[]{};,->/"
const whitespace = " \t\r\n\f"
const newline = "\r\n"

func (l *Lexer) Run() {
	defer close(l.tokens)
	for state := Lex; state != nil; {
		state = state(l)
	}
}

type LexFn func(*Lexer) LexFn

func LexOrder(l *Lexer) LexFn {
	l.AcceptWhile(numeric)
	if !l.Assert(whitespace + control) {
		return l.EmitErr()
	}
	l.Emit(TokOrder)
	return Lex
}

func LexArrow(l *Lexer) LexFn {
	l.Next()
	if !l.Take(">") {
		return l.EmitErr()
	}
	l.Emit(TokArrow)
	return Lex
}

func LexComment(l *Lexer) LexFn {
	l.Next()
	if !l.Take("/") {
		return l.EmitErr()
	}
	l.AcceptUntil(newline)
	l.Skip()
	return Lex
}

func LexString(l *Lexer) LexFn {
	l.AcceptUntil(whitespace + control)
	l.EmitStr()
	return Lex
}

func Lex(l *Lexer) LexFn {
	l.AcceptWhile(whitespace)
	l.Skip()

	switch l.Peek() {
	case eof:
		return nil
	case '=':
		l.Consume(TokEqual)
	case '{':
		l.Consume(TokLBrace)
	case '}':
		l.Consume(TokRBrace)
	case '(':
		l.Consume(TokLParen)
	case ')':
		l.Consume(TokRParen)
	case '[':
		l.Consume(TokLBrack)
	case ']':
		l.Consume(TokRBrack)
	case ';':
		l.Consume(TokTerm)
	case ',':
		l.Consume(TokSep)
	case '-':
		return LexArrow(l)
	case '/':
		return LexComment(l)
	default:
		if l.Accept(numeric) {
			return LexOrder(l)
		} else {
			return LexString(l)
		}
	}
	return Lex
}
