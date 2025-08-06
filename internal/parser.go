package internal

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Parser struct {
	Tokens    chan Token
	LastToken Token
	HasToken  bool
	Stack     []Ast
	Asts      []Ast
}

func NewParser(tokens chan Token) Parser {
	return Parser{Tokens: tokens}
}

func (p *Parser) Top() Ast {
	return p.Stack[len(p.Stack)-1]
}

func (p *Parser) Pop() {
	p.Stack = p.Stack[:len(p.Stack)-1]
}

func (p *Parser) Push(ast Ast) {
	if ast == nil {
		panic("assertion failed: attempted to push a nil ast to stack")
	}
	p.Stack = append(p.Stack, ast)
}

func (p *Parser) Read() Token {
	var token Token
	if p.HasToken {
		token = p.LastToken
		p.LastToken = Token{}
	} else {
		token = <-p.Tokens
	}
	p.HasToken = false
	return token
}

func (p *Parser) Consume() {
	if p.HasToken {
		p.LastToken = Token{}
	} else {
		<-p.Tokens
	}
	p.HasToken = false
}

func (p *Parser) Peek() Token {
	if p.HasToken {
		return p.LastToken
	} else {
		p.LastToken = <-p.Tokens
		p.HasToken = true
		return p.LastToken
	}
}

func (p *Parser) Expect(expected TokType) (Token, error) {
	token := p.Read()
	if expected == token.T {
		return token, nil
	}
	return Token{}, parseErr(token, expected)
}

func (p *Parser) ExpectChain(chain ...TokType) error {
	for _, expected := range chain {
		if _, err := p.Expect(expected); err != nil {
			return err
		}
	}
	return nil
}

var Eof = errors.New("reached end of token stream")

func (p *Parser) Parse() error {
	for {
		err := p.ParseAst()
		if errors.Is(err, Eof) {
			break
		} else if err != nil {
			if len(p.Stack) > 0 {
				top := p.Top()
				p.Pop()
				top.Error(err)
				if len(p.Stack) == 0 {
					p.Asts = append(p.Asts, top)
				}
			} else {
				return err
			}
		}
	}
	if len(p.Stack) > 0 {
		return fmt.Errorf("reached end of stream while parsing: %s", p.Top().String())
	}
	return nil
}

func (p *Parser) ParseAst() error {
	if len(p.Stack) == 0 {
		return p.ParseRoot()
	} else {
		top := p.Top()
		switch top.(type) {
		case *StructAst:
			return p.ParseStruct(top.(*StructAst))
		case *UnionAst:
			return p.ParseUnion(top.(*UnionAst))
		case *EnumAst:
			return p.ParseEnum(top.(*EnumAst))
		case *FieldAst:
			return p.ParseField(top.(*FieldAst))
		case *ServiceAst:
			return p.ParseService(top.(*ServiceAst))
		case *RpcAst:
			return p.ParseRpc(top.(*RpcAst))
		default:
			panic(fmt.Sprintf("assertion failed: expected ast node to be ast type, got: %v", reflect.TypeOf(top)))
		}
	}
	return nil
}

func (p *Parser) ParseRoot() error {
	token := p.Read()
	if token.T == TokEof {
		return Eof
	}

	var ast Ast
	var err error

	switch token.T {
	case TokMessage:
		if ast, err = p.ParseMessage(); err != nil {
			return err
		}
		p.Push(ast)
	case TokService:
		if token, err = p.Expect(TokIden); err != nil {
			return err
		}
		name := token.Value
		if _, err = p.Expect(TokLBrace); err != nil {
			return err
		}
		ast = &ServiceAst{Name: name}
		p.Push(ast)
	default:
		return parseErr(token, TokMessage, TokService)
	}

	return nil
}

func (p *Parser) ParseBeginObject(name string) (Ast, error) {
	var ast Ast

	token := p.Read()
	switch token.T {
	case TokStruct:
		ast = &StructAst{Name: name}
		if _, err := p.Expect(TokLBrace); err != nil {
			return nil, err
		}
	case TokEnum:
		ast = &EnumAst{Name: name}
		if _, err := p.Expect(TokLBrace); err != nil {
			return nil, err
		}
	case TokUnion:
		ast = &UnionAst{Name: name}
		if _, err := p.Expect(TokEqual); err != nil {
			return nil, err
		}
	default:
		return nil, parseErr(token, TokStruct, TokUnion, TokEnum)
	}

	return ast, nil
}

func (p *Parser) ParseMessage() (Ast, error) {
	token, err := p.Expect(TokIden)
	if err != nil {
		return nil, err
	}
	name := token.Value

	return p.ParseBeginObject(name)
}

func (p *Parser) ParseStruct(strct *StructAst) error {
	// invariant: LBrack has already been consumed
	var field *FieldAst

	// either we expect a field, a nested message, or we expect the end of the struct
	token := p.Peek()
	switch token.T {
	case TokOptional, TokRequired:
		field = &FieldAst{}
	case TokMessage:
		p.Consume()
		message, err := p.ParseMessage()
		if err != nil {
			return err
		}
		strct.LocalDefs = append(strct.LocalDefs, message)
		p.Push(message)
	case TokRBrace:
		p.Consume()
		p.Pop()
		p.Asts = append(p.Asts, strct)
		return nil
	default:
		p.Consume()
		return parseErr(token, TokRequired, TokOptional, TokRBrace)
	}

	if field != nil {
		strct.Fields = append(strct.Fields, field)
		p.Push(field)
	}

	return nil
}

func (p *Parser) ParseUnion(union *UnionAst) error {
	// invariant: Equal has already been consumed
	for {
		if len(union.Options) != 0 {
			if _, err := p.Expect(TokPipe); err != nil {
				return err
			}
		}

		token := p.Read()
		switch token.T {
		case TokIden:
			union.Options = append(union.Options, token.Value)
			continue
		case TokMessage:
			message, err := p.ParseMessage()
			if err != nil {
				return err
			}
			union.LocalDefs = append(union.LocalDefs, message)
			p.Push(message)
		case TokTerm:
			p.Pop()
			p.Asts = append(p.Asts, union)
		default:
			return parseErr(token, TokIden, TokStruct, TokTerm)
		}
		return nil
	}
}

func (p *Parser) ParseNumeric() (int64, error) {
	token := p.Read()

	var intBase int
	switch token.T {
	case TokInteger:
		intBase = 10
	case TokBinary:
		intBase = 2
	case TokHex:
		intBase = 16
	default:
		return 0, parseErr(token, TokInteger, TokBinary, TokHex)
	}

	value, err := strconv.ParseInt(token.Value, intBase, 64)
	if err != nil {
		return 0, fmt.Errorf("%v: %s is not a valid integer", err, token.String())
	}
	return value, nil
}

func (p *Parser) ParseEnum(enum *EnumAst) error {
	// invariant: LBrack has already been consumed
	for {
		var ec EnumCase

		token := p.Read()
		switch token.T {
		case TokIden:
			ec.Name = token.Value
		case TokMessage:
			message, err := p.ParseMessage()
			if err != nil {
				return err
			}
			enum.LocalDefs = append(enum.LocalDefs, message)
			p.Push(message)
		case TokRBrack:
			p.Pop()
			p.Asts = append(p.Asts, enum)
			return nil
		default:
			return parseErr(token, TokIden, TokRBrack)
		}
		if _, err := p.Expect(TokEqual); err != nil {
			return err
		}

		value, err := p.ParseNumeric()
		if err != nil {
			return err
		}

		ec.Value = value
		enum.Cases = append(enum.Cases, ec)
	}
}

func (p *Parser) ParseArrayPrefix() (uint64, error) {
	if _, err := p.Expect(TokLBrack); err != nil {
		return 0, err
	}
	token := p.Read()
	switch token.T {
	case TokInteger:
		if _, err := p.Expect(TokRBrack); err != nil {
			return 0, err
		}
		size, err := strconv.ParseUint(token.Value, 10, 64)
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

func (p *Parser) ParseBeginType() (Ast, error) {
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

	for isTerminal := false; !isTerminal; {
		token := p.Peek()
		switch token.T {
		case TokIden:
			p.Consume()
			ref := &TypeRefAst{Name: token.Value}
			appendAst(ref)
			isTerminal = true
		case TokLBrack:
			size, err := p.ParseArrayPrefix()
			if err != nil {
				return nil, err
			}
			array := &TypeArrayAst{Size: size}
			appendAst(array)
		case TokStruct, TokUnion, TokEnum:
			object, err := p.ParseBeginObject("")
			if err != nil {
				return nil, err
			}
			p.Push(object)
			appendAst(object)
			isTerminal = true
		default:
			p.Consume()
			return nil, parseErr(token, TokIden, TokStruct, TokUnion, TokEnum)
		}
	}

	if root == nil {
		panic("assertion failed: root is nil after parsing a <type>")
	}
	return root, nil
}

func (p *Parser) ParseOrd() (uint64, error) {
	token, err := p.Expect(TokOrd)
	if err != nil {
		return 0, err
	}
	if len(token.Value) < 2 {
		panic(fmt.Errorf("assertion failed: an ord token should have at least 2 characters"))
	}
	ord, err := strconv.ParseUint(token.Value[1:], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%v: %s is not a valid ord", err, token.String())
	}
	return ord, nil
}

func (p *Parser) ParseField(field *FieldAst) error {
	var token Token
	var err error
	var ord uint64
	var typ Ast

	if field.Modifier == Undefined {
		token = p.Read()
		switch token.T {
		case TokRequired:
			field.Modifier = Required
		case TokOptional:
			field.Modifier = Optional
		default:
			return parseErr(token, TokRequired, TokOptional)
		}
	} else if field.Name == "" {
		if token, err = p.Expect(TokIden); err != nil {
			return err
		}
		field.Name = token.Value
	} else if field.Ord == 0 {
		if ord, err = p.ParseOrd(); err != nil {
			return err
		}
		field.Ord = ord
	} else if field.Type == nil {
		if typ, err = p.ParseBeginType(); err != nil {
			return err
		}
		field.Type = typ
	} else {
		if token, err = p.Expect(TokTerm); err != nil {
			return err
		}
		p.Pop()
		if field.Modifier == Undefined || field.Name == "" || field.Ord < 0 || field.Type == nil {
			panic(fmt.Sprintf("assertion failed: field: %v should never be unset after parsing", field))
		}
	}

	return nil
}

func (p *Parser) ParseService(service *ServiceAst) error {
	var rpc *RpcAst
	var err error

	token := p.Read()
	switch token.T {
	case TokRpc:
		if token, err = p.Expect(TokIden); err != nil {
			return err
		}
		rpc = &RpcAst{Name: token.Value}
		if _, err = p.Expect(TokLParen); err != nil {
			return err
		}
	case TokMessage:
		message, err := p.ParseMessage()
		if err != nil {
			return err
		}
		service.LocalDefs = append(service.LocalDefs, message)
		p.Push(message)
	case TokRBrack:
		p.Pop()
		p.Asts = append(p.Asts, service)
	default:
		return parseErr(token, TokIden, TokRBrack)
	}

	if rpc != nil {
		service.Operations = append(service.Operations, rpc)
		p.Push(rpc)
	}
	return nil
}

func (p *Parser) ParseRpc(rpc *RpcAst) error {
	// Iden and LParen has already been consumed
	var typ Ast
	var err error

	if rpc.Arg == nil {
		if typ, err = p.ParseBeginType(); err != nil {
			return err
		}
		rpc.Arg = typ
	} else if rpc.Ret == nil {
		if err = p.ExpectChain(TokRParen, TokReturns, TokLParen); err != nil {
			return err
		}
		if typ, err = p.ParseBeginType(); err != nil {
			return err
		}
		rpc.Ret = typ
		if _, err = p.Expect(TokRParen); err != nil {
			return err
		}
	} else {
		p.Pop()
		if rpc.Arg == nil || rpc.Ret == nil {
			panic(fmt.Sprintf("assertion failed: rpc: %v should never be unset after parsing", rpc))
		}
	}

	return nil
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
