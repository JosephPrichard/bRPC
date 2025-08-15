package internal

import (
	"errors"
	"fmt"
	"path/filepath"
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

func newParser(tokens []Token) Parser {
	return Parser{tokens: tokens, hasEofErr: false}
}

func runParser(input string) ([]Ast, []error) {
	lex := newLexer(input)
	lex.run()

	p := newParser(lex.tokens)
	p.parse()

	return p.asts, p.errs
}

func (p *Parser) next() Token {
	token := p.tokens[p.curr]
	if token.t != TokEof {
		p.curr++
	}
	return token
}

func (p *Parser) eat() {
	if p.peek().t != TokEof {
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

func (p *Parser) expect(expected TokType) (Token, ParseError) {
	token := p.next()
	if expected == token.t {
		return token, nil
	}
	return Token{}, makeTokenErr(token, expected)
}

func (p *Parser) eatWhile(expected TokType) (Token, bool) {
	firstToken := p.peek()
	ok := false
	for p.peek().t == expected {
		if !ok {
			ok = true
		}
		p.eat()
	}
	return firstToken, ok
}

func (p *Parser) expectChain(chain ...TokType) ParseError {
	for _, expected := range chain {
		if _, err := p.expect(expected); err != nil {
			return err
		}
	}
	return nil
}

// EatTokens sentinel tokens which are eaten during forwarding
var EatTokens = []TokType{TokSemicolon}

// StopTokens sentinel tokens which are stopped at during forwarding
var StopTokens = []TokType{TokLBrace, TokRBrace, TokRequired, TokOptional, TokMessage, TokStruct, TokUnion, TokEnum}

func (p *Parser) forwardSentinel() {
	for {
		token := p.peek()
		if token.t == TokEof {
			return
		}
		matchIdx := slices.Index(EatTokens, token.t)
		if matchIdx != -1 {
			p.eatWhile(EatTokens[matchIdx])
			return
		}
		if slices.Index(StopTokens, token.t) != -1 {
			return
		}
		p.eat()
	}
}

func (p *Parser) appendErr(err error) {
	if !p.hasEofErr {
		// don't emit anymore errors if a single err has been emitted after reaching eof
		p.errs = append(p.errs, err)
	}
	p.hasEofErr = p.peek().t == TokEof
}

var Eof = errors.New("reached end of token stream while parsing")

func (p *Parser) parse() {
	for {
		root, err := p.parseRoot()
		if errors.Is(err, Eof) {
			break
		}
		if err != nil {
			p.appendErr(err)
			p.forwardSentinel()
		}
		if root != nil {
			p.asts = append(p.asts, root)
		}
	}
}

func (p *Parser) parseRoot() (Ast, error) {
	token := p.next()
	switch token.t {
	case TokEof:
		return nil, Eof
	case TokMessage:
		return p.parseMessage()
	case TokService:
		return p.parseService()
	case TokImport:
		return p.parseImport()
	case TokIden:
		return p.parseProperty(token.value)
	default:
		return nil, makeTokenErr(token, TokMessage, TokService, TokImport, TokIden)
	}
}

func (p *Parser) parseProperty(name string) (Ast, error) {
	var property PropertyAst
	property.Name = name

	handleErr := func(err ParseError) (Ast, ParseError) {
		err.addKind(PropertyAstKind)
		p.forwardSentinel()
		p.appendErr(err)
		return &property, nil
	}

	if _, err := p.expect(TokEqual); err != nil {
		return handleErr(err)
	}

	var token Token
	str, err := p.parseString(&token)
	if err != nil {
		return handleErr(err)
	}
	property.Value = str

	return &property, nil
}

var EscSeqTable = map[rune]rune{
	'\\': '\\',
	'n':  '\n',
	't':  '\t',
	'f':  '\f',
	'r':  '\r',
	'"':  '"',
}

func (p *Parser) parseString(token *Token) (string, ParseError) {
	t, err := p.expect(TokString)
	if err != nil {
		return "", err
	}
	*token = t

	if len(token.value) < 2 {
		panic(fmt.Sprintf("assertion error: import path string must be at least length 2, was: %s", token.value))
	}

	var sb strings.Builder

	isEscaped := false
	str := token.value[1 : len(token.value)-1]

	for _, ch := range str {
		if isEscaped {
			ch, ok := EscSeqTable[ch]
			if !ok {
				return "", makeParseErr(*token, fmt.Sprintf("invalid escape sequence: '/%c'", ch))
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

const BrpcExt = ".brpc"

func (p *Parser) parseImport() (Ast, ParseError) {
	var imp ImportAst

	handleErr := func(err ParseError) (Ast, ParseError) {
		err.addKind(ImportAstKind)
		p.forwardSentinel()
		p.appendErr(err)
		return &imp, nil
	}

	var token Token
	pathStr, err := p.parseString(&token)
	if err != nil {
		return handleErr(err)
	}
	imp.Path = pathStr

	ext := filepath.Ext(pathStr)
	if ext != BrpcExt {
		return handleErr(makeParseErr(token, fmt.Sprintf("import path must refer to a brpc file ending with extension: %s", BrpcExt)))
	}

	return &imp, nil
}

func (p *Parser) parseMessage() (Ast, ParseError) {
	// invariant: assume that 'message' token has been consumed
	token, err := p.expect(TokIden)
	if err != nil {
		return nil, err
	}
	name := token.value

	return p.parseType(name)
}

func (p *Parser) parseTypeArgs() ([]string, ParseError) {
	var typeArgs []string

	token := p.next()
	switch token.t {
	case TokLBrace:
	case TokLParen:
		for parsing := true; parsing; {
			token := p.next()
			switch token.t {
			case TokRParen:
				parsing = false
			case TokIden:
				typeArgs = append(typeArgs, token.value)
			default:
				return nil, makeTokenErr(token, TokRParen, TokIden)
			}
		}
		if _, err := p.expect(TokLBrace); err != nil {
			return nil, err
		}
	default:
		return nil, makeTokenErr(token, TokLBrace, TokLParen)
	}

	return typeArgs, nil
}

func (p *Parser) parseStruct(name string) Ast {
	var strct StructAst
	strct.Name = name

	handleErr := func(err ParseError) {
		err.addKind(StructAstKind)
		p.forwardSentinel()
		p.appendErr(err)
	}

	typeArgs, err := p.parseTypeArgs()
	if err != nil {
		handleErr(err)
		return &strct
	}
	strct.TypeArgs = typeArgs

	for {
		token := p.next()
		switch token.t {
		case TokOptional, TokRequired:
			p.prev()
			field := p.parseField()
			strct.Fields = append(strct.Fields, field)
		case TokMessage:
			message, err := p.parseMessage()
			if err != nil {
				handleErr(err)
				continue
			}
			strct.LocalDefs = append(strct.LocalDefs, message)
		case TokRBrace:
			return &strct
		default:
			handleErr(makeTokenErr(token, TokOptional, TokRequired, TokMessage, TokRBrace))
			if token.t == TokEof {
				return &strct
			}
		}
	}
}

func (p *Parser) parseOption() OptionAst {
	var option OptionAst

	handleErr := func(err ParseError) OptionAst {
		err.addKind(OptionAstKind)
		p.forwardSentinel()
		p.appendErr(err)
		return option
	}

	ord, err := p.parseOrd()
	if err != nil {
		return handleErr(err)
	}
	option.Ord = ord

	typ, err := p.parseType("")
	if err != nil {
		return handleErr(err)
	}
	option.Type = typ

	if firstToken, ok := p.eatWhile(TokSemicolon); !ok {
		return handleErr(makeTokenErr(firstToken, TokSemicolon))
	}

	return option
}

func (p *Parser) parseUnion(name string) Ast {
	var union UnionAst
	union.Name = name

	handleErr := func(err ParseError) {
		err.addKind(UnionAstKind)
		p.forwardSentinel()
		p.appendErr(err)
	}

	typeArgs, err := p.parseTypeArgs()
	if err != nil {
		handleErr(err)
		return &union
	}
	union.TypeArgs = typeArgs

	for {
		token := p.next()
		switch token.t {
		case TokOrd:
			p.prev()
			option := p.parseOption()
			union.Options = append(union.Options, option)
		case TokMessage:
			message, err := p.parseMessage()
			if err != nil {
				handleErr(err)
				continue
			}
			union.LocalDefs = append(union.LocalDefs, message)
		case TokRBrace:
			return &union
		default:
			handleErr(makeTokenErr(token, TokOrd, TokMessage, TokRBrace))
			if token.t == TokEof {
				return &union
			}
		}

	}
}

func (p *Parser) parseNumeric() (int64, ParseError) {
	token, err := p.expect(TokInteger)
	if err != nil {
		return 0, err
	}
	value, convErr := strconv.ParseInt(token.value, 10, 64)
	if convErr != nil {
		return 0, makeParseErr(token, fmt.Sprintf("%v: %s is not a valid integer", err, token.String()))
	}
	return value, nil
}

func (p *Parser) parseCase() (EnumCase, ParseError) {
	var ec EnumCase

	ord, err := p.parseOrd()
	if err != nil {
		return ec, err
	}
	ec.Ord = ord

	token, err := p.expect(TokIden)
	if err != nil {
		return ec, err
	}
	ec.Name = token.value

	if firstToken, ok := p.eatWhile(TokSemicolon); !ok {
		return ec, makeTokenErr(firstToken, TokSemicolon)
	}

	return ec, nil
}

func (p *Parser) parseEnum(name string) Ast {
	var enum EnumAst
	enum.Name = name

	handleErr := func(err ParseError) Ast {
		err.addKind(EnumAstKind)
		p.forwardSentinel()
		p.appendErr(err)
		return &enum
	}

	if _, err := p.expect(TokLBrace); err != nil {
		return handleErr(err)
	}
	for {
		token := p.next()
		switch token.t {
		case TokOrd:
			p.prev()
			ec, err := p.parseCase()
			if err != nil {
				_ = handleErr(err)
				continue
			}
			enum.Cases = append(enum.Cases, ec)
		case TokRBrace:
			return &enum
		default:
			handleErr(makeTokenErr(token, TokOrd, TokRBrace))
			if token.t == TokEof {
				return &enum
			}
		}
	}
}

func (p *Parser) parseArrayPrefix() (uint64, ParseError) {
	token := p.next()
	switch token.t {
	case TokInteger:
		if _, err := p.expect(TokRBrack); err != nil {
			return 0, err
		}
		size, err := strconv.ParseUint(token.value, 10, 64)
		if err != nil {
			return 0, makeParseErr(token, fmt.Sprintf("%v: %s is not a valid arrPrefix size", err, token.String()))
		}
		return size, nil
	case TokRBrack:
		return 0, nil
	default:
		return 0, makeTokenErr(token, TokInteger, TokRBrack)
	}
}

func (p *Parser) parseTypeInputs() ([]Ast, ParseError) {
	var typeArgs []Ast

	if p.peek().t != TokLParen {
		return typeArgs, nil
	}
	p.eat()

	for {
		if p.peek().t == TokRParen {
			p.eat()
			return typeArgs, nil
		}
		typ, err := p.parseType("")
		if err != nil {
			return nil, err
		}
		typeArgs = append(typeArgs, typ)
	}
}

func (p *Parser) parseType(name string) (Ast, ParseError) {
	var array []uint64

	makeAst := func(typ Ast) Ast {
		if array != nil {
			return &ArrayAst{Type: typ, Size: array}
		}
		return typ
	}

	for {
		token := p.next()
		switch token.t {
		case TokLBrack:
			size, err := p.parseArrayPrefix()
			if err != nil {
				return nil, err
			}
			array = append(array, size)
		case TokIden:
			typArgs, err := p.parseTypeInputs()
			if err != nil {
				return nil, err
			}
			ast := makeAst(&TypeAst{Alias: name, Iden: token.value, TypeArgs: typArgs})
			return ast, nil
		case TokStruct:
			ast := makeAst(p.parseStruct(name))
			return ast, nil
		case TokUnion:
			ast := makeAst(p.parseUnion(name))
			return ast, nil
		case TokEnum:
			ast := makeAst(p.parseEnum(name))
			return ast, nil
		default:
			return nil, makeParseErr(token, "<type>")
		}
	}
}

func (p *Parser) parseOrd() (uint64, ParseError) {
	token, err := p.expect(TokOrd)
	if err != nil {
		return 0, err
	}
	if len(token.value) < 2 {
		panic(fmt.Errorf("assertion failed: an ord token should have at least 2 characters"))
	}
	ord, strErr := strconv.ParseUint(token.value[1:], 10, 64)
	if strErr != nil {
		return 0, makeParseErr(token, fmt.Sprintf("%v: %s is not a valid ord", err, token.String()))
	}
	return ord, nil
}

func (p *Parser) parseField() FieldAst {
	var token Token
	var err ParseError
	var ord uint64
	var field FieldAst

	handleErr := func(err ParseError) FieldAst {
		err.addKind(FieldAstKind)
		p.forwardSentinel()
		p.appendErr(err)
		return field
	}

	token = p.next()
	switch token.t {
	case TokRequired:
		field.Modifier = Required
	case TokOptional:
		field.Modifier = Optional
	default:
		return handleErr(makeTokenErr(token, TokRequired, TokOptional))
	}

	if token, err = p.expect(TokIden); err != nil {
		return handleErr(err)
	}
	field.Name = token.value

	if ord, err = p.parseOrd(); err != nil {
		return handleErr(err)
	}
	field.Ord = ord

	typ, err := p.parseType("")
	if err != nil {
		return handleErr(err)
	}
	field.Type = typ

	if firstToken, ok := p.eatWhile(TokSemicolon); !ok {
		return handleErr(makeTokenErr(firstToken, TokSemicolon))
	}

	return field
}

func (p *Parser) parseService() (Ast, ParseError) {
	var svc ServiceAst

	handleErr := func(err ParseError) {
		err.addKind(ServiceAstKind)
		p.forwardSentinel()
		p.appendErr(err)

	}

	token, err := p.expect(TokIden)
	if err != nil {
		handleErr(err)
		return &svc, nil
	}
	svc.Name = token.value
	if _, err := p.expect(TokLBrace); err != nil {
		handleErr(err)
		return &svc, nil
	}

	for {
		token = p.next()
		switch token.t {
		case TokRpc:
			rpc := p.parseRpc()
			svc.Procedures = append(svc.Procedures, rpc)
		case TokMessage:
			message, err := p.parseMessage()
			if err != nil {
				handleErr(err)
				continue
			}
			svc.LocalDefs = append(svc.LocalDefs, message)
		case TokRBrace:
			return &svc, nil
		default:
			handleErr(makeTokenErr(token, TokRpc, TokMessage, TokRBrace))
			if token.t == TokEof {
				return &svc, nil
			}
		}
	}
}

func (p *Parser) parseRpc() RpcAst {
	var token Token
	var err ParseError
	var rpc RpcAst

	handleErr := func(err ParseError) RpcAst {
		err.addKind(RpcAstKind)
		p.appendErr(err)
		return rpc
	}

	ord, err := p.parseOrd()
	if err != nil {
		return handleErr(err)
	}
	rpc.Ord = ord

	if token, err = p.expect(TokIden); err != nil {
		return handleErr(err)
	}
	rpc.Name = token.value

	if _, err = p.expect(TokLParen); err != nil {
		return handleErr(err)
	}

	typ, err := p.parseType("")
	if err != nil {
		return handleErr(err)
	}
	rpc.Arg = typ

	if err = p.expectChain(TokRParen, TokReturns, TokLParen); err != nil {
		return handleErr(err)
	}

	typ, err = p.parseType("")
	if err != nil {
		return handleErr(err)
	}
	rpc.Ret = typ

	if _, err = p.expect(TokRParen); err != nil {
		return handleErr(err)
	}

	return rpc
}
