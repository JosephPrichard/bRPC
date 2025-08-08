package internal

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	TokErr TokType = iota
	TokEof
	TokIden
	TokInteger
	TokString
	TokOrd
	TokTerminal
	TokSep
	TokLBrace
	TokRBrace
	TokLParen
	TokRParen
	TokLBrack
	TokRBrack
	TokEqual
	TokPipe
	TokMessage
	TokService
	TokRequired
	TokOptional
	TokStruct
	TokUnion
	TokEnum
	TokReturns
	TokRpc
)

type TokType int

func (t TokType) String() string {
	switch t {
	case TokErr:
		return "<error>"
	case TokEof:
		return "<eof>"
	case TokIden:
		return "<iden>"
	case TokInteger:
		return "<integer>"
	case TokString:
		return "<string>"
	case TokOrd:
		return "<ord>"
	case TokTerminal:
		return "';'"
	case TokSep:
		return "','"
	case TokLBrace:
		return "'{'"
	case TokRBrace:
		return "'}'"
	case TokLParen:
		return "'('"
	case TokRParen:
		return "')'"
	case TokLBrack:
		return "'['"
	case TokRBrack:
		return "']'"
	case TokEqual:
		return "'='"
	case TokPipe:
		return "'|'"
	case TokMessage:
		return "'message'"
	case TokService:
		return "'service'"
	case TokRequired:
		return "'required'"
	case TokOptional:
		return "'optional'"
	case TokStruct:
		return "'struct'"
	case TokUnion:
		return "'union'"
	case TokEnum:
		return "'enum'"
	case TokReturns:
		return "'returns'"
	case TokRpc:
		return "'rpc'"
	}
	return ""
}

type TokVal struct {
	t     TokType
	value string
}

func (t TokVal) String() string {
	return t.value
}

type Token struct {
	TokVal
	startRow int
	startCol int
	endRow   int
	endCol   int
}

func (t Token) String() string {
	return fmt.Sprintf("'%s' at %d,%d:%d,%d", t.TokVal.String(), t.startRow, t.startCol, t.endRow, t.endCol)
}

type Lexer struct {
	input    string
	prev     rune
	curr     int
	currCol  int
	currRow  int
	start    int
	startCol int
	startRow int
	width    int
	tokens   chan Token
}

const eof = 0

func newLexer(input string) Lexer {
	return Lexer{input: input, tokens: make(chan Token), startCol: 1, startRow: 1, currCol: 1, currRow: 1}
}

func (lex *Lexer) emit(tok TokType) {
	val := lex.input[lex.start:lex.curr]
	lex.tokens <- Token{TokVal: TokVal{t: tok, value: val}, startRow: lex.startRow, startCol: lex.startCol, endRow: lex.currRow, endCol: lex.currCol}
	lex.skip()
}

func (lex *Lexer) emitText() {
	str := lex.input[lex.start:lex.curr]

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
	case "returns":
		tok = TokReturns
	case "rpc":
		tok = TokRpc
	}

	lex.tokens <- Token{TokVal: TokVal{t: tok, value: str}, startRow: lex.startRow, startCol: lex.startCol, endRow: lex.currRow, endCol: lex.currCol}
	lex.skip()
}

func (lex *Lexer) emitNext(tok TokType) {
	lex.consume()
	lex.emit(tok)
}

func (lex *Lexer) consume() {
	lex.curr += lex.width
	if lex.prev == '\n' {
		lex.currCol = 1
		lex.currRow++
	} else {
		lex.currCol++
	}
}

func (lex *Lexer) peek() rune {
	if lex.curr >= len(lex.input) {
		lex.width = 0
		return eof
	}
	lex.prev, lex.width = utf8.DecodeRuneInString(lex.input[lex.curr:])
	return lex.prev
}

func (lex *Lexer) next() (r rune) {
	r = lex.peek()
	lex.consume()
	return
}

func (lex *Lexer) skip() {
	lex.start = lex.curr
	lex.startRow = lex.currRow
	lex.startCol = lex.currCol
}

func (lex *Lexer) accept(valid string) bool {
	if strings.IndexRune(valid, lex.peek()) >= 0 {
		lex.consume()
		return true
	}
	return false
}

func (lex *Lexer) take(valid string) bool {
	if strings.IndexRune(valid, lex.next()) >= 0 {
		return true
	}
	return false
}

func (lex *Lexer) assert(valid string) bool {
	if strings.IndexRune(valid, lex.peek()) >= 0 {
		return true
	}
	lex.consume()
	return false
}

func (lex *Lexer) acceptWhile(valid string) {
	for strings.IndexRune(valid, lex.peek()) >= 0 {
		lex.consume()
	}
}

func (lex *Lexer) acceptUntil(invalid string) {
	for strings.IndexRune(invalid, lex.peek()) < 0 {
		lex.consume()
	}
}

func (lex *Lexer) emitErr() {
	lex.acceptUntil(whitespace + control) // scan until a sentinel symbol
	lex.emit(TokErr)
}

const numeric = "1234567890"
const control = "=()[]{};,/|"
const whitespace = " \t\r\n\f"
const newline = "\r\n"

func (lex *Lexer) run() {
	defer close(lex.tokens)
	for hasNext := true; hasNext; {
		hasNext = lex.lex()
	}
}

func (lex *Lexer) lexInteger() {
	lex.acceptWhile(numeric)
	if !lex.assert(whitespace + control) {
		lex.emitErr()
		return
	}
	lex.emit(TokInteger)
}

func (lex *Lexer) lexComment() {
	lex.next()
	if !lex.take("/") {
		lex.emitErr()
		return
	}
	lex.acceptUntil(newline)
	lex.skip()
}

func (lex *Lexer) lexOrd() {
	lex.next()
	lex.acceptWhile(numeric)
	if !lex.assert(whitespace + control) {
		lex.emitErr()
		return
	}
	if lex.curr-lex.start <= 1 {
		lex.emitErr()
		return
	}
	lex.emit(TokOrd)
}

func (lex *Lexer) lexText() {
	ch := lex.peek()
	if unicode.IsControl(ch) || unicode.IsPunct(ch) || unicode.IsSpace(ch) {
		lex.next()
		lex.emitErr()
		return
	}
	lex.acceptUntil(whitespace + control)
	lex.emitText()
}

func (lex *Lexer) lexString() {
	lex.next()

	isEscaped := false
	for isString := true; isString; {
		ch := lex.next()
		switch ch {
		case '"':
			if !isEscaped {
				isString = false
			}
			isEscaped = false
		case '\\':
			isEscaped = true
		default:
			isEscaped = false
		}
	}

	lex.emit(TokString)
}

func (lex *Lexer) lex() bool {
	lex.acceptWhile(whitespace)
	lex.skip()

	switch lex.peek() {
	case eof:
		lex.emit(TokEof)
		return false
	case '=':
		lex.emitNext(TokEqual)
	case '{':
		lex.emitNext(TokLBrace)
	case '}':
		lex.emitNext(TokRBrace)
	case '(':
		lex.emitNext(TokLParen)
	case ')':
		lex.emitNext(TokRParen)
	case '[':
		lex.emitNext(TokLBrack)
	case ']':
		lex.emitNext(TokRBrack)
	case ';':
		lex.emitNext(TokTerminal)
	case ',':
		lex.emitNext(TokSep)
	case '|':
		lex.emitNext(TokPipe)
	case '/':
		lex.lexComment()
	case '@':
		lex.lexOrd()
	case '"':
		lex.lexString()
	default:
		if lex.accept(numeric) {
			lex.lexInteger()
		} else {
			lex.lexText()
		}
	}
	return true
}
