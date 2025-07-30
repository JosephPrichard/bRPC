package internal

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type TokType int

const (
	TokErr TokType = iota

	TokIden
	TokInteger
	TokOrd
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
	TokPipe

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
	token := Token{
		TokVal:   TokVal{T: tok, Value: val},
		StartRow: lex.startRow,
		StartCol: lex.startCol,
		EndRow:   lex.currRow,
		EndCol:   lex.currCol,
	}
	lex.tokens <- token
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
	}
	token := Token{
		TokVal:   TokVal{T: tok, Value: str},
		StartRow: lex.startRow,
		StartCol: lex.startCol,
		EndRow:   lex.currRow,
		EndCol:   lex.currCol,
	}
	lex.tokens <- token
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

const numeric = "1234567890"
const control = "=()[]{};,->/|"
const whitespace = " \t\r\n\f"
const newline = "\r\n"

func (lex *Lexer) Run() {
	defer close(lex.tokens)
	for hasNext := true; hasNext; {
		hasNext = Lex(lex)
	}
}

func LexInteger(lex *Lexer) bool {
	lex.AcceptWhile(numeric)
	if !lex.Assert(whitespace + control) {
		lex.Emit(TokErr)
		return false
	}
	lex.Emit(TokInteger)
	return true
}

func LexArrow(lex *Lexer) bool {
	lex.Next()
	if !lex.Take(">") {
		lex.Emit(TokErr)
		return false
	}
	lex.Emit(TokArrow)
	return true
}

func LexComment(lex *Lexer) bool {
	lex.Next()
	if !lex.Take("/") {
		lex.Emit(TokErr)
		return false
	}
	lex.AcceptUntil(newline)
	lex.Skip()
	return true
}

func LexOrd(lex *Lexer) bool {
	lex.Next()
	lex.AcceptWhile(numeric)
	if !lex.Assert(whitespace + control) {
		lex.Emit(TokErr)
		return false
	}
	lex.Emit(TokOrd)
	return true
}

func LexString(lex *Lexer) bool {
	r := lex.Peek()
	if unicode.IsControl(r) || unicode.IsPunct(r) || unicode.IsSpace(r) {
		lex.Next()
		lex.Emit(TokErr)
		return false
	}
	lex.AcceptUntil(whitespace + control)
	lex.EmitStr()
	return true
}

func Lex(lex *Lexer) bool {
	lex.AcceptWhile(whitespace)
	lex.Skip()

	switch lex.Peek() {
	case eof:
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
	case '-':
		return LexArrow(lex)
	case '/':
		return LexComment(lex)
	case '@':
		return LexOrd(lex)
	default:
		if lex.Accept(numeric) {
			return LexInteger(lex)
		} else {
			return LexString(lex)
		}
	}
	return true
}
