package internal

import "fmt"

type Modifier int

const (
	Required Modifier = iota
	Optional
)

type NodeKind int

const (
	UnknownNodeKind NodeKind = iota
	PropertyNodeKind
	ImportNodeKind
	MessageNodeKind
	StructNodeKind
	EnumNodeKind
	UnionNodeKind
	CaseNodeKind
	FieldNodeKind
	OptionNodeKind
	ServiceNodeKind
	RpcNodeKind
	TypeRefNodeKind
)

func (kind NodeKind) String() string {
	switch kind {
	case UnknownNodeKind:
		return "unknown"
	case PropertyNodeKind:
		return "property"
	case ImportNodeKind:
		return "import"
	case MessageNodeKind:
		return "message"
	case StructNodeKind:
		return "struct"
	case EnumNodeKind:
		return "enum"
	case UnionNodeKind:
		return "union"
	case FieldNodeKind:
		return "field"
	case CaseNodeKind:
		return "case"
	case OptionNodeKind:
		return "option"
	case ServiceNodeKind:
		return "service"
	case RpcNodeKind:
		return "rpc"
	case TypeRefNodeKind:
		return "type"
	default:
		panic(fmt.Sprintf("assertion error: unknown AstKind: %d", kind))
	}
}

// Node an ast represents a recursive ast node
type Node interface {
	Kind() NodeKind
	Begin() int
	End() int
	Header() string
	ClearPos()
	IsPoisoned() bool
}

type Positions struct {
	B int
	E int
}

// Tags stores some common information we want to keep track for each Node
type Tags struct {
	Poisoned bool
}

type PropertyNode struct {
	Tags
	Positions
	Name  string
	Value string
}

type ImportNode struct {
	Tags
	Positions
	Path string
}

type StructNode struct {
	Tags
	Positions
	Table      *SymbolTable
	Name       string
	Fields     []FieldNode
	TypeParams []string
	LocalDefs  []Node
}

type FieldNode struct {
	Tags
	Positions
	Modifier Modifier
	Name     string
	Type     TypeRefNode
	Ord      uint64
}

type EnumNode struct {
	Tags
	Positions
	Table *SymbolTable
	Name  string
	Size  uint64
	Cases []CaseNode
}

type CaseNode struct {
	Tags // an enum case still contains ast meta tags even though it is not a recursive AST
	Name string
	Ord  uint64
}

type UnionNode struct {
	Tags
	Positions
	Table      *SymbolTable
	Name       string
	Size       uint64
	Options    []OptionNode
	TypeParams []string
	LocalDefs  []Node
}

type OptionNode struct {
	Tags
	Positions
	Type TypeRefNode
	Ord  uint64
}

type ServiceNode struct {
	Tags
	Positions
	Table      *SymbolTable
	Name       string
	Procedures []RpcNode
	LocalDefs  []Node
}

type RpcNode struct {
	Tags
	Positions
	Name string
	Ord  uint64
	Arg  TypeRefNode
	Ret  TypeRefNode
}

type TypeRefNode struct {
	Tags
	Positions
	Table    *SymbolTable
	Iden     string
	TypeArgs []TypeRefNode
	Array    []uint64
}

func (tags *Tags) IsPoisoned() bool {
	return tags.Poisoned
}

func (ast *PropertyNode) Kind() NodeKind { return PropertyNodeKind }
func (ast *ImportNode) Kind() NodeKind   { return ImportNodeKind }
func (ast *StructNode) Kind() NodeKind   { return StructNodeKind }
func (ast *EnumNode) Kind() NodeKind     { return EnumNodeKind }
func (ast *UnionNode) Kind() NodeKind    { return UnionNodeKind }
func (ast *ServiceNode) Kind() NodeKind  { return ServiceNodeKind }
func (ast *FieldNode) Kind() NodeKind    { return FieldNodeKind }
func (ast *OptionNode) Kind() NodeKind   { return OptionNodeKind }
func (ast *RpcNode) Kind() NodeKind      { return RpcNodeKind }
func (ast *TypeRefNode) Kind() NodeKind  { return TypeRefNodeKind }

func (r *Positions) Begin() int { return r.B }
func (r *Positions) End() int   { return r.E }

func (r *Positions) Header() string {
	if r.B == r.E {
		return fmt.Sprintf("%d: ", r.B)
	} else {
		return fmt.Sprintf("%d:%d: ", r.B, r.E)
	}
}

func (r *Positions) ClearPos() {
	r.E = 0
	r.B = 0
}

func WalkMeta(visit func(Node), ast Node) {
	if ast == nil {
		return
	}
	visit(ast)
	switch ast := ast.(type) {
	case *PropertyNode, *ImportNode, *EnumNode:
		// no children
	case *StructNode:
		WalkMetaList(visit, ast.LocalDefs)
		for i := range ast.Fields {
			field := &ast.Fields[i]
			visit(field)
			visit(&field.Type)
		}
	case *UnionNode:
		WalkMetaList(visit, ast.LocalDefs)
		for i := range ast.Options {
			option := &ast.Options[i]
			visit(option)
			visit(&option.Type)
		}
	case *ServiceNode:
		WalkMetaList(visit, ast.LocalDefs)
		for i := range ast.Procedures {
			proc := &ast.Procedures[i]
			visit(proc)
			visit(&proc.Arg)
			visit(&proc.Ret)
		}
	default:
		panic(fmt.Sprintf("unsupported: walk call for ast type is not implemented: %T", ast))
	}
}

func WalkMetaList(visit func(Node), asts []Node) {
	for _, ast := range asts {
		WalkMeta(visit, ast)
	}
}
