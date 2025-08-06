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
	return fmt.Sprintf("'%s' at %d,%d:%d,%d", t.TokVal.String(), t.StartRow, t.StartCol, t.EndRow, t.EndCol)
}

type Lexer struct {
	Input    string
	Prev     rune
	Curr     int
	CurrCol  int
	CurrRow  int
	Start    int
	StartCol int
	StartRow int
	Width    int
	Tokens   chan Token
}

const eof = 0

func NewLexer(input string) Lexer {
	return Lexer{Input: input, Tokens: make(chan Token)}
}

func (lex *Lexer) Emit(tok TokType) {
	val := lex.Input[lex.Start:lex.Curr]
	lex.Tokens <- Token{TokVal: TokVal{T: tok, Value: val}, StartRow: lex.StartRow, StartCol: lex.StartCol, EndRow: lex.CurrRow, EndCol: lex.CurrCol}
	lex.Skip()
}

func (lex *Lexer) EmitStr() {
	str := lex.Input[lex.Start:lex.Curr]

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

	lex.Tokens <- Token{TokVal: TokVal{T: tok, Value: str}, StartRow: lex.StartRow, StartCol: lex.StartCol, EndRow: lex.CurrRow, EndCol: lex.CurrCol}
	lex.Skip()
}

func (lex *Lexer) EmitNext(tok TokType) {
	lex.Consume()
	lex.Emit(tok)
}

func (lex *Lexer) Consume() {
	lex.Curr += lex.Width
	if lex.Prev == '\n' {
		lex.CurrCol = 0
		lex.CurrRow++
	} else {
		lex.CurrCol++
	}
}

func (lex *Lexer) Peek() rune {
	if lex.Curr >= len(lex.Input) {
		lex.Width = 0
		return eof
	}
	lex.Prev, lex.Width = utf8.DecodeRuneInString(lex.Input[lex.Curr:])
	return lex.Prev
}

func (lex *Lexer) Next() (r rune) {
	r = lex.Peek()
	lex.Consume()
	return
}

func (lex *Lexer) Skip() {
	lex.Start = lex.Curr
	lex.StartRow = lex.CurrRow
	lex.StartCol = lex.CurrCol
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
	defer close(lex.Tokens)
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
	if lex.Curr-lex.Start <= 1 {
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
