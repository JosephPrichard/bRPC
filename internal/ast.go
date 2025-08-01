package internal

import "fmt"

type Ast any

func StringOfAst(ast Ast) string {
	switch ast.(type) {
	case *StructAst:
		return "<struct>"
	case *EnumAst:
		return "<enum>"
	case *UnionAst:
		return "<union>"
	case *FieldAst:
		return "<field>"
	case *EnumCase:
		return "<case>"
	case *ServiceAst:
		return "<service>"
	case *OperationAst:
		return "<operation>"
	case *PairAst:
		return "<pair>"
	default:
		panic(fmt.Sprintf("expected ast node to be ast type, got: %v", ast))
	}
}

type StructAst struct {
	Name   string
	Fields []*FieldAst
}

type EnumAst struct {
	Name  string
	Cases []EnumCase
}

type EnumCase struct {
	Name  string
	Value int64
}

type UnionAst struct {
	Name    string
	Options []string
}

type Modifier int

const (
	Required Modifier = iota
	Optional
)

type FieldAst struct {
	Modifier Modifier
	Name     string
	Type     Ast
	Ord      int
}

type ServiceAst struct {
	Name       string
	Operations []*OperationAst
}

type OperationAst struct {
	Name string
	Args []*PairAst
	Ret  []Ast
}

type PairAst struct {
	Name string
	Type Ast
}
