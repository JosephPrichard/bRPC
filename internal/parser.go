package internal

import (
	"fmt"
	"slices"
	"strings"
)

type Parser struct {
	tokens    []Token
	curr      int
	nodes     []DefNode
	errs      []ParseError
	hasEofErr bool // stores whether an error has been emitted after token stream has reached eof
}

func makeParser(tokens []Token) Parser {
	return Parser{tokens: tokens, hasEofErr: false, errs: make([]ParseError, 0)}
}

func runParser(program string) ([]DefNode, []ParseError) {
	lex := makeLexer(program)
	lex.run()

	p := makeParser(lex.tokens)
	p.parse()

	return p.nodes, p.errs
}

func parseOrElse(program string) []DefNode {
	nodes, errs := runParser(program)

	if len(errs) > 0 {
		panic(fmt.Sprintf("%+v", errs))
	}

	return nodes
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

func (p *Parser) expect(expected TokKind) (Token, ParseError) {
	token := p.next()
	var err ParseError
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

func (p *Parser) expectChain(chain ...TokKind) ParseError {
	for _, expected := range chain {
		if _, err := p.expect(expected); err.isPresent() {
			return err
		}
	}
	return ParseError{}
}

func (p *Parser) skipUntil(eatTokens []TokKind, stopTokens []TokKind) {
	for {
		token := p.peek()
		if token.Kind == TokEof {
			return
		}
		matchIdx := slices.Index(eatTokens, token.Kind)
		if matchIdx != -1 {
			p.eatWhile(eatTokens[matchIdx])
			return
		}
		if slices.Index(stopTokens, token.Kind) != -1 {
			return
		}
		p.eat()
	}
}

// EatTokens sentinel tokens which are eaten during propagating an error
var EatTokens = []TokKind{TokSemicolon}

// StopTokens sentinel tokens which are stopped at during propagating an error
var StopTokens = []TokKind{TokLBrace, TokRBrace, TokService, TokRpc, TokRequired, TokOptional, TokDeprecated, TokMessage, TokStruct, TokUnion, TokEnum}

func (p *Parser) skipUntilSentinel() {
	p.skipUntil(EatTokens, StopTokens)
}

// StopFieldTokens sentinel tokens which are stopped at during propagating field-level errors
var StopFieldTokens = []TokKind{TokSemicolon}

func (p *Parser) skipUntilFieldSentinel() {
	p.skipUntil(nil, StopFieldTokens)
}

func (p *Parser) emitError(err ParseError) {
	if p.hasEofErr {
		return
	}
	// don't emit anymore errors if a single eof err has been reached
	p.errs = append(p.errs, err)
	p.hasEofErr = p.peek().Kind == TokEof
}

func (p *Parser) parse() {
	for {
		root, err := p.parseRoot()
		if err.errKind == EofErrKind {
			break
		}
		if err.isPresent() {
			p.emitError(err)
			p.skipUntilSentinel()
		}
		p.nodes = append(p.nodes, root)
	}
}

func (p *Parser) parseRoot() (DefNode, ParseError) {
	var node DefNode
	var err ParseError

	token := p.peek()
	switch token.Kind {
	case TokEof:
		return DefNode{}, makeEofErr()
	case TokMessage:
		p.eat()
		node, err = p.parseMessage()
	case TokService:
		node = p.parseService()
	case TokImport:
		node = p.parseImport()
	case TokIden:
		node = p.parseProperty()
	default:
		p.eat()
		err = makeExpectErr(token, TokMessage, TokService, TokImport, TokIden)
	}

	return node, err
}

func (p *Parser) parseProperty() DefNode {
	propertyNode := DefNode{Kind: PropertyNodeKind}

	propagate := func(err ParseError) DefNode {
		propertyNode.End = err.actualToken.End
		propertyNode.Poisoned = true
		err.addKind(PropertyNodeKind)
		p.skipUntilSentinel()
		p.emitError(err)
		return propertyNode
	}

	var token Token

	token, err := p.expect(TokIden)
	if err.isPresent() {
		panic(fmt.Sprintf("assertion error: %s", err))
	}
	propertyNode.Begin = token.Begin
	propertyNode.Iden = token.Str

	if _, err := p.expect(TokEqual); err.isPresent() {
		return propagate(err)
	}

	str, err := p.parseString(&token)
	if err.isPresent() {
		return propagate(err)
	}
	propertyNode.End = token.End
	propertyNode.StrValue = str

	return propertyNode
}

var escSeqTable = map[rune]rune{'\\': '\\', 'n': '\n', '\t': '\t', 'f': '\f', 'r': '\r', '"': '"'}

func (p *Parser) parseString(token *Token) (string, ParseError) {
	t, err := p.expect(TokString)
	if err.isPresent() {
		return "", err
	}
	*token = t

	if len(token.Str) < 2 {
		panic(fmt.Sprintf("assertion error: string must be at least length 2, was: %s", token.Str))
	}

	var sb strings.Builder

	isEscaped := false
	str := token.Str[1 : len(token.Str)-1]

	for _, ch := range str {
		if isEscaped {
			ch, ok := escSeqTable[ch]
			if !ok {
				return "", makeEscSeqErr(*token, ch)
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

	return sb.String(), ParseError{}
}

func (p *Parser) parseImport() DefNode {
	importNode := DefNode{Kind: ImportNodeKind}

	propagate := func(err ParseError) DefNode {
		importNode.End = err.actualToken.End
		importNode.Poisoned = true
		err.addKind(ImportNodeKind)
		p.skipUntilSentinel()
		p.emitError(err)
		return importNode
	}

	var token Token

	token, err := p.expect(TokImport)
	if err.isPresent() {
		panic(fmt.Sprintf("assertion error: %s", err))
	}
	importNode.Begin = token.Begin

	pathStr, err := p.parseString(&token)
	if err.isPresent() {
		return propagate(err)
	}
	importNode.End = token.End
	importNode.StrValue = pathStr

	return importNode
}

const DefaultMSize = 16

func (p *Parser) parseMessageSize(callKind NodeKind) (uint64, ParseError) {
	if token := p.peek(); token.Kind != TokLBrack {
		return DefaultMSize, ParseError{} // defaults when size is not provided - struct will never use this
	}
	p.eat()

	token, err := p.expect(TokInteger)
	if err.isPresent() {
		return 0, err
	}
	size := token.Int.Uint64()

	if _, err := p.expect(TokRBrack); err.isPresent() {
		return 0, err
	}

	// if the next token is a struct, emit an error, but not return the error to caller, we wish to continue parsing
	if p.peek().Kind == TokStruct {
		p.emitError(makeKindErr(token, SizeErrKind).withKind(callKind))
	}
	return size, ParseError{}
}

func (p *Parser) parseMessage() (DefNode, ParseError) {
	var token Token
	var err ParseError

	kind := MessageNodeKind

	// invariant: assume that 'errKind' token has been consumed
	token, err = p.expect(TokIden)
	if err.isPresent() {
		return DefNode{}, err.withKind(kind)
	}
	name := token.Str

	size, err := p.parseMessageSize(kind)
	if err.isPresent() {
		return DefNode{}, err.withKind(kind)
	}

	var node DefNode

	token = p.peek()
	switch token.Kind {
	case TokStruct:
		node = p.parseStruct(name)
	case TokEnum:
		node = p.parseEnum(name, size)
	case TokUnion:
		node = p.parseUnion(name, size)
	default:
		p.eat()
		err = makeExpectErr(token, TokTypeDef).withKind(kind)
	}

	return node, err
}

func (p *Parser) parseStruct(name string) DefNode {
	strct := DefNode{Kind: StructNodeKind, Iden: name}

	propagate := func(err ParseError) {
		strct.End = err.actualToken.End
		strct.Poisoned = true
		err.addKind(StructNodeKind)
		p.skipUntilSentinel()
		p.emitError(err)
	}

	token, err := p.expect(TokStruct)
	if err.isPresent() {
		panic(fmt.Sprintf("assertion error: in struct: %s", err))
	}
	strct.Begin = token.Begin

	typeParams, err := p.parseTypeParams()
	if err.isPresent() {
		propagate(err)
		return strct
	}
	strct.TypeParams = typeParams

	if _, err := p.expect(TokLBrace); err.isPresent() {
		propagate(err)
		return strct
	}

	for {
		token := p.next()
		switch token.Kind {
		case TokOptional, TokRequired, TokDeprecated:
			p.prev()
			field := p.parseField()
			strct.Members = append(strct.Members, field)
		case TokMessage:
			message, err := p.parseMessage()
			if err.isPresent() {
				propagate(err)
				continue
			}
			strct.LocalDefs = append(strct.LocalDefs, message)
		case TokRBrace:
			strct.End = token.End
			return strct
		default:
			propagate(makeExpectErr(token, TokField, TokMessage, TokRBrace))
			if token.Kind == TokEof {
				return strct
			}
		}
	}
}

func (p *Parser) parseField() MemberNode {
	field := MemberNode{}

	propagate := func(err ParseError) MemberNode {
		field.End = err.actualToken.End
		field.Poisoned = true
		err.addKind(FieldNodeKind)
		p.skipUntilSentinel()
		p.emitError(err)
		return field
	}

	var token Token
	var err ParseError

	token = p.next()
	field.Begin = token.Begin

	switch token.Kind {
	case TokRequired:
		field.Modifier = Required
	case TokOptional:
		field.Modifier = Optional
	case TokDeprecated:
		field.Modifier = Deprecated
	default:
		return propagate(makeExpectErr(token, TokRequired, TokOptional, TokDeprecated))
	}

	token, err = p.expect(TokIden)
	if err.isPresent() {
		return propagate(err)
	}
	field.Iden = token.Str

	tag, err := p.parseTag()
	if err.isPresent() {
		return propagate(err)
	}
	field.Tag = tag

	typeNode, err := p.parseType()
	if err.isPresent() {
		return propagate(err)
	}
	field.LeftType = typeNode

	token = p.peek()
	switch token.Kind {
	case TokEqual:
		field.DefaultValue = p.parseValue()
	case TokSemicolon: // skip parsing if default value is not provided
	default:
		return propagate(makeExpectErr(token, TokEqual, TokSemicolon))
	}

	firstToken, ok := p.eatWhile(TokSemicolon)
	if !ok {
		return propagate(makeExpectErr(firstToken, TokSemicolon))
	}
	field.End = firstToken.End

	return field
}

func (p *Parser) parseValue() ValueNode {
	valueNode := ValueNode{}

	propagate := func(err ParseError) ValueNode {
		valueNode.End = err.actualToken.End
		valueNode.Poisoned = true
		err.addKind(ValueNodeKind)
		p.skipUntilFieldSentinel()
		p.emitError(err)
		return valueNode
	}

	var token Token

	token, err := p.expect(TokEqual)
	if err.isPresent() {
		panic(fmt.Sprintf("assertion error: in value node: %s", err))
	}
	valueNode.Begin = token.Begin

	token = p.peek()
	switch token.Kind {
	case TokInteger:
		valueNode.Kind, valueNode.Int = IntInstanceKind, token.Int
		p.eat()
	case TokFloat:
		valueNode.Kind, valueNode.Float64 = Float64InstanceKind, token.Float64
		p.eat()
	case TokString:
		str, err := p.parseString(&token)
		if err.isPresent() {
			return propagate(err)
		}
		valueNode.Kind, valueNode.Str = StringInstanceKind, str
	default:
		return propagate(makeExpectErr(token, TokInteger, TokFloat, TokString))
	}
	valueNode.End = token.End

	return valueNode
}

func (p *Parser) parseUnion(name string, size uint64) DefNode {
	union := DefNode{Kind: UnionNodeKind, Iden: name, Size: size}

	propagate := func(err ParseError) {
		union.End = err.actualToken.End
		union.Poisoned = true
		err.addKind(UnionNodeKind)
		p.skipUntilSentinel()
		p.emitError(err)
	}

	token, err := p.expect(TokUnion)
	if err.isPresent() {
		panic(fmt.Sprintf("assertion error: in union: %s", err))
	}
	union.Begin = token.Begin

	typeParams, err := p.parseTypeParams()
	if err.isPresent() {
		propagate(err)
		return union
	}
	union.TypeParams = typeParams

	if _, err := p.expect(TokLBrace); err.isPresent() {
		propagate(err)
		return union
	}

	for {
		token := p.next()
		switch token.Kind {
		case TokIden:
			p.prev()
			option := p.parseOption()
			union.Members = append(union.Members, option)
		case TokMessage:
			message, err := p.parseMessage()
			if err.isPresent() {
				propagate(err)
				continue
			}
			union.LocalDefs = append(union.LocalDefs, message)
		case TokRBrace:
			union.End = token.End
			return union
		default:
			propagate(makeExpectErr(token, TokOption, TokMessage, TokRBrace))
			if token.Kind == TokEof {
				return union
			}
		}
	}
}

func (p *Parser) parseOption() MemberNode {
	option := MemberNode{}

	propagate := func(err ParseError) MemberNode {
		option.End = err.actualToken.End
		option.Poisoned = true
		err.addKind(OptionNodeKind)
		p.skipUntilSentinel()
		p.emitError(err)
		return option
	}

	var token Token

	token, err := p.expect(TokIden)
	if err.isPresent() {
		return propagate(err)
	}
	option.Iden = token.Str

	ord, err := p.parseTagWithToken(&token)
	if err.isPresent() {
		return propagate(err)
	}
	option.Begin = token.Begin
	option.Tag = ord

	typeNode, err := p.parseType()
	if err.isPresent() {
		return propagate(err)
	}
	option.LeftType = typeNode

	firstToken, ok := p.eatWhile(TokSemicolon)
	if !ok {
		return propagate(makeExpectErr(firstToken, TokSemicolon))
	}
	option.End = firstToken.End

	return option
}

func (p *Parser) parseEnum(name string, size uint64) DefNode {
	enum := DefNode{Kind: EnumNodeKind, Iden: name, Size: size}

	propagate := func(err ParseError) {
		enum.End = err.actualToken.End
		enum.Poisoned = true
		err.addKind(EnumNodeKind)
		p.skipUntilSentinel()
		p.emitError(err)
	}

	token, err := p.expect(TokEnum)
	if err.isPresent() {
		panic(fmt.Sprintf("assertion error: in enum: %s", err))
	}
	enum.Begin = token.Begin

	if _, err := p.expect(TokLBrace); err.isPresent() {
		propagate(err)
		return enum
	}
	for {
		token := p.next()
		switch token.Kind {
		case TokTag:
			p.prev()
			enumCase := p.parseCase()
			enum.Members = append(enum.Members, enumCase)
		case TokRBrace:
			enum.End = token.End
			return enum
		default:
			propagate(makeExpectErr(token, TokCase, TokRBrace))
			if token.Kind == TokEof {
				return enum
			}
		}
	}
}

func (p *Parser) parseCase() MemberNode {
	enumCase := MemberNode{}

	propagate := func(err ParseError) MemberNode {
		enumCase.End = err.actualToken.End
		enumCase.Poisoned = true
		err.addKind(CaseNodeKind)
		p.skipUntilSentinel()
		p.emitError(err)
		return enumCase
	}

	var token Token

	tag, err := p.parseTagWithToken(&token)
	if err.isPresent() {
		return propagate(err)
	}
	enumCase.Tag = tag
	enumCase.Begin = token.Begin

	token, err = p.expect(TokIden)
	if err.isPresent() {
		return propagate(err)
	}
	enumCase.Iden = token.Str

	firstToken, ok := p.eatWhile(TokSemicolon)
	if !ok {
		return propagate(makeExpectErr(firstToken, TokSemicolon))
	}
	enumCase.End = firstToken.End

	return enumCase
}

func (p *Parser) parseArraySize() (uint64, ParseError) {
	token := p.next()
	switch token.Kind {
	case TokInteger:
		size := token.Int.Uint64()
		if _, err := p.expect(TokRBrack); err.isPresent() {
			return 0, err
		}
		return size, ParseError{}
	case TokRBrack:
		return 0, ParseError{}
	default:
		return 0, makeExpectErr(token, TokInteger, TokRBrack)
	}
}

func (p *Parser) parseTypeParams() ([]string, ParseError) {
	var typeParams []string

	if p.peek().Kind != TokLParen {
		return typeParams, ParseError{}
	}
	p.eat()

	for {
		if p.peek().Kind == TokRParen {
			break
		}
		token, err := p.expect(TokIden)
		if err.isPresent() {
			return nil, err
		}
		typeParams = append(typeParams, token.Str)

		if p.peek().Kind != TokComma {
			break
		}
		p.eat()
	}

	if _, err := p.expect(TokRParen); err.isPresent() {
		return nil, err
	}
	return typeParams, ParseError{}
}

func (p *Parser) parseTypeArgs(token *Token) ([]TypeNode, ParseError) {
	var typeArgs []TypeNode

	if p.peek().Kind != TokLParen {
		return typeArgs, ParseError{}
	}
	p.eat()

	for {
		if p.peek().Kind == TokRParen {
			break
		}
		typeNode, err := p.parseType()
		if err.isPresent() {
			return nil, err
		}
		typeArgs = append(typeArgs, typeNode)

		if p.peek().Kind != TokComma {
			break
		}
		p.eat()
	}

	if _, err := p.expect(TokRParen); err.isPresent() {
		*token = p.next()
		return nil, err
	}

	return typeArgs, ParseError{}
}

func (p *Parser) parseType() (TypeNode, ParseError) {
	// each element of the array is a nested array index
	var array []uint64
	var arrTokenBegin Token

	propagate := func(err ParseError) (TypeNode, ParseError) {
		err.addKind(TypeNodeKind)
		// don't emit the error, caller will handle this
		return TypeNode{}, err
	}

	for {
		token := p.next()
		switch token.Kind {
		case TokLBrack:
			if arrTokenBegin.Kind == TokUnknown {
				// if begin token is unset, we know we're at the first array token
				arrTokenBegin = token
			}
			size, err := p.parseArraySize()
			if err.isPresent() {
				return propagate(err)
			}
			array = append(array, size)
		case TokIden:
			name := token.Str

			// select the beginning token depending on whether the type ref is an array or not
			var tokenBegin = token
			if arrTokenBegin.Kind != TokUnknown {
				tokenBegin = arrTokenBegin
			}
			tokenEnd := tokenBegin

			typeArgs, err := p.parseTypeArgs(&tokenEnd)
			if err.isPresent() {
				return propagate(err)
			}
			node := TypeNode{
				Iden:      name,
				Array:     array,
				TypeArgs:  typeArgs,
				Positions: Positions{Begin: tokenBegin.Begin, End: tokenEnd.End},
			}
			return node, ParseError{}
		default:
			return TypeNode{}, makeExpectErr(token, TokTypeRef)
		}
	}
}

func (p *Parser) parseTag() (uint64, ParseError) {
	var token Token
	return p.parseTagWithToken(&token)
}

// parseTagWithToken 'writes back' the token it reads to the caller for further processing
func (p *Parser) parseTagWithToken(token *Token) (uint64, ParseError) {
	t, err := p.expect(TokTag)
	if err.isPresent() {
		return 0, err
	}
	*token = t
	if len(token.Str) < 2 {
		panic("assertion error: a tag should have at least 2 characters")
	}
	tag := token.Int.Uint64()
	return tag, ParseError{}
}

func (p *Parser) parseService() DefNode {
	svc := DefNode{Kind: ServiceNodeKind}

	propagate := func(err ParseError) {
		svc.End = err.actualToken.End
		svc.Poisoned = true
		err.addKind(ServiceNodeKind)
		p.skipUntilSentinel()
		p.emitError(err)
	}

	token, err := p.expect(TokService)
	if err.isPresent() {
		panic(fmt.Sprintf("assertion error: in service: %s", err))
	}

	token, err = p.expect(TokIden)
	if err.isPresent() {
		propagate(err)
		return svc
	}
	svc.Iden = token.Str
	if _, err := p.expect(TokLBrace); err.isPresent() {
		propagate(err)
		return svc
	}

	for {
		token = p.next()
		switch token.Kind {
		case TokRpc:
			p.prev()
			rpc := p.parseRpc()
			svc.Members = append(svc.Members, rpc)
		case TokMessage:
			message, err := p.parseMessage()
			if err.isPresent() {
				propagate(err)
				continue
			}
			svc.LocalDefs = append(svc.LocalDefs, message)
		case TokRBrace:
			return svc
		default:
			propagate(makeExpectErr(token, TokRpc, TokMessage, TokRBrace))
			if token.Kind == TokEof {
				return svc
			}
		}
	}
}

func (p *Parser) parseRpc() MemberNode {
	rpc := MemberNode{}

	propagate := func(err ParseError) MemberNode {
		rpc.End = err.actualToken.End
		rpc.Poisoned = true
		err.addKind(RpcNodeKind)
		p.skipUntilSentinel()
		p.emitError(err)
		return rpc
	}

	token, err := p.expect(TokRpc)
	if err.isPresent() {
		panic(fmt.Sprintf("assertion error: in rpc: %s", err))
	}
	rpc.Begin = token.Begin

	tag, err := p.parseTag()
	if err.isPresent() {
		return propagate(err)
	}
	rpc.Tag = tag

	if token, err = p.expect(TokIden); err.isPresent() {
		return propagate(err)
	}
	rpc.Iden = token.Str

	if _, err = p.expect(TokLParen); err.isPresent() {
		return propagate(err)
	}

	typeNode, err := p.parseType()
	if err.isPresent() {
		return propagate(err)
	}
	rpc.LeftType = typeNode

	if err = p.expectChain(TokRParen, TokReturns, TokLParen); err.isPresent() {
		return propagate(err)
	}

	typeNode, err = p.parseType()
	if err.isPresent() {
		return propagate(err)
	}
	rpc.RightType = typeNode

	token, err = p.expect(TokRParen)
	if err.isPresent() {
		return propagate(err)
	}
	rpc.End = token.End

	return rpc
}
