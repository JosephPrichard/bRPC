package internal

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Parser struct {
	tokens    chan Token
	lastToken Token
	hasToken  bool
	asts      []Ast
	errs      []error
}

func newParser(tokens chan Token) Parser {
	return Parser{tokens: tokens}
}

func runParser(input string) ([]Ast, []error) {
	lex := newLexer(input)
	go lex.run()

	p := newParser(lex.tokens)
	p.parse()

	return p.asts, p.errs
}

func (p *Parser) read() Token {
	var token Token
	if p.hasToken {
		token = p.lastToken
		p.lastToken = Token{}
	} else {
		token = <-p.tokens
	}
	p.hasToken = false
	return token
}

func (p *Parser) consume() {
	if p.hasToken {
		p.lastToken = Token{}
	} else {
		<-p.tokens
	}
	p.hasToken = false
}

func (p *Parser) peek() Token {
	if p.hasToken {
		return p.lastToken
	} else {
		p.lastToken = <-p.tokens
		p.hasToken = true
		return p.lastToken
	}
}

func (p *Parser) expect(expected TokType) (Token, error) {
	token := p.read()
	if expected == token.t {
		return token, nil
	}
	return Token{}, parseErr(token, expected)
}

func (p *Parser) expectChain(chain ...TokType) error {
	for _, expected := range chain {
		if _, err := p.expect(expected); err != nil {
			return err
		}
	}
	return nil
}

var Eof = errors.New("reached end of token stream")

func (p *Parser) parse() {
	for {
		root, err := p.parseRoot()
		if errors.Is(err, Eof) {
			break
		}
		if err != nil {
			p.errs = append(p.errs, err)
			continue
		}
		if root == nil {
			panic(fmt.Sprintf("assertion error: parsed root ast should never be nil"))
		}
		p.asts = append(p.asts, root)
	}
}

func (p *Parser) parseRoot() (Ast, error) {
	token := p.read()

	switch token.t {
	case TokEof:
		return nil, Eof
	case TokMessage:
		return p.parseMessage()
	case TokService:
		return p.parseService()
	case TokIden:
		return p.parseProperty(token.value)
	default:
		return nil, parseErr(token, TokMessage, TokService)
	}
}

func (p *Parser) parseProperty(name string) (Ast, error) {
	var property PropertyAst
	property.Name = name

	if _, err := p.expect(TokEqual); err != nil {
		property.Error(err)
		return &property, nil
	}

	token, err := p.expect(TokString)
	if err != nil {
		property.Error(err)
		return &property, nil
	}
	property.Value = token.value

	if _, err := p.expect(TokTerminal); err != nil {
		property.Error(err)
		return &property, nil
	}

	return &property, nil
}

func (p *Parser) parseObject(name string) (Ast, error) {
	var ast Ast

	token := p.read()
	switch token.t {
	case TokStruct:
		ast = p.parseStruct(name)
	case TokEnum:
		ast = p.parseEnum(name)
	case TokUnion:
		ast = p.parseUnion(name)
	default:
		return nil, parseErr(token, TokStruct, TokUnion, TokEnum)
	}

	return ast, nil
}

func (p *Parser) parseMessage() (Ast, error) {
	// invariant: assume that 'message' token has been consumed
	token, err := p.expect(TokIden)
	if err != nil {
		return nil, err
	}
	name := token.value

	return p.parseObject(name)
}

func (p *Parser) parseStruct(name string) Ast {
	var strct StructAst
	strct.Name = name

	if _, err := p.expect(TokLBrace); err != nil {
		strct.Error(err)
		return &strct
	}

	for {
		token := p.peek()
		switch token.t {
		case TokOptional, TokRequired:
			field := p.parseField()
			strct.Fields = append(strct.Fields, field)
		case TokMessage:
			p.consume()
			message, err := p.parseMessage()
			if err != nil {
				strct.Error(err)
				continue
			}
			strct.LocalDefs = append(strct.LocalDefs, message)
		case TokRBrace:
			p.consume()
			return &strct
		default:
			p.consume()
			strct.Error(parseErr(token, TokOptional, TokRequired, TokMessage, TokRBrace))
			return &strct
		}
	}
}

func (p *Parser) parseOption() OptionAst {
	var option OptionAst

	ord, err := p.parseOrd()
	if err != nil {
		option.Error(err)
		return option
	}
	option.Ord = ord

	typ, err := p.parseType()
	if err != nil {
		option.Error(err)
		return option
	}
	option.Type = typ

	if _, err := p.expect(TokTerminal); err != nil {
		option.Error(err)
		return option
	}

	return option
}

func (p *Parser) parseUnion(name string) Ast {
	var union UnionAst
	union.Name = name

	if _, err := p.expect(TokLBrace); err != nil {
		union.Error(err)
		return &union
	}

	for {
		token := p.peek()
		switch token.t {
		case TokOrd:
			option := p.parseOption()
			union.Options = append(union.Options, option)
		case TokMessage:
			p.consume()
			message, err := p.parseMessage()
			if err != nil {
				union.Error(err)
				continue
			}
			union.LocalDefs = append(union.LocalDefs, message)
		case TokRBrace:
			p.consume()
			return &union
		default:
			p.consume()
			union.Error(parseErr(token, TokOrd, TokMessage, TokRBrace))
			return &union
		}

	}
}

func (p *Parser) parseNumeric() (int64, error) {
	token, err := p.expect(TokInteger)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseInt(token.value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%v: %s is not a valid integer", err, token.String())
	}
	return value, nil
}

func (p *Parser) parseCase() (EnumCase, error) {
	ord, err := p.parseOrd()
	if err != nil {
		return EnumCase{}, err
	}

	var ec EnumCase
	ec.Ord = ord

	token, err := p.expect(TokIden)
	if err != nil {
		return ec, err
	}
	ec.Name = token.value

	if _, err := p.expect(TokTerminal); err != nil {
		return ec, err
	}

	return ec, nil
}

func (p *Parser) parseEnum(name string) Ast {
	var enum EnumAst
	enum.Name = name

	if _, err := p.expect(TokLBrace); err != nil {
		enum.Error(err)
		return &enum
	}
	for {
		token := p.peek()
		switch token.t {
		case TokOrd:
			ec, err := p.parseCase()
			if err != nil {
				enum.Error(err)
				continue
			}
			enum.Cases = append(enum.Cases, ec)
		case TokMessage:
			p.consume()
			message, err := p.parseMessage()
			if err != nil {
				enum.Error(err)
				continue
			}
			enum.LocalDefs = append(enum.LocalDefs, message)
		case TokRBrace:
			p.consume()
			return &enum
		default:
			p.consume()
			enum.Error(parseErr(token, TokOrd, TokMessage, TokRBrace))
			return &enum
		}
	}
}

func (p *Parser) ParseArrayPrefix() (uint64, error) {
	if _, err := p.expect(TokLBrack); err != nil {
		return 0, err
	}
	token := p.read()
	switch token.t {
	case TokInteger:
		if _, err := p.expect(TokRBrack); err != nil {
			return 0, err
		}
		size, err := strconv.ParseUint(token.value, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%v: %s is not a valid array size", err, token.String())
		}
		return size, nil
	case TokRBrack:
		return 0, nil
	default:
		return 0, parseErr(token, TokInteger, TokRBrack)
	}
}

func (p *Parser) parseType() (Ast, error) {
	var root Ast
	var prevArray *TypeArrayAst

	appendAst := func(ast Ast) {
		if root == nil {
			root = ast
		}
		if prevArray != nil {
			prevArray.Type = ast
		}
		typeArray, ok := ast.(*TypeArrayAst)
		if ok {
			prevArray = typeArray
		}
	}

	for keepParsing := true; keepParsing; {
		token := p.peek()
		switch token.t {
		case TokIden:
			p.consume()
			ref := &TypeRefAst{Name: token.value}
			appendAst(ref)
			keepParsing = false
		case TokLBrack:
			size, err := p.ParseArrayPrefix()
			if err != nil {
				return nil, err
			}
			array := &TypeArrayAst{Size: size}
			appendAst(array)
		case TokStruct, TokUnion, TokEnum:
			object, err := p.parseObject("")
			if err != nil {
				return nil, err
			}
			appendAst(object)
			keepParsing = false
		default:
			p.consume()
			return nil, parseErr(token, TokIden, TokLBrack, TokStruct, TokUnion, TokEnum)
		}
	}

	if root == nil {
		panic("assertion failed: root is nil after parsing a <type>")
	}
	return root, nil
}

func (p *Parser) parseOrd() (uint64, error) {
	token, err := p.expect(TokOrd)
	if err != nil {
		return 0, err
	}
	if len(token.value) < 2 {
		panic(fmt.Errorf("assertion failed: an ord token should have at least 2 characters"))
	}
	ord, err := strconv.ParseUint(token.value[1:], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%v: %s is not a valid ord", err, token.String())
	}
	return ord, nil
}

func (p *Parser) parseField() FieldAst {
	var token Token
	var err error
	var ord uint64
	var field FieldAst

	token = p.read()
	switch token.t {
	case TokRequired:
		field.Modifier = Required
	case TokOptional:
		field.Modifier = Optional
	default:
		field.Error(parseErr(token, TokRequired, TokOptional))
		return field
	}

	if token, err = p.expect(TokIden); err != nil {
		field.Error(err)
		return field
	}
	field.Name = token.value

	if ord, err = p.parseOrd(); err != nil {
		field.Error(err)
		return field
	}
	field.Ord = ord

	typ, err := p.parseType()
	if err != nil {
		field.Error(err)
		return field
	}
	field.Type = typ

	if token, err = p.expect(TokTerminal); err != nil {
		field.Error(err)
		return field
	}

	return field
}

func (p *Parser) parseService() (Ast, error) {
	var svc ServiceAst

	token, err := p.expect(TokIden)
	if err != nil {
		return nil, err
	}
	svc.Name = token.value
	if _, err := p.expect(TokLBrace); err != nil {
		return nil, err
	}

	for {
		token = p.read()
		switch token.t {
		case TokRpc:
			rpc := p.parseRpc()
			svc.Procedures = append(svc.Procedures, rpc)
		case TokMessage:
			message, err := p.parseMessage()
			if err != nil {
				svc.Error(err)
				continue
			}
			svc.LocalDefs = append(svc.LocalDefs, message)
		case TokRBrace:
			return &svc, nil
		default:
			svc.Error(parseErr(token, TokRpc, TokMessage, TokRBrace))
			return &svc, nil
		}
	}
}

func (p *Parser) parseRpc() RpcAst {
	var token Token
	var err error
	var rpc RpcAst

	ord, err := p.parseOrd()
	if err != nil {
		rpc.Error(err)
		return rpc
	}
	rpc.Ord = ord

	if token, err = p.expect(TokIden); err != nil {
		rpc.Error(err)
		return rpc
	}
	rpc.Name = token.value

	if _, err = p.expect(TokLParen); err != nil {
		rpc.Error(err)
		return rpc
	}

	typ, err := p.parseType()
	if err != nil {
		rpc.Error(err)
		return rpc
	}
	rpc.Arg = typ

	if err = p.expectChain(TokRParen, TokReturns, TokLParen); err != nil {
		rpc.Error(err)
		return rpc
	}

	typ, err = p.parseType()
	if err != nil {
		rpc.Error(err)
		return rpc
	}
	rpc.Ret = typ

	if _, err = p.expect(TokRParen); err != nil {
		rpc.Error(err)
		return rpc
	}

	return rpc
}

func parseErr(actual Token, expected ...TokType) error {
	var sb strings.Builder

	sb.WriteString("expected ")
	for i, tok := range expected {
		sb.WriteString(tok.String())
		if i == len(expected)-2 {
			sb.WriteString(" or ")
		} else if i != len(expected)-1 {
			sb.WriteString(", ")
		}
	}

	sb.WriteString(" but got ")
	sb.WriteString(actual.String())
	return errors.New(sb.String())
}
