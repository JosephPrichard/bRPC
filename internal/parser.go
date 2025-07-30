package internal

import (
	"errors"
	"fmt"
)

type Ast any

type NodeState int

const (
	StructName NodeState = iota
	StructFields
	EnumName
	EnumCases
	UnionName
	UnionCases
	UnionCase
	UnionSplit
	FieldModifier
	FieldName
	FieldOrd
	FieldType
	FieldEnd
	ServiceName
	OperationName
)

type StructAst struct {
	state  NodeState
	Name   string
	Fields []FieldAst
}

type EnumAst struct {
	state NodeState
	Name  string
	Cases []CaseAst
}

type UnionAst struct {
	state   NodeState
	Name    string
	Options []string
}

type Modifier int

const (
	Required Modifier = iota
	Optional
)

type FieldAst struct {
	state    NodeState
	Modifier Modifier
	Name     string
	Type     Ast
	Ord      int
}

type CaseAst struct {
	state NodeState
	Name  string
	Value int
}

type ServiceAst struct {
	state      NodeState
	Name       string
	Operations []OperationAst
}

type OperationAst struct {
	state NodeState
	Name  string
	Args  []PairAst
	Ret   []Ast
}

type PairAst struct {
	state NodeState
	Name  string
	Type  Ast
}

type Parser struct {
	Tokens chan Token
	Stack  []Ast
	Nodes  []Ast
}

func (p *Parser) Top() Ast {
	return p.Stack[len(p.Stack)-1]
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
		return fmt.Errorf("reached end of stream while parsing: %v", p.Top())
	}
	return nil
}

func (p *Parser) Ast() error {
	if len(p.Stack) == 0 {
		return p.Root()
	} else {
		top := p.Top()
		switch top.(type) {
		case StructAst:
			return p.Struct()
		case UnionAst:
			return p.Union()
		case EnumAst:
			return p.Enum()
		case FieldAst:
			return p.Field()
		case ServiceAst:
			return p.Service()
		case OperationAst:
			return p.Operation()
		case PairAst:
			return p.Pair()
		default:
			panic(fmt.Sprintf("expected ast node to be ast type, got: %v", top))
		}
	}
	return nil
}

func (p *Parser) Root() error {
	token, ok := <-p.Tokens
	if !ok {
		return Eof
	}

	switch token.T {
	case TokMessage:
	case TokService:
	default:
		return fmt.Errorf("expected 'message' or 'service', but got %v", token.String())
	}

	return nil
}

func (p *Parser) Message() error {
	return nil
}

func (p *Parser) Struct() error {
	return nil
}
func (p *Parser) Union() error {
	return nil
}

func (p *Parser) Enum() error {
	return nil
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
