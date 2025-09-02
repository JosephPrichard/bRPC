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

type Node interface {
	Kind() NodeKind
	Begin() int
	End() int
	Header() string
	Clear()
}

type Member interface {
	Node
	Name() string
	Order() uint64
}

type TypedMember interface {
	Member
	Types(func(Type))
}

type PropertyNode struct {
	Positions
	Iden     string
	Poisoned bool
	Value    string
}

type ImportNode struct {
	Positions
	Poisoned bool
	Path     string
}

type StructNode struct {
	Positions
	Iden       string
	TypeTable  *TypeTable
	Poisoned   bool
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

func (m Modifier) String() string {
	var modStr string
	switch m {
	case Required:
		modStr = "required"
	case Optional:
		modStr = "optional"
	case Deprecated:
		modStr = "deprecated"
	}
	return modStr
}

type FieldNode struct {
	Positions
	Ord      uint64
	Iden     string
	Poisoned bool
	Modifier Modifier
	Type     TypeNode
}

type EnumNode struct {
	Positions
	Iden      string
	TypeTable *TypeTable
	Poisoned  bool
	Size      uint64
	Cases     []*CaseNode
}

type CaseNode struct {
	Positions
	Ord      uint64
	Iden     string
	Poisoned bool
}

type UnionNode struct {
	Positions
	Iden       string
	TypeTable  *TypeTable
	Poisoned   bool
	Size       uint64
	Options    []*OptionNode
	TypeParams []string
	LocalDefs  []Node
}

type OptionNode struct {
	Positions
	Type
	Ord      uint64
	Iden     string
	Poisoned bool
}

type ServiceNode struct {
	Positions
	Iden       string
	TypeTable  *TypeTable
	Poisoned   bool
	Procedures []*RpcNode
	LocalDefs  []Node
}

type RpcNode struct {
	Positions
	Ord      uint64
	Iden     string
	Poisoned bool
	Arg      TypeNode
	Ret      TypeNode
}

type TypeNode struct {
	Positions
	Type
	Iden     string
	TypeArgs []TypeNode
	Array    []uint64
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

func (n *FieldNode) Name() string  { return n.Iden }
func (n *OptionNode) Name() string { return n.Iden }
func (n *CaseNode) Name() string   { return n.Iden }
func (n *RpcNode) Name() string    { return n.Iden }

func (n *FieldNode) Order() uint64  { return n.Ord }
func (n *OptionNode) Order() uint64 { return n.Ord }
func (n *CaseNode) Order() uint64   { return n.Ord }
func (n *RpcNode) Order() uint64    { return n.Ord }

func (n *FieldNode) Types(f func(Type)) {
	f(n.Type.Type)
}
func (n *OptionNode) Types(f func(Type)) {
	f(n.Type)
}
func (n *RpcNode) Types(f func(Type)) {
	f(n.Arg.Type)
	f(n.Ret.Type)
}

func (n *StructNode) Table() *TypeTable  { return n.TypeTable }
func (n *UnionNode) Table() *TypeTable   { return n.TypeTable }
func (n *EnumNode) Table() *TypeTable    { return n.TypeTable }
func (n *ServiceNode) Table() *TypeTable { return n.TypeTable }

func (n *PropertyNode) Kind() NodeKind { return PropertyNodeKind }
func (n *ImportNode) Kind() NodeKind   { return ImportNodeKind }
func (n *StructNode) Kind() NodeKind   { return StructNodeKind }
func (n *EnumNode) Kind() NodeKind     { return EnumNodeKind }
func (n *UnionNode) Kind() NodeKind    { return UnionNodeKind }
func (n *ServiceNode) Kind() NodeKind  { return ServiceNodeKind }
func (n *FieldNode) Kind() NodeKind    { return FieldNodeKind }
func (n *OptionNode) Kind() NodeKind   { return OptionNodeKind }
func (n *CaseNode) Kind() NodeKind     { return CaseNodeKind }
func (n *RpcNode) Kind() NodeKind      { return RpcNodeKind }
func (n *TypeNode) Kind() NodeKind     { return TypeNodeKind }
