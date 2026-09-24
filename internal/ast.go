package internal

import (
	"fmt"
	"slices"
)

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
	Begin int
	End   int
}

func (r *Positions) Offset() string {
	if r.Begin == r.End {
		return fmt.Sprintf("%d:", r.Begin)
	} else {
		return fmt.Sprintf("%d:%d:", r.Begin, r.End)
	}
}

func (r *Positions) ClearPositions() {
	r.End = 0
	r.Begin = 0
}

var DeclNodeKinds = []NodeKind{StructNodeKind, UnionNodeKind, EnumNodeKind, ServiceNodeKind}

func (k NodeKind) isTypeDecl() bool {
	return slices.Contains(DeclNodeKinds, k)
}

func (k NodeKind) MemberKind() NodeKind {
	switch k {
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

type DefNode struct {
	Positions
	Kind       NodeKind
	Poisoned   bool
	Iden       string
	Value      string
	DefStack   *TypeDefStack
	Members    []MembNode
	TypeParams []string
	LocalDefs  []DefNode
	Size       uint64
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
