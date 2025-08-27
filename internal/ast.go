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

// Node a node represents a recursive ast node
type Node interface {
	Kind() NodeKind
	Begin() int
	End() int
	Header() string
	Clear()
	Order() uint64
}

type Ordered struct {
	Ord uint64
}

func (o Ordered) Order() uint64 {
	return o.Ord
}

type Unordered struct{}

func (o Unordered) Order() uint64 {
	return 0
}

type Positions struct {
	B int
	E int
}

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

type PropertyNode struct {
	Positions
	Unordered
	Poisoned bool
	Name     string
	Value    string
}

type ImportNode struct {
	Positions
	Unordered
	Poisoned bool
	Path     string
}

type StructNode struct {
	Positions
	Unordered
	Poisoned   bool
	Table      *TypeTable
	Name       string
	Fields     []*FieldNode
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
	Ordered
	Poisoned bool
	Modifier Modifier
	Name     string
	Type     TypeNode
}

type EnumNode struct {
	Positions
	Unordered
	Poisoned bool
	Table    *TypeTable
	Name     string
	Size     uint64
	Cases    []*CaseNode
}

type CaseNode struct {
	Positions
	Ordered
	Poisoned bool
	Name     string
}

type UnionNode struct {
	Positions
	Unordered
	Poisoned   bool
	Table      *TypeTable
	Name       string
	Size       uint64
	Options    []*OptionNode
	TypeParams []string
	LocalDefs  []Node
}

type OptionNode struct {
	Positions
	Ordered
	Poisoned bool
	Iden     string
	RType    RType
}

type ServiceNode struct {
	Positions
	Unordered
	Poisoned   bool
	Table      *TypeTable
	Name       string
	Procedures []*RpcNode
	LocalDefs  []Node
}

type RpcNode struct {
	Positions
	Ordered
	Poisoned bool
	Name     string
	Arg      TypeNode
	Ret      TypeNode
}

type TypeNode struct {
	Positions
	Unordered
	Iden     string
	TypeArgs []TypeNode
	Array    []uint64
	RType    RType
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

func cmpOrd(o1 uint64, o2 uint64) int     { return int(o1 - o2) }
func cmpFieldOrd(n1, n2 *FieldNode) int   { return cmpOrd(n1.Ord, n2.Ord) }
func cmpOptionOrd(n1, n2 *OptionNode) int { return cmpOrd(n1.Ord, n2.Ord) }
func cmpCaseOrd(n1, n2 *CaseNode) int     { return cmpOrd(n1.Ord, n2.Ord) }
func cmpRpcOrd(n1, n2 *RpcNode) int       { return cmpOrd(n1.Ord, n2.Ord) }
