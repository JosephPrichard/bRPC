package internal

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type Parser struct {
	tokens    []Token
	curr      int
	nodes     []Node
	errs      *[]error
	hasEofErr bool // stores whether an error has been emitted after token stream has reached eof
}

func makeParser(tokens []Token, errs *[]error) Parser {
	return Parser{tokens: tokens, hasEofErr: false, errs: errs}
}

func runParser(program string, errs *[]error) []Node {
	lex := makeLexer(program)
	lex.run()

	p := makeParser(lex.tokens, errs)
	p.parse()

	return p.nodes
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
var StopTokens = []TokKind{TokLBrace, TokRBrace, TokService, TokRpc, TokRequired, TokOptional, TokDeprecated, TokMessage, TokStruct, TokUnion, TokEnum}

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
		*p.errs = append(*p.errs, err)
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
			p.nodes = append(p.nodes, root)
		}
	}
}

func (p *Parser) parseRoot() (Node, error) {
	var node Node
	var err error

	token := p.peek()
	switch token.Kind {
	case TokEof:
		return nil, Eof
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

func (p *Parser) parseProperty() Node {
	var prop PropertyNode

	forwardErr := func(err ParserError) Node {
		prop.E = err.token().E
		prop.Poisoned = true
		err.addKind(PropertyNodeKind)
		p.skipUntilSentinel()
		p.emitError(err)
		return &prop
	}

	var token Token

	token, err := p.expect(TokIden)
	if err != nil {
		panic(fmt.Sprintf("assertion error: %s", err))
	}
	prop.B = token.B
	prop.Name = token.Value

	if _, err := p.expect(TokEqual); err != nil {
		return forwardErr(err)
	}

	str, err := p.parseString(&token)
	if err != nil {
		return forwardErr(err)
	}
	prop.E = token.E
	prop.Value = str

	return &prop
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

	return sb.String(), nil
}

func (p *Parser) parseImport() Node {
	var imp ImportNode

	forwardErr := func(err ParserError) Node {
		imp.E = err.token().E
		imp.Poisoned = true
		err.addKind(ImportNodeKind)
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

const DefaultMSize = 16

func (p *Parser) parseMessageSize(callKind NodeKind) (uint64, ParserError) {
	if token := p.peek(); token.Kind != TokLBrack {
		return DefaultMSize, nil // defaults when size is not provided - struct will never use this
	}
	p.eat()

	token, err := p.expect(TokInteger)
	if err != nil {
		return 0, err
	}
	size, intErr := strconv.ParseUint(token.Value, 10, 64)
	if intErr != nil {
		panic(fmt.Sprintf("assertion error: integer token is invalid: %v", intErr))
	}
	if _, err := p.expect(TokRBrack); err != nil {
		return 0, err
	}

	// if the next token is a struct, emit an error, but not return the error to caller, we wish to continue parsing
	if p.peek().Kind == TokStruct {
		p.emitError(makeKindErr(token, SizeErrKind).withKind(callKind))
	}
	return size, nil
}

func validateMsgName(name string) bool {
	for i, c := range name {
		if i == 0 && unicode.IsLower(c) {
			return false
		}
		if !unicode.IsLetter(c) && !unicode.IsNumber(c) {
			return false
		}
	}
	return true
}

func (p *Parser) parseMessage() (Node, ParserError) {
	var token Token
	var err ParserError

	kind := MessageNodeKind

	// invariant: assume that 'errKind' token has been consumed
	token, err = p.expect(TokIden)
	if err != nil {
		return nil, err.withKind(kind)
	}
	name := token.Value
	nameOk := validateMsgName(name)
	if !nameOk {
		p.emitError(makeKindErr(token, IdenErrKind).withKind(kind))
	}

	size, err := p.parseMessageSize(kind)
	if err != nil {
		return nil, err.withKind(kind)
	}

	var node Node

	token = p.peek()
	switch token.Kind {
	case TokStruct:
		node = p.parseStruct(name, nameOk)
	case TokEnum:
		node = p.parseEnum(name, nameOk, size)
	case TokUnion:
		node = p.parseUnion(name, nameOk, size)
	default:
		p.eat()
		err = makeExpectErr(token, TokTypeDef).withKind(kind)
	}

	return node, err
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

func (p *Parser) parseStruct(name string, nameOk bool) Node {
	strct := StructNode{Name: name}
	strct.Poisoned = !nameOk

	forwardErr := func(err ParserError) {
		strct.E = err.token().E
		strct.Poisoned = true
		err.addKind(StructNodeKind)
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
		case TokOptional, TokRequired, TokDeprecated:
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

func (p *Parser) parseOption() OptionNode {
	var option OptionNode

	forwardErr := func(err ParserError) OptionNode {
		option.E = err.token().E
		option.Poisoned = true
		err.addKind(OptionNodeKind)
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

	token, err = p.expect(TokIden)
	if err != nil {
		return forwardErr(err)
	}
	option.Type = TypeVal{Iden: token.Value}

	if firstToken, ok := p.eatWhile(TokSemicolon); !ok {
		return forwardErr(makeExpectErr(firstToken, TokSemicolon))
	}

	return option
}

func (p *Parser) parseUnion(name string, nameOk bool, size uint64) Node {
	union := UnionNode{Name: name, Size: size}
	union.Poisoned = !nameOk

	forwardErr := func(err ParserError) {
		union.E = err.token().E
		union.Poisoned = true
		err.addKind(UnionNodeKind)
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
		return 0, makeKindErr(token, NumErrKind)
	}
	return value, nil
}

func (p *Parser) parseCase() CaseNode {
	var ec CaseNode

	forwardErr := func(err ParserError) CaseNode {
		ec.Poisoned = true
		err.addKind(CaseNodeKind)
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

func (p *Parser) parseEnum(name string, nameOk bool, size uint64) Node {
	enum := EnumNode{Name: name, Size: size}
	enum.Poisoned = !nameOk

	forwardErr := func(err ParserError) {
		enum.E = err.token().E
		enum.Poisoned = true
		err.addKind(EnumNodeKind)
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

func (p *Parser) parseTypeArgs(token *Token) ([]TypeNode, ParserError) {
	var typeArgs []TypeNode

	if p.peek().Kind != TokLParen {
		return typeArgs, nil
	}
	p.eat()

	for {
		if p.peek().Kind == TokRParen {
			*token = p.next()
			return typeArgs, nil
		}
		typ, err := p.parseType()
		if err != nil {
			return nil, err
		}
		typeArgs = append(typeArgs, typ)
	}
}

func (p *Parser) parseType() (TypeNode, ParserError) {
	// each element of the array is a nested array index
	var array []uint64
	var arrTokenB Token

	forwardErr := func(err ParserError) (TypeNode, ParserError) {
		err.addKind(TypeNodeKind)
		// don't emit the error, caller will handle this
		return TypeNode{}, err
	}

	for {
		token := p.next()
		switch token.Kind {
		case TokLBrack:
			if arrTokenB.Kind == TokUnknown {
				// if begin token is unset, we know we're at the first array token
				arrTokenB = token
			}
			size, err := p.parseArraySize()
			if err != nil {
				return forwardErr(err)
			}
			array = append(array, size)
		case TokIden:
			iden := token.Value

			// select the beginning token depending on whether the type ref is an array or not
			var tokenB = token
			if arrTokenB.Kind == TokUnknown {
				tokenB = arrTokenB
			}
			tokenE := tokenB

			typeArgs, err := p.parseTypeArgs(&tokenE)
			if err != nil {
				return forwardErr(err)
			}
			node := TypeNode{
				TypeVal:   TypeVal{Iden: iden},
				Array:     array,
				TypeArgs:  typeArgs,
				Positions: Positions{B: tokenB.B, E: tokenE.E},
			}
			return node, nil
		default:
			return TypeNode{}, makeExpectErr(token, TokTypeRef)
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
		panic("assertion error: an ord should have at least 2 characters")
	}
	ord, strErr := strconv.ParseUint(token.Value[1:], 10, 64)
	if strErr != nil {
		panic(fmt.Sprintf("assertion error: ord token is invalid: %v", err))
	}
	return ord, nil
}

func (p *Parser) parseField() FieldNode {
	var field FieldNode

	forwardErr := func(err ParserError) FieldNode {
		field.E = err.token().E
		field.Poisoned = true
		err.addKind(FieldNodeKind)
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
	case TokDeprecated:
		field.Modifier = Deprecated
	default:
		return forwardErr(makeExpectErr(token, TokRequired, TokOptional, TokDeprecated))
	}

	if token, err = p.expect(TokIden); err != nil {
		return forwardErr(err)
	}
	field.Name = token.Value

	if ord, err = p.parseOrd(); err != nil {
		return forwardErr(err)
	}
	field.Ord = ord

	typ, err := p.parseType()
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

func (p *Parser) parseService() Node {
	var svc ServiceNode

	forwardErr := func(err ParserError) {
		svc.E = err.token().E
		svc.Poisoned = true
		err.addKind(ServiceNodeKind)
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

func (p *Parser) parseRpc() RpcNode {
	var rpc RpcNode

	forwardErr := func(err ParserError) RpcNode {
		rpc.E = err.token().E
		rpc.Poisoned = true
		err.addKind(RpcNodeKind)
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

	typ, err := p.parseType()
	if err != nil {
		return forwardErr(err)
	}
	rpc.Arg = typ

	if err = p.expectChain(TokRParen, TokReturns, TokLParen); err != nil {
		return forwardErr(err)
	}

	typ, err = p.parseType()
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
