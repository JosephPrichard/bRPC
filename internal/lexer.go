package internal

import (
	"fmt"
	"strconv"
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
	TokDeprecated
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
	case TokDeprecated:
		return "deprecated"
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
	Expected TokKind // the expected token whenever an error is occurred, only populated for Kind of TokErr
	Num      uint64  // only populated if the token has a numeric value (TokOrd, TokInteger)
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

func (lex *Lexer) span() string {
	return lex.input[lex.start:lex.curr]
}

func (lex *Lexer) makePositions() Positions {
	return Positions{B: lex.start, E: lex.curr}
}

func (lex *Lexer) emit(kind TokKind) {
	value := lex.span()
	lex.tokens = append(lex.tokens, Token{TokVal{Kind: kind, Value: value}, lex.makePositions()})
	lex.skip()
}

func (lex *Lexer) emitText() {
	str := lex.span()

	kind := TokIden
	switch str {
	case "struct":
		kind = TokStruct
	case "union":
		kind = TokUnion
	case "enum":
		kind = TokEnum
	case "message":
		kind = TokMessage
	case "service":
		kind = TokService
	case "required":
		kind = TokRequired
	case "optional":
		kind = TokOptional
	case "deprecated":
		kind = TokDeprecated
	case "returns":
		kind = TokReturns
	case "rpc":
		kind = TokRpc
	case "import":
		kind = TokImport
	}

	lex.tokens = append(lex.tokens, Token{TokVal{Kind: kind, Value: str}, lex.makePositions()})
	lex.skip()
}

func (lex *Lexer) emitNumeric(kind TokKind, num uint64) {
	value := lex.span()
	lex.tokens = append(lex.tokens, Token{TokVal{Kind: kind, Value: value, Num: num}, lex.makePositions()})
	lex.skip()
}

func (lex *Lexer) emitNext(kind TokKind) {
	lex.consume()
	lex.emit(kind)
}

func (lex *Lexer) emitErr(expected TokKind) {
	// scan until a sentinel symbol
	if expected == TokComment {
		lex.acceptUntil(newline)
	} else {
		lex.acceptUntil(whitespace + control)
	}

	value := lex.span()
	token := Token{TokVal{Kind: TokErr, Value: value, Expected: expected}, lex.makePositions()}

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
	if strings.ContainsRune(valid, lex.peek()) {
		lex.consume()
		return true
	}
	return false
}

func (lex *Lexer) take(valid string) bool {
	return strings.ContainsRune(valid, lex.next())
}

func (lex *Lexer) assert(valid string) bool {
	if strings.ContainsRune(valid, lex.peek()) {
		return true
	}
	lex.consume()
	return false
}

func (lex *Lexer) acceptWhile(valid string) {
	for strings.ContainsRune(valid, lex.peek()) {
		lex.consume()
	}
}

func (lex *Lexer) acceptUntil(invalid string) {
	for !strings.ContainsRune(invalid, lex.peek()) {
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
	kind := TokInteger
	lex.acceptWhile(numeric)
	if !lex.assert(whitespace + control) {
		lex.emitErr(kind)
		return
	}

	numStr := lex.span()
	num, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		// we can panic here because the lexer should have stopped if there were any non-numerics
		panic(fmt.Sprintf("assertion error: integer token is invalid: %v", err))
	}
	lex.emitNumeric(TokInteger, num)
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
	kind := TokOrd
	lex.next()
	lex.acceptWhile(numeric)
	if !lex.assert(whitespace + control) {
		lex.emitErr(kind)
		return
	}
	if lex.curr-lex.start <= 1 {
		lex.emitErr(kind)
		return
	}

	value := lex.span()
	ord, strErr := strconv.ParseUint(value[1:], 10, 64)
	if strErr != nil {
		// we can panic here because the lexer should have stopped if there were any non-numerics
		panic(fmt.Sprintf("assertion error: ord token is invalid: %v", strErr))
	}
	lex.emitNumeric(TokOrd, ord)
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
