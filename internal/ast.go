package internal

import "fmt"

type NodeKind int

const (
	NoNodeKind NodeKind = iota
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
	case NoNodeKind:
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

type Positions struct {
	B int
	E int
}

func (r *Positions) Offset() string {
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

type DefNode struct {
	Positions
	Kind       NodeKind
	Poisoned   bool
	Iden       string
	Value      string
	TypeTable  *TypeTable
	Members    []MembNode
	TypeParams []string
	LocalDefs  []DefNode
	Size       uint64
}

func (n *DefNode) MemberKind() NodeKind {
	switch n.Kind {
	case StructNodeKind:
		return FieldNodeKind
	case UnionNodeKind:
		return OptionNodeKind
	case EnumNodeKind:
		return CaseNodeKind
	case ServiceNodeKind:
		return RpcNodeKind
	}
	return NoNodeKind
}

type MembNode struct {
	Positions
	Poisoned bool
	Ord      uint64
	Iden     string
	Modifier Modifier
	LType    TypeNode
	RType    TypeNode
	TypeIden string
}

type TypeNode struct {
	Positions
	Value    Type
	Iden     string
	TypeArgs []TypeNode
	Array    []uint64
}
