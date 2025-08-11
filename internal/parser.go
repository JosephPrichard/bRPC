package internal

import (
	"errors"
	"fmt"
	"path/filepath"
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
	return Token{}, expectErr(token, expected)
}

func (p *Parser) consumeIf(expected TokType) bool {
	if token := p.peek(); token.t == expected {
		p.consume()
		return true
	}
	return false
}

func (p *Parser) expectChain(chain ...TokType) error {
	for _, expected := range chain {
		if _, err := p.expect(expected); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) forwardSentinel(until ...TokType) {
	for {
		token := p.peek()
		if token.t == TokEof {
			return
		}
		for _, expected := range until {
			if expected == token.t {
				return
			}
		}
		p.consume()
	}
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
			break
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
	case TokImport:
		return p.parseImport()
	case TokIden:
		return p.parseProperty(token.value)
	default:
		return nil, expectErr(token, TokMessage, TokService)
	}
}

func (p *Parser) parseProperty(name string) (Ast, error) {
	var property PropertyAst
	property.Name = name

	handleErr := func(err error) (Ast, error) {
		addAstKind(err, PropertyAstKind)
		p.errs = append(p.errs, err)
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

func (p *Parser) parseString(token *Token) (string, error) {
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
				return "", parseErr(*token, fmt.Sprintf("invalid escape sequence: '/%c'", ch))
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

func (p *Parser) parseImport() (Ast, error) {
	var imp ImportAst

	handleErr := func(err error) (Ast, error) {
		addAstKind(err, ImportAstKind)
		p.errs = append(p.errs, err)
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
		return handleErr(parseErr(token, fmt.Sprintf("import path must refer to a brpc file ending with extension: %s", BrpcExt)))
	}

	return &imp, nil
}

func (p *Parser) parseMessage() (Ast, error) {
	// invariant: assume that 'message' token has been consumed
	token, err := p.expect(TokIden)
	if err != nil {
		return nil, err
	}
	name := token.value

	return p.parseType(name)
}

func (p *Parser) parseTypeArgs() ([]string, error) {
	var typeArgs []string

	token := p.read()
	switch token.t {
	case TokLBrace:
	case TokLParen:
		for parsing := true; parsing; {
			token := p.read()
			switch token.t {
			case TokRParen:
				parsing = false
			case TokIden:
				typeArgs = append(typeArgs, token.value)
			default:
				return nil, expectErr(token, TokRParen, TokIden)
			}
		}
		if _, err := p.expect(TokLBrace); err != nil {
			return nil, err
		}
	default:
		return nil, expectErr(token, TokLBrace, TokLParen)
	}

	return typeArgs, nil
}

func (p *Parser) parseStruct(name string) Ast {
	var strct StructAst
	strct.Name = name

	handleErr := func(err error) Ast {
		addAstKind(err, StructAstKind)
		p.errs = append(p.errs, err)
		return &strct
	}

	typeArgs, err := p.parseTypeArgs()
	if err != nil {
		return handleErr(err)
	}
	strct.TypeArgs = typeArgs

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
				_ = handleErr(err)
				continue
			}
			strct.LocalDefs = append(strct.LocalDefs, message)
		case TokRBrace:
			p.consume()
			return &strct
		default:
			p.consume()
			return handleErr(expectErr(token, TokOptional, TokRequired, TokMessage, TokRBrace))
		}
	}
}

func (p *Parser) parseOption() OptionAst {
	var option OptionAst

	handleErr := func(err error) OptionAst {
		addAstKind(err, OptionAstKind)
		p.errs = append(p.errs, err)
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

	if _, err := p.expect(TokTerminal); err != nil {
		return handleErr(err)
	}

	return option
}

func (p *Parser) parseUnion(name string) Ast {
	var union UnionAst
	union.Name = name

	handleErr := func(err error) Ast {
		addAstKind(err, UnionAstKind)
		p.errs = append(p.errs, err)
		return &union
	}

	typeArgs, err := p.parseTypeArgs()
	if err != nil {
		return handleErr(err)
	}
	union.TypeArgs = typeArgs

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
				_ = handleErr(err)
				continue
			}
			union.LocalDefs = append(union.LocalDefs, message)
		case TokRBrace:
			p.consume()
			return &union
		default:
			p.consume()
			return handleErr(expectErr(token, TokOrd, TokMessage, TokRBrace))
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

	if _, err := p.expect(TokTerminal); err != nil {
		return ec, err
	}

	return ec, nil
}

func (p *Parser) parseEnum(name string) Ast {
	var enum EnumAst
	enum.Name = name

	handleErr := func(err error) Ast {
		addAstKind(err, EnumAstKind)
		p.errs = append(p.errs, err)
		return &enum
	}

	if _, err := p.expect(TokLBrace); err != nil {
		return handleErr(err)
	}
	for {
		token := p.peek()
		switch token.t {
		case TokOrd:
			ec, err := p.parseCase()
			if err != nil {
				_ = handleErr(err)
				continue
			}
			enum.Cases = append(enum.Cases, ec)
		case TokRBrace:
			p.consume()
			return &enum
		default:
			p.consume()
			return handleErr(expectErr(token, TokOrd, TokRBrace))
		}
	}
}

func (p *Parser) ParseArrayPrefix() (uint64, error) {
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
		return 0, expectErr(token, TokInteger, TokRBrack)
	}
}

func (p *Parser) parseTypeInputs() ([]Ast, error) {
	var typeArgs []Ast

	if p.peek().t != TokLParen {
		return typeArgs, nil
	}
	p.consume()

	for {
		if p.peek().t == TokRParen {
			p.consume()
			return typeArgs, nil
		}
		typ, err := p.parseType("")
		if err != nil {
			return nil, err
		}
		typeArgs = append(typeArgs, typ)
	}
}

func (p *Parser) parseType(name string) (Ast, error) {
	var root Ast
	var prevArray *ArrayAst

	appendAst := func(ast Ast) {
		if root == nil {
			root = ast
		}
		if prevArray != nil {
			prevArray.Type = ast
		}
		typeArray, ok := ast.(*ArrayAst)
		if ok {
			prevArray = typeArray
		}
	}

	for {
		token := p.read()
		switch token.t {
		case TokLBrack:
			size, err := p.ParseArrayPrefix()
			if err != nil {
				return nil, err
			}
			appendAst(&ArrayAst{Size: size})
		case TokIden:
			typArgs, err := p.parseTypeInputs()
			if err != nil {
				return nil, err
			}
			appendAst(&TypeAst{Alias: name, Value: token.value, TypeArgs: typArgs})
			return root, nil
		case TokStruct:
			appendAst(p.parseStruct(name))
			return root, nil
		case TokUnion:
			appendAst(p.parseUnion(name))
			return root, nil
		case TokEnum:
			appendAst(p.parseEnum(name))
			return root, nil
		default:
			return nil, expectErr(token, TokLBrack, TokIden, TokStruct, TokUnion, TokEnum)
		}
	}
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

	handleErr := func(err error) FieldAst {
		addAstKind(err, FieldAstKind)
		p.errs = append(p.errs, err)
		return field
	}

	token = p.read()
	switch token.t {
	case TokRequired:
		field.Modifier = Required
	case TokOptional:
		field.Modifier = Optional
	default:
		return handleErr(expectErr(token, TokRequired, TokOptional))
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

	if token, err = p.expect(TokTerminal); err != nil {
		return handleErr(err)
	}

	return field
}

func (p *Parser) parseService() (Ast, error) {
	var svc ServiceAst

	handleErr := func(err error) (Ast, error) {
		addAstKind(err, ServiceAstKind)
		p.errs = append(p.errs, err)
		return &svc, nil
	}

	token, err := p.expect(TokIden)
	if err != nil {
		return handleErr(err)
	}
	svc.Name = token.value
	if _, err := p.expect(TokLBrace); err != nil {
		return handleErr(err)
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
				_, _ = handleErr(err)
				continue
			}
			svc.LocalDefs = append(svc.LocalDefs, message)
		case TokRBrace:
			return &svc, nil
		default:
			return handleErr(expectErr(token, TokRpc, TokMessage, TokRBrace))
		}
	}
}

func (p *Parser) parseRpc() RpcAst {
	var token Token
	var err error
	var rpc RpcAst

	handleErr := func(err error) RpcAst {
		addAstKind(err, RpcAstKind)
		p.errs = append(p.errs, err)
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

func addAstKind(err error, kind AstKind) {
	switch err.(type) {
	case *ParseErr:
		err.(*ParseErr).Within = kind
	case *ExpectErr:
		err.(*ExpectErr).Within = kind
	}
}

type ParseErr struct {
	Actual Token
	Within AstKind
	Msg    string
}

func (err ParseErr) Error() string {
	return err.Msg + "at" + err.Actual.String()
}

func parseErr(actual Token, msg string) error {
	return &ParseErr{Actual: actual, Msg: msg}
}

type ExpectErr struct {
	Actual   Token
	Within   AstKind
	Expected []TokType
}

func (err ExpectErr) Error() string {
	var sb strings.Builder

	sb.WriteString("expected ")
	for i, tok := range err.Expected {
		sb.WriteString(tok.String())
		if i == len(err.Expected)-2 {
			sb.WriteString(" or ")
		} else if i != len(err.Expected)-1 {
			sb.WriteString(", ")
		}
	}

	sb.WriteString(" but got ")
	sb.WriteString(err.Actual.String())

	return sb.String()
}

func expectErr(actual Token, expected ...TokType) error {
	return &ExpectErr{Actual: actual, Expected: expected}
}
