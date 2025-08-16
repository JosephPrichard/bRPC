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
	Begin() Token
	End() Token
	Clear()
}

type Markers struct {
	B Token
	E Token
}

func makeMarkers(tB TokType, vB string, tE TokType, vE string) Markers {
	return Markers{
		B: Token{TokVal: TokVal{t: tB, value: vB}},
		E: Token{TokVal: TokVal{t: tE, value: vE}},
	}
}

type PropertyAst struct {
	Markers
	Name  string
	Value string
}

type ImportAst struct {
	Markers
	Path string
}

type StructAst struct {
	Markers
	Table      *SymbolTable
	Name       string // an empty string is an anonymous struct
	Fields     []FieldAst
	TypeParams []string
	LocalDefs  []Ast
}

type EnumAst struct {
	Markers
	Table *SymbolTable
	Name  string // an empty string is an anonymous enum
	Cases []EnumCase
}

type EnumCase struct {
	Name string
	Ord  uint64
}

type UnionAst struct {
	Markers
	Table      *SymbolTable
	Name       string // an empty string is an anonymous union
	Options    []OptionAst
	TypeParams []string
	LocalDefs  []Ast
}

type FieldAst struct {
	Markers
	Modifier Modifier
	Name     string
	Type     Ast
	Ord      uint64
}

type OptionAst struct {
	Markers
	Type Ast
	Ord  uint64
}

type ServiceAst struct {
	Markers
	Table      *SymbolTable
	Name       string
	Procedures []RpcAst
	LocalDefs  []Ast
}

type RpcAst struct {
	Markers
	Name string
	Ord  uint64
	Arg  Ast
	Ret  Ast
}

type TypRefAst struct {
	Markers
	Table    *SymbolTable
	Alias    string // an empty string is not an alias
	Iden     string
	TypeArgs []Ast
}

type TypArrAst struct {
	Markers
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
func (ast *TypRefAst) Kind() AstKind   { return TypeAstKind }
func (ast *TypArrAst) Kind() AstKind   { return ArrayAstKind }

func (m *Markers) Begin() Token { return m.B }
func (m *Markers) End() Token   { return m.E }
func (m *Markers) Clear() {
	// clears positional marker information, useful for testing - we don't really care to assert this information since it changes very frequently
	m.B.beg = 0
	m.B.end = 0
	m.E.beg = 0
	m.E.end = 0
}
