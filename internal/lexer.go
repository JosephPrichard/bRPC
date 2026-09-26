package internal

import (
    "fmt"
    "math/big"
    "strconv"
    "strings"
    "unicode"
)

const (
    // TokUnknown etc. are used to control the flow of the parser itself, with error reporting, termination, etc.
    TokUnknown TokKind = iota
    TokErr
    TokEof

    // TokIden etc. represent "variable" data that may need to be parsed later
    TokIden
    TokInteger
    TokFloat
    TokString
    TokTag

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
    case TokFloat:
        return "float"
    case TokString:
        return "string"
    case TokTag:
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
        return "type"
    case TokTypeDef:
        return "typedef"
    case TokField:
        return "struct field"
    case TokCase:
        return "enum case"
    case TokOption:
        return "union option"
    default:
        panic(fmt.Sprintf("assertion error: string func: unknown token: %d", k))
    }
}

type TokVal struct {
    Kind     TokKind
    Str      string
    Expected TokKind // the expected token whenever an error is occurred, only populated for Kind of TokErr
    Int      big.Int // populated for tokens with unbounded integer values (TokOrd, TokInteger)
    Float64  float64 // populated for tokens with float values (TokFloat)
}

func (t TokVal) String() string {
    switch t.Kind {
    case TokUnknown:
        return "<unknown>"
    case TokEof:
        return "<eof>"
    default:
        return t.Str
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
    return Positions{Begin: lex.start, End: lex.curr}
}

func (lex *Lexer) emitValue(tokVal TokVal) {
    lex.tokens = append(lex.tokens, Token{tokVal, lex.makePositions()})
    lex.skip()
}

func (lex *Lexer) emit(kind TokKind) {
    value := lex.span()
    lex.emitValue(TokVal{Kind: kind, Str: value})
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

    lex.tokens = append(lex.tokens, Token{TokVal{Kind: kind, Str: str}, lex.makePositions()})
    lex.skip()
}

func (lex *Lexer) emitFloat(kind TokKind, f64 float64) {
    value := lex.span()
    lex.emitValue(TokVal{Kind: kind, Str: value, Float64: f64})
}

func (lex *Lexer) emitInteger(kind TokKind, i big.Int) {
    value := lex.span()
    lex.emitValue(TokVal{Kind: kind, Str: value, Int: i})
}

func (lex *Lexer) emitNext(kind TokKind) {
    lex.consume()
    lex.emit(kind)
}

func (lex *Lexer) emitError(expected TokKind) struct{} {
    // scan until a sentinel symbol
    if expected == TokComment {
        lex.acceptUntil(newline)
    } else {
        lex.acceptUntil(whitespace + control)
    }

    value := lex.span()
    token := Token{TokVal{Kind: TokErr, Str: value, Expected: expected}, lex.makePositions()}

    lex.tokens = append(lex.tokens, token)
    lex.skip()

    return struct{}{}
}

func (lex *Lexer) consume() {
    lex.curr += lex.width
}

func (lex *Lexer) peek() rune {
    if lex.curr >= len(lex.input) {
        lex.width = 0
        return eof
    }

    // lex.prev, lex.width = utf8.DecodeRuneInString(lex.input[lex.curr:])

    // note(Joseph): Doesn't parse multiple byte unicode characters properly for performance
    // A big assumption here is that we never need to use peek() to check a unicode character
    prev := lex.input[lex.curr]
    lex.width = 1

    return rune(prev)
}

func (lex *Lexer) next() rune {
    prev := lex.peek()
    lex.consume()
    return prev
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

func (lex *Lexer) take(ch rune) bool {
    return ch == lex.next()
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
    for lex.peek() != eof && !strings.ContainsRune(invalid, lex.peek()) {
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

func (lex *Lexer) lexNumeric() struct{} {
    kind := TokInteger

    for {
        lex.acceptWhile(numeric)
        if lex.accept(".") {
            kind = TokFloat
        } else if lex.assert(whitespace + control) {
            break
        } else {
            // stop at first invalid non-numeric
            return lex.emitError(kind)
        }
    }

    numericStr := lex.span()
    switch kind {
    case TokInteger:
        i, ok := new(big.Int).SetString(numericStr, 10)
        if !ok {
            return lex.emitError(kind)
        }
        lex.emitInteger(kind, *i)
    case TokFloat:
        f64, err := strconv.ParseFloat(numericStr, 64)
        if err != nil {
            return lex.emitError(kind)
        }
        lex.emitFloat(kind, f64)
    default:
        panic(fmt.Sprintf("assertion error: numeric kind is unexpected: %s", kind))
    }

    return struct{}{}
}

func (lex *Lexer) lexComment() {
    lex.next()
    if !lex.take('/') {
        lex.emitError(TokComment)
        return
    }
    lex.acceptUntil(newline)
    lex.skip()
}

func (lex *Lexer) lexTag() struct{} {
    kind := TokTag
    lex.next()
    lex.acceptWhile(numeric)
    if !lex.assert(whitespace + control) {
        return lex.emitError(kind)
    }

    // tag must be at least 2 characters long
    if lex.curr-lex.start <= 1 {
        return lex.emitError(kind)
    }

    value := lex.span()

    // invariant: tag is at least 2 characters long
    tagInt, ok := new(big.Int).SetString(value[1:], 10)
    if !ok {
        return lex.emitError(kind)
    }

    lex.emitInteger(TokTag, *tagInt)

    return struct{}{}
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
        lex.lexTag()
    case '"':
        lex.lexString()
    default:
        if lex.accept(numeric) {
            lex.lexNumeric()
        } else if !unicode.IsControl(ch) && !unicode.IsPunct(ch) && !unicode.IsSpace(ch) {
            lex.lexText()
        } else {
            lex.emitError(TokUnknown)
        }
    }
    return true
}

