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
	TypeNodeKind
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
	case TypeNodeKind:
		return "type"
	default:
		panic(fmt.Sprintf("assertion error: unknown NodeKind: %d", kind))
	}
}

// Node an ast represents a recursive ast node
type Node interface {
	Kind() NodeKind
	Begin() int
	End() int
	Header() string
	Clear()
}

func cmpOrd(o1 uint64, o2 uint64) int {
	return int(o1 - o2)
}

type Positions struct {
	B int
	E int
}

type PropertyNode struct {
	Positions
	Poisoned bool
	Name     string
	Value    string
}

type ImportNode struct {
	Positions
	Poisoned bool
	Path     string
}

type StructNode struct {
	Positions
	Poisoned   bool
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
	Positions
	Poisoned bool
	Ord      uint64
	Modifier Modifier
	Name     string
	Type     TypeNode
}

type EnumNode struct {
	Positions
	Poisoned bool
	Table    *TypeTable
	Name     string
	Size     uint64
	Cases    []CaseNode
}

type CaseNode struct {
	Positions
	Poisoned bool
	Ord      uint64
	Name     string
}

type UnionNode struct {
	Positions
	Poisoned   bool
	Table      *TypeTable
	Name       string
	Size       uint64
	Options    []OptionNode
	TypeParams []string
	LocalDefs  []Node
}

type TypeVal struct {
	Iden      string
	Primitive bool
}

type OptionNode struct {
	Positions
	Poisoned bool
	Ord      uint64
	Type     TypeVal
}

type ServiceNode struct {
	Positions
	Poisoned   bool
	Table      *TypeTable
	Name       string
	Procedures []RpcNode
	LocalDefs  []Node
}

type RpcNode struct {
	Positions
	Poisoned bool
	Ord      uint64
	Name     string
	Arg      TypeNode
	Ret      TypeNode
}

type TypeNode struct {
	Positions
	TypeVal
	TypeArgs []TypeNode
	Array    []uint64
}

func (node *PropertyNode) Kind() NodeKind { return PropertyNodeKind }
func (node *ImportNode) Kind() NodeKind   { return ImportNodeKind }
func (node *StructNode) Kind() NodeKind   { return StructNodeKind }
func (node *EnumNode) Kind() NodeKind     { return EnumNodeKind }
func (node *UnionNode) Kind() NodeKind    { return UnionNodeKind }
func (node *ServiceNode) Kind() NodeKind  { return ServiceNodeKind }
func (node *FieldNode) Kind() NodeKind    { return FieldNodeKind }
func (node *OptionNode) Kind() NodeKind   { return OptionNodeKind }
func (node *CaseNode) Kind() NodeKind     { return CaseNodeKind }
func (node *RpcNode) Kind() NodeKind      { return RpcNodeKind }
func (node *TypeNode) Kind() NodeKind     { return TypeNodeKind }

func (r *Positions) Begin() int { return r.B }
func (r *Positions) End() int   { return r.E }

func (r *Positions) Header() string {
	if r.B == r.E {
		return fmt.Sprintf("%d:", r.B)
	} else {
		return fmt.Sprintf("%d:%d:", r.B, r.E)
	}
}

func (r *Positions) Clear() {
	r.E = 0
	r.B = 0
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

func ClearAll(nodes []Node) {
	WalkMetaList(Node.Clear, nodes)
}
