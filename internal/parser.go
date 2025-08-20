package internal

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type Parser struct {
	tokens    []Token
	curr      int
	asts      []Ast
	errs      []error
	hasEofErr bool // stores whether an error has been emitted after token stream has reached eof
}

func makeParser(tokens []Token) Parser {
	return Parser{tokens: tokens, hasEofErr: false}
}

func runParser(input string) ([]Ast, []error) {
	lex := makeLexer(input)
	lex.run()

	p := makeParser(lex.tokens)
	p.parse()

	return p.asts, p.errs
}

func (p *Parser) next() Token {
	token := p.tokens[p.curr]
	if token.Kind != TokEof {
		p.curr++
	}
	return token
}

func (p *Parser) eat() {
	if p.peek().Kind != TokEof {
		p.curr++
	}
}

func (p *Parser) prev() {
	p.curr--
	if p.curr < 0 {
		panic("assertion error: curr position in parser should never be less than 0")
	}
}

func (p *Parser) peek() Token {
	return p.tokens[p.curr]
}

func (p *Parser) expect(expected TokKind) (Token, ParserError) {
	token := p.next()
	var err ParserError
	if expected != token.Kind {
		err = makeExpectErr(token, expected)
	}
	return token, err
}

func (p *Parser) eatWhile(expected TokKind) (Token, bool) {
	firstToken := p.peek()
	ok := false
	for p.peek().Kind == expected {
		if !ok {
			ok = true
		}
		p.eat()
	}
	return firstToken, ok
}

func (p *Parser) expectChain(chain ...TokKind) ParserError {
	for _, expected := range chain {
		if _, err := p.expect(expected); err != nil {
			return err
		}
	}
	return nil
}

// EatTokens sentinel tokens which are eaten during forwarding
var EatTokens = []TokKind{TokSemicolon}

// StopTokens sentinel tokens which are stopped at during forwarding
var StopTokens = []TokKind{TokLBrace, TokRBrace, TokService, TokRpc, TokRequired, TokOptional, TokMessage, TokStruct, TokUnion, TokEnum}

func (p *Parser) skipUntilSentinel() {
	for {
		token := p.peek()
		if token.Kind == TokEof {
			return
		}
		matchIdx := slices.Index(EatTokens, token.Kind)
		if matchIdx != -1 {
			p.eatWhile(EatTokens[matchIdx])
			return
		}
		if slices.Index(StopTokens, token.Kind) != -1 {
			return
		}
		p.eat()
	}
}

func (p *Parser) emitError(err error) {
	if !p.hasEofErr {
		// don't emit anymore errors if a single err has been emitted after reaching eof
		p.errs = append(p.errs, err)
	}
	p.hasEofErr = p.peek().Kind == TokEof
}

var Eof = errors.New("reached end of stream while parsing")

func (p *Parser) parse() {
	for {
		root, err := p.parseRoot()
		if errors.Is(err, Eof) {
			break
		}
		if err != nil {
			p.emitError(err)
			p.skipUntilSentinel()
		}
		if root != nil {
			p.asts = append(p.asts, root)
		}
	}
}

func (p *Parser) parseRoot() (Ast, error) {
	var ast Ast
	var err error

	token := p.peek()
	switch token.Kind {
	case TokEof:
		return nil, Eof
	case TokMessage:
		p.eat()
		ast, err = p.parseMessage()
	case TokService:
		ast = p.parseService()
	case TokImport:
		ast = p.parseImport()
	case TokIden:
		ast = p.parseProperty()
	default:
		p.eat()
		err = makeExpectErr(token, TokMessage, TokService, TokImport, TokIden)
	}

	return ast, err
}

func (p *Parser) parseProperty() Ast {
	var property PropertyAst

	forwardErr := func(err ParserError) Ast {
		property.E = err.token().E
		property.Poisoned = true
		err.addKind(PropertyAstKind)
		p.skipUntilSentinel()
		p.emitError(err)
		return &property
	}

	var token Token

	token, err := p.expect(TokIden)
	if err != nil {
		panic(fmt.Sprintf("assertion error: %s", err))
	}
	property.B = token.B
	property.Name = token.Value

	if _, err := p.expect(TokEqual); err != nil {
		return forwardErr(err)
	}

	str, err := p.parseString(&token)
	if err != nil {
		return forwardErr(err)
	}
	property.E = token.E
	property.Value = str

	return &property
}

var escSeqTable = map[rune]rune{'\\': '\\', 'n': '\n', '\t': '\t', 'f': '\f', 'r': '\r', '"': '"'}

func (p *Parser) parseString(token *Token) (string, ParserError) {
	t, err := p.expect(TokString)
	if err != nil {
		return "", err
	}
	*token = t

	if len(token.Value) < 2 {
		panic(fmt.Sprintf("assertion error: import path string must be at least length 2, was: %s", token.Value))
	}

	var sb strings.Builder

	isEscaped := false
	str := token.Value[1 : len(token.Value)-1]

	for _, ch := range str {
		if isEscaped {
			ch, ok := escSeqTable[ch]
			if !ok {
				return "", makeMessageErr(*token, fmt.Sprintf("invalid escape sequence: '/%c'", ch))
			}
			isEscaped = false
			sb.WriteRune(ch)
		} else {
			if ch == '\\' {
				isEscaped = true
			} else {
				sb.WriteRune(ch)
			}
		}
	}

	return sb.String(), nil
}

func (p *Parser) parseImport() Ast {
	var imp ImportAst

	forwardErr := func(err ParserError) Ast {
		imp.E = err.token().E
		imp.Poisoned = true
		err.addKind(ImportAstKind)
		p.skipUntilSentinel()
		p.emitError(err)
		return &imp
	}

	var token Token

	token, err := p.expect(TokImport)
	if err != nil {
		panic(fmt.Sprintf("assertion error: %s", err))
	}
	imp.B = token.B

	pathStr, err := p.parseString(&token)
	if err != nil {
		return forwardErr(err)
	}
	imp.E = token.E
	imp.Path = pathStr

	return &imp
}

func (p *Parser) parseMessage() (Ast, ParserError) {
	// invariant: assume that 'message' token has been consumed
	token, err := p.expect(TokIden)
	if err != nil {
		return nil, err
	}
	name := token.Value

	return p.parseType(name)
}

func (p *Parser) parseTypeParams() ([]string, ParserError) {
	var typeParams []string

	token := p.next()
	switch token.Kind {
	case TokLBrace:
	case TokLParen:
		for parsing := true; parsing; {
			token := p.next()
			switch token.Kind {
			case TokRParen:
				parsing = false
			case TokIden:
				typeParams = append(typeParams, token.Value)
			default:
				return nil, makeExpectErr(token, TokRParen, TokIden)
			}
		}
		if _, err := p.expect(TokLBrace); err != nil {
			return nil, err
		}
	default:
		return nil, makeExpectErr(token, TokLBrace, TokLParen)
	}

	return typeParams, nil
}

func (p *Parser) parseStruct(name string) Ast {
	var strct StructAst
	strct.Name = name

	forwardErr := func(err ParserError) {
		strct.E = err.token().E
		strct.Poisoned = true
		err.addKind(StructAstKind)
		p.skipUntilSentinel()
		p.emitError(err)
	}

	token, err := p.expect(TokStruct)
	if err != nil {
		panic(fmt.Sprintf("assertion error: in struct: %s", err))
	}
	strct.B = token.B

	typeArgs, err := p.parseTypeParams()
	if err != nil {
		forwardErr(err)
		return &strct
	}
	strct.TypeParams = typeArgs

	for {
		token := p.next()
		switch token.Kind {
		case TokOptional, TokRequired:
			p.prev()
			field := p.parseField()
			strct.Fields = append(strct.Fields, field)
		case TokMessage:
			message, err := p.parseMessage()
			if err != nil {
				forwardErr(err)
				continue
			}
			strct.LocalDefs = append(strct.LocalDefs, message)
		case TokRBrace:
			strct.E = token.E
			return &strct
		default:
			forwardErr(makeExpectErr(token, TokField, TokMessage, TokRBrace))
			if token.Kind == TokEof {
				return &strct
			}
		}
	}
}

func (p *Parser) parseOption() OptionAst {
	var option OptionAst

	forwardErr := func(err ParserError) OptionAst {
		option.E = err.token().E
		option.Poisoned = true
		err.addKind(OptionAstKind)
		p.skipUntilSentinel()
		p.emitError(err)
		return option
	}

	var token Token

	ord, err := p.parseOrdWithToken(&token)
	if err != nil {
		return forwardErr(err)
	}
	option.B = token.B
	option.Ord = ord

	typ, err := p.parseType("")
	if err != nil {
		return forwardErr(err)
	}
	option.Type = typ

	if firstToken, ok := p.eatWhile(TokSemicolon); !ok {
		return forwardErr(makeExpectErr(firstToken, TokSemicolon))
	}

	return option
}

func (p *Parser) parseUnion(name string) Ast {
	var union UnionAst
	union.Name = name

	forwardErr := func(err ParserError) {
		union.E = err.token().E
		union.Poisoned = true
		err.addKind(UnionAstKind)
		p.skipUntilSentinel()
		p.emitError(err)
	}

	token, err := p.expect(TokUnion)
	if err != nil {
		panic(fmt.Sprintf("assertion error: in union: %s", err))
	}
	union.B = token.B

	typeArgs, err := p.parseTypeParams()
	if err != nil {
		forwardErr(err)
		return &union
	}
	union.TypeParams = typeArgs

	for {
		token := p.next()
		switch token.Kind {
		case TokOrd:
			p.prev()
			option := p.parseOption()
			union.Options = append(union.Options, option)
		case TokMessage:
			message, err := p.parseMessage()
			if err != nil {
				forwardErr(err)
				continue
			}
			union.LocalDefs = append(union.LocalDefs, message)
		case TokRBrace:
			union.E = token.E
			return &union
		default:
			forwardErr(makeExpectErr(token, TokOption, TokMessage, TokRBrace))
			if token.Kind == TokEof {
				return &union
			}
		}

	}
}

func (p *Parser) parseNumeric() (int64, ParserError) {
	token, err := p.expect(TokInteger)
	if err != nil {
		return 0, err
	}
	value, convErr := strconv.ParseInt(token.Value, 10, 64)
	if convErr != nil {
		return 0, makeMessageErr(token, fmt.Sprintf("%v: %s is not a valid integer", err, token.String()))
	}
	return value, nil
}

func (p *Parser) parseCase() CaseAst {
	var ec CaseAst

	forwardErr := func(err ParserError) CaseAst {
		ec.Poisoned = true
		err.addKind(CaseAstKind)
		p.skipUntilSentinel()
		p.emitError(err)
		return ec
	}

	ord, err := p.parseOrd()
	if err != nil {
		return forwardErr(err)
	}
	ec.Ord = ord

	token, err := p.expect(TokIden)
	if err != nil {
		return forwardErr(err)
	}
	ec.Name = token.Value

	if firstToken, ok := p.eatWhile(TokSemicolon); !ok {
		return forwardErr(makeExpectErr(firstToken, TokSemicolon))
	}
	return ec
}

func (p *Parser) parseEnum(name string) Ast {
	var enum EnumAst
	enum.Name = name

	forwardErr := func(err ParserError) {
		enum.E = err.token().E
		enum.Poisoned = true
		err.addKind(EnumAstKind)
		p.skipUntilSentinel()
		p.emitError(err)
	}

	token, err := p.expect(TokEnum)
	if err != nil {
		panic(fmt.Sprintf("assertion error: in enum: %s", err))
	}
	enum.B = token.B

	if _, err := p.expect(TokLBrace); err != nil {
		forwardErr(err)
		return &enum
	}
	for {
		token := p.next()
		switch token.Kind {
		case TokOrd:
			p.prev()
			ec := p.parseCase()
			enum.Cases = append(enum.Cases, ec)
		case TokRBrace:
			enum.E = token.E
			return &enum
		default:
			forwardErr(makeExpectErr(token, TokCase, TokRBrace))
			if token.Kind == TokEof {
				return &enum
			}
		}
	}
}

func (p *Parser) parseArraySize() (uint64, ParserError) {
	token := p.next()
	switch token.Kind {
	case TokInteger:
		if _, err := p.expect(TokRBrack); err != nil {
			return 0, err
		}
		size, err := strconv.ParseUint(token.Value, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("assertion error: integer token is invalid: %v", err))
		}
		return size, nil
	case TokRBrack:
		return 0, nil
	default:
		return 0, makeExpectErr(token, TokInteger, TokRBrack)
	}
}

func (p *Parser) parseTypeArgs(token *Token) ([]Ast, ParserError) {
	var typeArgs []Ast

	if p.peek().Kind != TokLParen {
		return typeArgs, nil
	}
	p.eat()

	for {
		if p.peek().Kind == TokRParen {
			*token = p.next()
			return typeArgs, nil
		}
		typ, err := p.parseType("")
		if err != nil {
			return nil, err
		}
		typeArgs = append(typeArgs, typ)
	}
}

func (p *Parser) parseType(name string) (Ast, ParserError) {
	// each element of the array is a nested array index
	var array []uint64
	var arrTokenB Token

	forwardErr := func(err ParserError, kind AstKind) (Ast, ParserError) {
		err.addKind(kind)
		// don't emit the error, caller will handle this
		return nil, err
	}

	makeTypeAst := func(ast Ast) Ast {
		// an array's last token is the same as it's leaf ast (since a type is nested inside an array)
		if array != nil {
			return &TypeArrAst{Type: ast, Size: array, Range: Range{B: arrTokenB.B, E: ast.End()}}
		}
		return ast
	}

	for {
		token := p.peek()
		switch token.Kind {
		case TokLBrack:
			if arrTokenB.Kind == TokUnknown {
				// if begin token is unset, we know we're at the first array token
				arrTokenB = token
			}
			p.eat()
			size, err := p.parseArraySize()
			if err != nil {
				return forwardErr(err, ArrayAstKind)
			}
			array = append(array, size)
		case TokIden:
			tokenB := p.next()
			tokenE := tokenB
			typeArgs, err := p.parseTypeArgs(&tokenE)
			if err != nil {
				return forwardErr(err, TypeAstKind)
			}
			ast := makeTypeAst(&TypeRefAst{Alias: name, Iden: tokenB.Value, TypeArgs: typeArgs, Range: Range{B: tokenB.B, E: tokenE.E}})
			return ast, nil
		case TokStruct:
			ast := makeTypeAst(p.parseStruct(name))
			return ast, nil
		case TokUnion:
			ast := makeTypeAst(p.parseUnion(name))
			return ast, nil
		case TokEnum:
			ast := makeTypeAst(p.parseEnum(name))
			return ast, nil
		default:
			return nil, makeExpectErr(token, TokType)
		}
	}
}

func (p *Parser) parseOrd() (uint64, ParserError) {
	var token Token
	return p.parseOrdWithToken(&token)
}

// parseOrdWithToken 'writes back' the token it reads to the caller for further processing
func (p *Parser) parseOrdWithToken(token *Token) (uint64, ParserError) {
	t, err := p.expect(TokOrd)
	if err != nil {
		return 0, err
	}
	*token = t
	if len(token.Value) < 2 {
		panic(fmt.Errorf("assertion error: an ord should have at least 2 characters"))
	}
	ord, strErr := strconv.ParseUint(token.Value[1:], 10, 64)
	if strErr != nil {
		panic(fmt.Sprintf("assertion error: ord token is invalid: %v", err))
	}
	return ord, nil
}

func (p *Parser) parseField() FieldAst {
	var field FieldAst

	forwardErr := func(err ParserError) FieldAst {
		field.E = err.token().E
		field.Poisoned = true
		err.addKind(FieldAstKind)
		p.skipUntilSentinel()
		p.emitError(err)
		return field
	}

	var token Token
	var err ParserError
	var ord uint64

	token = p.next()
	field.B = token.B

	switch token.Kind {
	case TokRequired:
		field.Modifier = Required
	case TokOptional:
		field.Modifier = Optional
	default:
		return forwardErr(makeExpectErr(token, TokRequired, TokOptional))
	}

	if token, err = p.expect(TokIden); err != nil {
		return forwardErr(err)
	}
	field.Name = token.Value

	if ord, err = p.parseOrd(); err != nil {
		return forwardErr(err)
	}
	field.Ord = ord

	typ, err := p.parseType("")
	if err != nil {
		return forwardErr(err)
	}
	field.Type = typ

	firstToken, ok := p.eatWhile(TokSemicolon)
	if !ok {
		return forwardErr(makeExpectErr(firstToken, TokSemicolon))
	}
	field.E = firstToken.E

	return field
}

func (p *Parser) parseService() Ast {
	var svc ServiceAst

	forwardErr := func(err ParserError) {
		svc.E = err.token().E
		svc.Poisoned = true
		err.addKind(ServiceAstKind)
		p.skipUntilSentinel()
		p.emitError(err)
	}

	token, err := p.expect(TokService)
	if err != nil {
		panic(fmt.Sprintf("assertion error: in service: %s", err))
	}

	token, err = p.expect(TokIden)
	if err != nil {
		forwardErr(err)
		return &svc
	}
	svc.Name = token.Value
	if _, err := p.expect(TokLBrace); err != nil {
		forwardErr(err)
		return &svc
	}

	for {
		token = p.next()
		switch token.Kind {
		case TokRpc:
			p.prev()
			rpc := p.parseRpc()
			svc.Procedures = append(svc.Procedures, rpc)
		case TokMessage:
			message, err := p.parseMessage()
			if err != nil {
				forwardErr(err)
				continue
			}
			svc.LocalDefs = append(svc.LocalDefs, message)
		case TokRBrace:
			return &svc
		default:
			forwardErr(makeExpectErr(token, TokRpc, TokMessage, TokRBrace))
			if token.Kind == TokEof {
				return &svc
			}
		}
	}
}

func (p *Parser) parseRpc() RpcAst {
	var rpc RpcAst

	forwardErr := func(err ParserError) RpcAst {
		rpc.E = err.token().E
		rpc.Poisoned = true
		err.addKind(RpcAstKind)
		p.skipUntilSentinel()
		p.emitError(err)
		return rpc
	}

	token, err := p.expect(TokRpc)
	if err != nil {
		panic(fmt.Sprintf("assertion error: in rpc: %s", err))
	}
	rpc.B = token.B

	ord, err := p.parseOrd()
	if err != nil {
		return forwardErr(err)
	}
	rpc.Ord = ord

	if token, err = p.expect(TokIden); err != nil {
		return forwardErr(err)
	}
	rpc.Name = token.Value

	if _, err = p.expect(TokLParen); err != nil {
		return forwardErr(err)
	}

	typ, err := p.parseType("")
	if err != nil {
		return forwardErr(err)
	}
	rpc.Arg = typ

	if err = p.expectChain(TokRParen, TokReturns, TokLParen); err != nil {
		return forwardErr(err)
	}

	typ, err = p.parseType("")
	if err != nil {
		return forwardErr(err)
	}
	rpc.Ret = typ

	token, err = p.expect(TokRParen)
	if err != nil {
		return forwardErr(err)
	}
	rpc.E = token.E

	return rpc
}
