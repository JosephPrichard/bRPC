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
	TokBinary
	TokHex
	TokOrd
	TokTerm
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
		fallthrough
	case TokHex:
		fallthrough
	case TokBinary:
		return "<integer>"
	case TokOrd:
		return "<ord>"
	case TokTerm:
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
	T     TokType
	Value string
}

func (t TokVal) String() string {
	return t.Value
}

type Token struct {
	TokVal
	StartRow int
	StartCol int
	EndRow   int
	EndCol   int
}

func (t Token) String() string {
	return fmt.Sprintf("%s at %d,%d:%d,%d", t.TokVal.String(), t.StartRow, t.StartCol, t.EndRow, t.EndCol)
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

func NewLexer(input string) Lexer {
	return Lexer{input: input, tokens: make(chan Token)}
}

func (lex *Lexer) Emit(tok TokType) {
	val := lex.input[lex.start:lex.curr]
	lex.tokens <- Token{TokVal: TokVal{T: tok, Value: val}, StartRow: lex.startRow, StartCol: lex.startCol, EndRow: lex.currRow, EndCol: lex.currCol}
	lex.Skip()
}

func (lex *Lexer) EmitStr() {
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

	lex.tokens <- Token{TokVal: TokVal{T: tok, Value: str}, StartRow: lex.startRow, StartCol: lex.startCol, EndRow: lex.currRow, EndCol: lex.currCol}
	lex.Skip()
}

func (lex *Lexer) EmitNext(tok TokType) {
	lex.Consume()
	lex.Emit(tok)
}

func (lex *Lexer) Consume() {
	lex.curr += lex.width
	if lex.prev == '\n' {
		lex.currCol = 0
		lex.currRow++
	} else {
		lex.currCol++
	}
}

func (lex *Lexer) Peek() rune {
	if lex.curr >= len(lex.input) {
		lex.width = 0
		return eof
	}
	lex.prev, lex.width = utf8.DecodeRuneInString(lex.input[lex.curr:])
	return lex.prev
}

func (lex *Lexer) Next() (r rune) {
	r = lex.Peek()
	lex.Consume()
	return
}

func (lex *Lexer) Skip() {
	lex.start = lex.curr
	lex.startRow = lex.currRow
	lex.startCol = lex.currCol
}

func (lex *Lexer) Accept(valid string) bool {
	if strings.IndexRune(valid, lex.Peek()) >= 0 {
		lex.Consume()
		return true
	}
	return false
}

func (lex *Lexer) Take(valid string) bool {
	if strings.IndexRune(valid, lex.Next()) >= 0 {
		return true
	}
	return false
}

func (lex *Lexer) Assert(valid string) bool {
	if strings.IndexRune(valid, lex.Peek()) >= 0 {
		return true
	}
	lex.Consume()
	return false
}

func (lex *Lexer) AcceptWhile(valid string) {
	for strings.IndexRune(valid, lex.Peek()) >= 0 {
		lex.Consume()
	}
}

func (lex *Lexer) AcceptUntil(invalid string) {
	for strings.IndexRune(invalid, lex.Peek()) < 0 {
		lex.Consume()
	}
}

func (lex *Lexer) EmitErr() {
	lex.AcceptUntil(whitespace + control) // scan until a sentinel symbol
	lex.Emit(TokErr)
}

const numeric = "1234567890"
const hex = "0123456789abcdef"
const binary = "01"
const control = "=()[]{};,/|"
const whitespace = " \t\r\n\f"
const newline = "\r\n"

func (lex *Lexer) Run() {
	defer close(lex.tokens)
	for hasNext := true; hasNext; {
		hasNext = lex.Lex()
	}
}

func (lex *Lexer) LexNumeric() {
	var t TokType

	if lex.Accept("bB") {
		lex.Next()
		lex.AcceptWhile(binary)
		t = TokBinary
	} else if lex.Accept("xX") {
		lex.Next()
		lex.AcceptWhile(hex)
		t = TokHex
	} else {
		lex.AcceptWhile(numeric)
		t = TokInteger
	}

	if !lex.Assert(whitespace + control) {
		lex.EmitErr()
		return
	}
	lex.Emit(t)
}

func (lex *Lexer) LexComment() {
	lex.Next()
	if !lex.Take("/") {
		lex.EmitErr()
		return
	}
	lex.AcceptUntil(newline)
	lex.Skip()
}

func (lex *Lexer) LexOrd() {
	lex.Next()
	lex.AcceptWhile(numeric)
	if !lex.Assert(whitespace + control) {
		lex.EmitErr()
		return
	}
	lex.Emit(TokOrd)
}

func (lex *Lexer) LexString() {
	r := lex.Peek()
	if unicode.IsControl(r) || unicode.IsPunct(r) || unicode.IsSpace(r) {
		lex.Next()
		lex.EmitErr()
		return
	}
	lex.AcceptUntil(whitespace + control)
	lex.EmitStr()
}

func (lex *Lexer) Lex() bool {
	lex.AcceptWhile(whitespace)
	lex.Skip()

	switch lex.Peek() {
	case eof:
		lex.Emit(TokEof)
		return false
	case '=':
		lex.EmitNext(TokEqual)
	case '{':
		lex.EmitNext(TokLBrace)
	case '}':
		lex.EmitNext(TokRBrace)
	case '(':
		lex.EmitNext(TokLParen)
	case ')':
		lex.EmitNext(TokRParen)
	case '[':
		lex.EmitNext(TokLBrack)
	case ']':
		lex.EmitNext(TokRBrack)
	case ';':
		lex.EmitNext(TokTerm)
	case ',':
		lex.EmitNext(TokSep)
	case '|':
		lex.EmitNext(TokPipe)
	case '/':
		lex.LexComment()
	case '@':
		lex.LexOrd()
	default:
		if lex.Accept(numeric) {
			lex.LexNumeric()
		} else {
			lex.LexString()
		}
	}
	return true
}
