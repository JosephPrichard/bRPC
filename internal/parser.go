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
	Nodes     []Ast
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
				p.Top().Error(err)
				p.Pop()
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

func (p *Parser) ParseTypeDef(name string) (Ast, error) {
	var ast Ast

	token := p.Read()
	switch token.T {
	case TokStruct:
		ast = &StructAst{Name: name}
		if _, err := p.Expect(TokLBrack); err != nil {
			return nil, err
		}
	case TokEnum:
		ast = &EnumAst{Name: name}
		if _, err := p.Expect(TokLBrack); err != nil {
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

	return p.ParseTypeDef(name)
}

func (p *Parser) ParseStruct(strct *StructAst) error {
	// LBrack has already been consumed
	var field *FieldAst

	// either we expect a field, a nested message, or we expect the end of the struct
	token := p.Peek()
	switch token.T {
	case TokRequired:
		fallthrough
	case TokOptional:
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
		return nil
	default:
		return parseErr(token, TokRequired, TokOptional, TokRBrace)
	}

	if field != nil {
		strct.Fields = append(strct.Fields, field)
		p.Push(field)
	}

	return nil
}

func (p *Parser) ParseUnion(union *UnionAst) error {
	// Equal has already been consumed
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
	// LBrack has already been consumed
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

func (p *Parser) ParseField(field *FieldAst) error {
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
	default:
		return parseErr(token, TokIden, TokRBrack)
	}

	if rpc != nil {
		service.Operations = append(service.Operations, rpc)
		p.Push(rpc)
	}
	return nil
}

func (p *Parser) ParseType() (Ast, error) {
	token := p.Peek()
	if token.T == TokIden {
		p.Consume()
		return &TypeAst{Name: token.Value}, nil
	} else if token.T == TokStruct || token.T == TokUnion || token.T == TokEnum {
		def, err := p.ParseTypeDef("")
		if err != nil {
			return nil, err
		}
		p.Push(def)
		return def, nil
	} else {
		p.Consume()
		return nil, parseErr(token, TokIden, TokStruct, TokUnion, TokEnum)
	}
}

func (p *Parser) ParseRpc(rpc *RpcAst) error {
	// Iden and LParen has already been consumed
	var def Ast
	var err error

	if rpc.Arg == nil {
		// we haven't parsed the arg yet
		if def, err = p.ParseType(); err != nil {
			return err
		}
		rpc.Arg = def
	} else if rpc.Ret == nil {
		// we haven't parsed the ret yet
		if err = p.ExpectChain(TokRParen, TokReturns, TokLParen); err != nil {
			return err
		}
		if def, err = p.ParseType(); err != nil {
			return err
		}
		rpc.Ret = def
		if _, err = p.Expect(TokRParen); err != nil {
			return err
		}
	} else {
		// we're done parsing the ast
		p.Pop()
		if rpc.Arg == nil || rpc.Ret == nil {
			panic(fmt.Sprintf("assertion failed: rpc arg: %v and ret: %v should never be nil after parsing", rpc.Arg, rpc.Ret))
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
			sb.WriteString("or")
		} else if i != len(expected)-1 {
			sb.WriteString(",")
		}
	}

	sb.WriteString(" but got '")
	sb.WriteString(actual.String())
	sb.WriteString("'")
	return errors.New(sb.String())
}
