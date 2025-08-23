package internal

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// TokUnknown etc. are used to control the flow of the parser itself, with error reporting, termination, etc.
	TokUnknown TokKind = iota
	TokErr
	TokEof

	// TokIden etc. represent "variable" data that may need to be parsed later
	TokIden
	TokInteger
	TokString
	TokOrd

	// TokSemicolon etc. are special character tokens used to control termination of ASTs
	TokSemicolon
	TokComma
	TokLBrace
	TokRBrace
	TokLParen
	TokRParen
	TokLBrack
	TokRBrack
	TokEqual

	// TokRequired etc., are "literal" tokens which represent extract symbols for controlling AST creation
	TokRequired
	TokOptional
	TokStruct
	TokUnion
	TokEnum
	TokReturns
	TokRpc
	TokImport
	TokMessage
	TokService

	// TokComment can be an "expected" token, but is never emitted for the parser to consume
	TokComment

	// TokField etc. these are "fake" tokens which represents multiple "literal" tokens, which the parser may expect, but will never attempt to consume
	TokField
	TokTypeRef
	TokTypeDef
	TokTypeAlias
	TokCase
	TokOption
)

type TokKind int

func (k TokKind) String() string {
	switch k {
	case TokErr:
		return "error"
	case TokEof:
		return "eof"
	case TokIden:
		return "iden"
	case TokInteger:
		return "integer"
	case TokString:
		return "string"
	case TokOrd:
		return "ord"
	case TokSemicolon:
		return "';'"
	case TokComma:
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
	case TokMessage:
		return "message"
	case TokService:
		return "service"
	case TokRequired:
		return "required"
	case TokOptional:
		return "optional"
	case TokStruct:
		return "struct"
	case TokUnion:
		return "union"
	case TokEnum:
		return "enum"
	case TokReturns:
		return "returns"
	case TokRpc:
		return "rpc"
	case TokImport:
		return "import"
	case TokComment:
		return "'//'"
	case TokTypeRef:
		return "typeref"
	case TokTypeDef:
		return "typedef"
	case TokField:
		return "field"
	case TokCase:
		return "case"
	case TokOption:
		return "option"
	default:
		panic(fmt.Sprintf("assertion error: unknown token: %d", k))
	}
}

type TokVal struct {
	Kind     TokKind
	Value    string
	Expected TokKind // the expected value for this token, only populated if Kind is TokErr
}

func (t TokVal) String() string {
	switch t.Kind {
	case TokUnknown:
		return "<unknown>"
	case TokEof:
		return "<eof>"
	default:
		return t.Value
	}
}

type Token struct {
	TokVal
	Positions
}

func (t Token) String() string {
	return fmt.Sprintf("'%s'", t.TokVal.String())
}

type Lexer struct {
	input  string
	prev   rune
	curr   int
	start  int
	width  int
	tokens []Token
}

const eof = 0

func makeLexer(input string) Lexer {
	return Lexer{input: input, tokens: make([]Token, 0)}
}

func (lex *Lexer) emit(tok TokKind) {
	val := lex.input[lex.start:lex.curr]
	lex.tokens = append(lex.tokens, Token{TokVal{Kind: tok, Value: val}, Positions{B: lex.start, E: lex.curr}})
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
	case "import":
		tok = TokImport
	}

	lex.tokens = append(lex.tokens, Token{TokVal{Kind: tok, Value: str}, Positions{B: lex.start, E: lex.curr}})
	lex.skip()
}

func (lex *Lexer) emitNext(tok TokKind) {
	lex.consume()
	lex.emit(tok)
}

func (lex *Lexer) emitErr(expected TokKind) {
	// scan until a sentinel symbol
	if expected == TokComment {
		lex.acceptUntil(newline)
	} else {
		lex.acceptUntil(whitespace + control)
	}

	val := lex.input[lex.start:lex.curr]
	token := Token{
		TokVal{Kind: TokErr, Value: val, Expected: expected},
		Positions{B: lex.start, E: lex.curr},
	}
	lex.tokens = append(lex.tokens, token)
	lex.skip()
}

func (lex *Lexer) consume() {
	lex.curr += lex.width
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

const numeric = "1234567890"
const control = "=()[]{};,/|"
const whitespace = " \t\r\n\f"
const newline = "\r\n"

func (lex *Lexer) run() {
	for hasNext := true; hasNext; {
		hasNext = lex.lex()
	}
}

func (lex *Lexer) lexInteger() {
	tok := TokInteger
	lex.acceptWhile(numeric)
	if !lex.assert(whitespace + control) {
		lex.emitErr(tok)
		return
	}
	lex.emit(tok)
}

func (lex *Lexer) lexComment() {
	lex.next()
	if !lex.take("/") {
		lex.emitErr(TokComment)
		return
	}
	lex.acceptUntil(newline)
	lex.skip()
}

func (lex *Lexer) lexOrd() {
	tok := TokOrd
	lex.next()
	lex.acceptWhile(numeric)
	if !lex.assert(whitespace + control) {
		lex.emitErr(tok)
		return
	}
	if lex.curr-lex.start <= 1 {
		lex.emitErr(tok)
		return
	}
	lex.emit(tok)
}

func (lex *Lexer) lexText() {
	lex.acceptUntil(whitespace + control)
	lex.emitText()
}

func (lex *Lexer) lexString() {
	lex.next()

	isEscaped := false
	for isTerminal := false; !isTerminal; {
		ch := lex.next()
		switch ch {
		case '"':
			if !isEscaped {
				isTerminal = true
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

	ch := lex.peek()
	switch ch {
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
		lex.emitNext(TokSemicolon)
	case ',':
		lex.emitNext(TokComma)
	case '/':
		lex.lexComment()
	case '@':
		lex.lexOrd()
	case '"':
		lex.lexString()
	default:
		if lex.accept(numeric) {
			lex.lexInteger()
		} else if !unicode.IsControl(ch) && !unicode.IsPunct(ch) && !unicode.IsSpace(ch) {
			lex.lexText()
		} else {
			lex.emitErr(TokUnknown)
		}
	}
	return true
}
