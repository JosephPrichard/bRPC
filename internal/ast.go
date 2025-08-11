package internal

import "fmt"

type Modifier int

const (
	Required Modifier = iota
	Optional
)

type AstKind int

const (
	UnknownAstKind AstKind = iota
	PropertyAstKind
	ImportAstKind
	StructAstKind
	EnumAstKind
	UnionAstKind
	FieldAstKind
	OptionAstKind
	ServiceAstKind
	RpcAstKind
	TypeAstKind
	ArrayAstKind
)

func (kind AstKind) String() string {
	switch kind {
	case UnknownAstKind:
		return "<unknown>"
	case PropertyAstKind:
		return "<property>"
	case ImportAstKind:
		return "<import>"
	case StructAstKind:
		return "<struct>"
	case EnumAstKind:
		return "<enum>"
	case UnionAstKind:
		return "<union>"
	case FieldAstKind:
		return "<field>"
	case OptionAstKind:
		return "<option>"
	case ServiceAstKind:
		return "<service>"
	case RpcAstKind:
		return "<rpc>"
	case TypeAstKind:
		return "<type>"
	case ArrayAstKind:
		return "<array>"
	default:
		panic(fmt.Sprintf("assertion error: unknown AstKind: %d", kind))
	}
}

type Ast interface {
	Kind() AstKind
}

type PropertyAst struct {
	Name  string
	Value string
}

type ImportAst struct {
	Path string
}

type StructAst struct {
	Name      string // an empty string is an anonymous struct
	Fields    []FieldAst
	TypeArgs  []string
	LocalDefs []Ast
}

type EnumAst struct {
	Name  string // an empty string is an anonymous enum
	Cases []EnumCase
}

type EnumCase struct {
	Name string
	Ord  uint64
}

type UnionAst struct {
	Name      string // an empty string is an anonymous union
	Options   []OptionAst
	TypeArgs  []string
	LocalDefs []Ast
}

type FieldAst struct {
	Modifier Modifier
	Name     string
	Type     Ast
	Ord      uint64
}

type OptionAst struct {
	Type Ast
	Ord  uint64
}

type ServiceAst struct {
	Name       string
	Procedures []RpcAst
	LocalDefs  []Ast
}

type RpcAst struct {
	Name string
	Ord  uint64
	Arg  Ast
	Ret  Ast
}

type TypeAst struct {
	Alias    string // an empty string is not an alias
	Value    string
	TypeArgs []Ast
}

type ArrayAst struct {
	Type Ast
	Size uint64 // 0 means the array is a dynamic array
}

func (ast *PropertyAst) Kind() AstKind { return PropertyAstKind }
func (ast *ImportAst) Kind() AstKind   { return ImportAstKind }
func (ast *StructAst) Kind() AstKind   { return StructAstKind }
func (ast *EnumAst) Kind() AstKind     { return EnumAstKind }
func (ast *UnionAst) Kind() AstKind    { return UnionAstKind }
func (ast *ServiceAst) Kind() AstKind  { return ServiceAstKind }
func (ast *RpcAst) Kind() AstKind      { return RpcAstKind }
func (ast *OptionAst) Kind() AstKind   { return OptionAstKind }
func (ast *FieldAst) Kind() AstKind    { return FieldAstKind }
func (ast *TypeAst) Kind() AstKind     { return TypeAstKind }
func (ast *ArrayAst) Kind() AstKind    { return ArrayAstKind }
