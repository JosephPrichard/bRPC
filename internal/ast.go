package internal

import "fmt"

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
		panic(fmt.Sprintf("assertion error: unknown NodeKind: %d", kind))
	}
}

// Node an ast represents a recursive ast node
type Node interface {
	Kind() NodeKind
	SetTable(*TypeTable)
	Begin() int
	End() int
	Header() string
	ClearPos()
	GetPoisoned() bool
	SetPoisoned()
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
	Table      *TypeTable
	Name       string
	Fields     []FieldNode
	TypeParams []string
	LocalDefs  []Node
}

type Modifier int

const (
	Required Modifier = iota
	Optional
	Deprecated
)

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
	Table *TypeTable
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
	Table      *TypeTable
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
	Table      *TypeTable
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
	Iden      string
	Primitive bool // primitives do not need to be looked up in the type table
	TypeArgs  []TypeRefNode
	Array     []uint64
}

func (tags *Tags) GetPoisoned() bool {
	return tags.Poisoned
}

func (tags *Tags) SetPoisoned() {
	tags.Poisoned = true
}

func (node *PropertyNode) Kind() NodeKind { return PropertyNodeKind }
func (node *ImportNode) Kind() NodeKind   { return ImportNodeKind }
func (node *StructNode) Kind() NodeKind   { return StructNodeKind }
func (node *EnumNode) Kind() NodeKind     { return EnumNodeKind }
func (node *UnionNode) Kind() NodeKind    { return UnionNodeKind }
func (node *ServiceNode) Kind() NodeKind  { return ServiceNodeKind }
func (node *FieldNode) Kind() NodeKind    { return FieldNodeKind }
func (node *OptionNode) Kind() NodeKind   { return OptionNodeKind }
func (node *RpcNode) Kind() NodeKind      { return RpcNodeKind }
func (node *TypeRefNode) Kind() NodeKind  { return TypeRefNodeKind }

func (node *PropertyNode) SetTable(*TypeTable)      {}
func (node *ImportNode) SetTable(*TypeTable)        {}
func (node *StructNode) SetTable(table *TypeTable)  { node.Table = table }
func (node *EnumNode) SetTable(table *TypeTable)    { node.Table = table }
func (node *UnionNode) SetTable(table *TypeTable)   { node.Table = table }
func (node *ServiceNode) SetTable(table *TypeTable) { node.Table = table }
func (node *FieldNode) SetTable(*TypeTable)         {}
func (node *OptionNode) SetTable(*TypeTable)        {}
func (node *RpcNode) SetTable(*TypeTable)           {}
func (node *TypeRefNode) SetTable(*TypeTable)       {}

func (r *Positions) Begin() int { return r.B }
func (r *Positions) End() int   { return r.E }

func (r *Positions) Header() string {
	if r.B == r.E {
		return fmt.Sprintf("%d:", r.B)
	} else {
		return fmt.Sprintf("%d:%d:", r.B, r.E)
	}
}

func (r *Positions) ClearPos() {
	r.E = 0
	r.B = 0
}

func (node *StructNode) IterStruct(f func(*FieldNode)) {
	for i := range node.Fields {
		f(&node.Fields[i])
	}
}

func (node *UnionNode) IterOptions(f func(*OptionNode)) {
	for i := range node.Options {
		f(&node.Options[i])
	}
}

func (node *EnumNode) IterCases(f func(node *CaseNode)) {
	for i := range node.Cases {
		f(&node.Cases[i])
	}
}

func (node *ServiceNode) IterProcedures(f func(node *RpcNode)) {
	for i := range node.Procedures {
		f(&node.Procedures[i])
	}
}

func WalkMeta(visit func(Node), node Node) {
	if node == nil {
		return
	}
	visit(node)
	switch node := node.(type) {
	case *PropertyNode, *ImportNode, *EnumNode:
		// no children
	case *StructNode:
		WalkMetaList(visit, node.LocalDefs)
		for i := range node.Fields {
			field := &node.Fields[i]
			visit(field)
			visit(&field.Type)
		}
	case *UnionNode:
		WalkMetaList(visit, node.LocalDefs)
		for i := range node.Options {
			option := &node.Options[i]
			visit(option)
			visit(&option.Type)
		}
	case *ServiceNode:
		WalkMetaList(visit, node.LocalDefs)
		for i := range node.Procedures {
			proc := &node.Procedures[i]
			visit(proc)
			visit(&proc.Arg)
			visit(&proc.Ret)
		}
	default:
		panic(fmt.Sprintf("unsupported: walk call for node type is not implemented: %T", node))
	}
}

func WalkMetaList(visit func(Node), nodes []Node) {
	for _, node := range nodes {
		WalkMeta(visit, node)
	}
}
