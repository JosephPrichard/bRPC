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

var Eof = errors.New("reached end of token stream")

func (p *Parser) Parse() error {
	for {
		err := p.Ast()
		if errors.Is(err, Eof) {
			break
		} else if err != nil {
			return err
		}
	}
	if len(p.Stack) > 0 {
		return fmt.Errorf("reached end of stream while parsing: %v", StringOfAst(p.Top()))
	}
	return nil
}

func (p *Parser) Ast() error {
	if len(p.Stack) == 0 {
		return p.Root()
	} else {
		top := p.Top()
		switch top.(type) {
		case *StructAst:
			return p.Struct(top.(*StructAst))
		case *UnionAst:
			return p.Union(top.(*UnionAst))
		case *EnumAst:
			return p.Enum(top.(*EnumAst))
		case *FieldAst:
			return p.Field()
		case *ServiceAst:
			return p.Service()
		case *OperationAst:
			return p.Operation()
		case *PairAst:
			return p.Pair()
		default:
			panic(fmt.Sprintf("expected ast node to be ast type, got: %v", reflect.TypeOf(top)))
		}
	}
	return nil
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

func (p *Parser) Peek() Token {
	if p.HasToken {
		return p.LastToken
	} else {
		p.LastToken = <-p.Tokens
		p.HasToken = true
		return p.LastToken
	}
}

func (p *Parser) Expect(expected ...TokType) (Token, error) {
	token := <-p.Tokens
	for _, t := range expected {
		if t == token.T {
			return token, nil
		}
	}
	return Token{}, parseErr(token, expected...)
}

func (p *Parser) Root() error {
	var token Token
	var err error

	if token = p.Read(); token.T == TokEof {
		return Eof
	}

	switch token.T {
	case TokMessage:
		if token, err = p.Expect(TokIden); err != nil {
			return err
		}
		name := token.Value

		var ast Ast
		var t TokType

		switch token.T {
		case TokStruct:
			ast = StructAst{Name: name}
			t = TokLBrace
		case TokEnum:
			ast = EnumAst{Name: name}
			t = TokLBrace
		case TokUnion:
			ast = UnionAst{Name: name}
			t = TokEqual
		default:
			return parseErr(token, TokStruct, TokUnion, TokEnum)
		}

		if _, err = p.Expect(t); err != nil {
			return err
		}
		p.Stack = append(p.Stack, ast)
	case TokService:
		if token, err = p.Expect(TokIden); err != nil {
			return err
		}
		name := token.Value

		if _, err = p.Expect(TokLBrace); err != nil {
			return err
		}
		ast := ServiceAst{Name: name}
		p.Stack = append(p.Stack, ast)
	default:
		return parseErr(token, TokMessage, TokService)
	}

	return nil
}

func (p *Parser) Struct(ast *StructAst) error {
	token := p.Read()

	var field *FieldAst
	switch token.T {
	case TokRequired:
		field = &FieldAst{Modifier: Required}
	case TokOptional:
		field = &FieldAst{Modifier: Optional}
	case TokRBrace:
	default:
		return parseErr(token, TokRequired, TokOptional)
	}

	if field != nil {
		ast.Fields = append(ast.Fields, field)
		p.Stack = append(p.Stack, field)
	}

	return nil
}

func (p *Parser) Union(ast *UnionAst) error {
	if token := p.Peek(); token.T == TokPipe {
		p.Read()
	}

	for {
		var option string
		var err error

		token := p.Read()
		switch token.T {
		case TokIden:
			option = token.Value
		case TokTerm:
			p.Pop()
			return nil
		default:
			return parseErr(token, TokIden, TokTerm)
		}
		if _, err = p.Expect(TokPipe); err != nil {
			return err
		}

		ast.Options = append(ast.Options, option)
	}
}

func (p *Parser) Enum(ast *EnumAst) error {
	for {
		var c EnumCase
		var err error

		token := p.Read()
		switch token.T {
		case TokIden:
			c.Name = token.Value
		case TokRBrack:
			p.Pop()
			return nil
		default:
			return parseErr(token, TokIden, TokRBrack)
		}
		if _, err = p.Expect(TokEqual); err != nil {
			return err
		}

		token = p.Read()

		var base int
		switch token.T {
		case TokInteger:
			base = 10
		case TokBinary:
			base = 2
		case TokHex:
			base = 16
		default:
			return parseErr(token, TokInteger, TokBinary, TokHex)
		}

		value, err := strconv.ParseInt(token.Value, base, 64)
		if err != nil {
			return fmt.Errorf("%v: %s is not a valid integer", err, token.String())
		}
		c.Value = value

		ast.Cases = append(ast.Cases, c)
	}
}

func (p *Parser) Field() error {
	return nil
}

func (p *Parser) Service() error {
	return nil
}

func (p *Parser) Operation() error {
	return nil
}

func (p *Parser) Pair() error {
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
