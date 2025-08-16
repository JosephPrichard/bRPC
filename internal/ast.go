package internal

import "fmt"

type Modifier int

const (
	Required Modifier = iota
	Optional
)

type AstKind int

const (
	RootAstKind AstKind = iota
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
	case RootAstKind:
		return "<root>"
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
		return "<arrPrefix>"
	default:
		panic(fmt.Sprintf("assertion error: unknown AstKind: %d", kind))
	}
}

type Ast interface {
	Kind() AstKind
	Begin() int
	End() int
	Header() string
	ClearPos()
}
type Range struct {
	B int
	E int
}

type PropertyAst struct {
	Range
	Name  string
	Value string
}

type ImportAst struct {
	Range
	Path string
}

type StructAst struct {
	Range
	Table      *SymbolTable
	Name       string // an empty string is an anonymous struct
	Fields     []FieldAst
	TypeParams []string
	LocalDefs  []Ast
}

type EnumAst struct {
	Range
	Table *SymbolTable
	Name  string // an empty string is an anonymous enum
	Cases []EnumCase
}

type EnumCase struct {
	Name string
	Ord  uint64
}

type UnionAst struct {
	Range
	Table      *SymbolTable
	Name       string // an empty string is an anonymous union
	Options    []OptionAst
	TypeParams []string
	LocalDefs  []Ast
}

type FieldAst struct {
	Range
	Modifier Modifier
	Name     string
	Type     Ast
	Ord      uint64
}

type OptionAst struct {
	Range
	Type Ast
	Ord  uint64
}

type ServiceAst struct {
	Range
	Table      *SymbolTable
	Name       string
	Procedures []RpcAst
	LocalDefs  []Ast
}

type RpcAst struct {
	Range
	Name string
	Ord  uint64
	Arg  Ast
	Ret  Ast
}

type TypeRefAst struct {
	Range
	Table    *SymbolTable
	Alias    string // an empty string is not an alias
	Iden     string
	TypeArgs []Ast
}

type TypeArrAst struct {
	Range
	Type Ast
	Size []uint64 // 0 means the array is a dynamic array
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
func (ast *TypeRefAst) Kind() AstKind  { return TypeAstKind }
func (ast *TypeArrAst) Kind() AstKind  { return ArrayAstKind }

func (r *Range) Begin() int { return r.B }
func (r *Range) End() int   { return r.E }

func (r *Range) Header() string {
	if r.B == r.E {
		return fmt.Sprintf("%d: ", r.B)
	} else {
		return fmt.Sprintf("%d:%d: ", r.B, r.E)
	}
}

func (r *Range) ClearPos() {
	r.E = 0
	r.B = 0
}

func Walk(visit func(Ast), ast Ast) {
	if ast == nil {
		return
	}
	visit(ast)
	switch ast := ast.(type) {
	case *PropertyAst, *ImportAst:
		// nothing to do
	case *StructAst:
		WalkList(visit, ast.LocalDefs)
		for i := range ast.Fields {
			Walk(visit, &ast.Fields[i])
		}
	case *UnionAst:
		WalkList(visit, ast.LocalDefs)
		for i := range ast.Options {
			Walk(visit, &ast.Options[i])
		}
	case *ServiceAst:
		WalkList(visit, ast.LocalDefs)
		for i := range ast.Procedures {
			Walk(visit, &ast.Procedures[i])
		}
	case *OptionAst:
		Walk(visit, ast.Type)
	case *FieldAst:
		Walk(visit, ast.Type)
	case *RpcAst:
		Walk(visit, ast.Arg)
		Walk(visit, ast.Ret)
	case *TypeRefAst:
		WalkList(visit, ast.TypeArgs)
	case *TypeArrAst:
		Walk(visit, ast.Type)
	}
}

func WalkList(visit func(Ast), asts []Ast) {
	for _, ast := range asts {
		Walk(visit, ast)
	}
}
