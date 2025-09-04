package internal

import (
	"fmt"
	"strings"
)

func ClearNodeList(nodes []DefNode) {
	for i := range nodes {
		node := &nodes[i]
		node.Clear()
		for i := range node.Members {
			node := &node.Members[i]
			node.Clear()
			ClearTypeNode(&node.LType)
			ClearTypeNode(&node.RType)
		}
		ClearNodeList(node.LocalDefs)
	}
}

func ClearTypeNode(node *TypeNode) {
	node.Clear()
	for i := range node.TypeArgs {
		ClearTypeNode(&node.TypeArgs[i])
	}
}

func WriteAst(nodes []DefNode) string {
	var sb strings.Builder
	WriteNodeList(&sb, nodes, 0)
	return sb.String()
}

func writeIndents(sb *strings.Builder, depth int) {
	for range depth {
		sb.WriteString("\t")
	}
}

func WriteNodeList(sb *strings.Builder, nodes []DefNode, depth int) {
	for _, node := range nodes {
		if depth != 0 && (node.Kind == StructNodeKind || node.Kind == UnionNodeKind || node.Kind == EnumNodeKind) {
			sb.WriteString("\n")
		}
		switch node.Kind {
		case ImportNodeKind:
			fmt.Fprintf(sb, "import \"%s\"\n", node.Value)
		case PropertyNodeKind:
			fmt.Fprintf(sb, "%s \"%s\"\n", node.Iden, node.Value)
		case StructNodeKind:
			writeIndents(sb, depth)
			fmt.Fprintf(sb, "message %s struct {\n", node.Iden)
			WriteMemberList(sb, node.MemberKind(), node.Members, depth+1)
			WriteNodeList(sb, node.LocalDefs, depth+1)
			writeIndents(sb, depth)
			sb.WriteString("}\n")
		case UnionNodeKind:
			writeIndents(sb, depth)
			fmt.Fprintf(sb, "message %s union {\n", node.Iden)
			WriteMemberList(sb, node.MemberKind(), node.Members, depth+1)
			WriteNodeList(sb, node.LocalDefs, depth+1)
			writeIndents(sb, depth)
			sb.WriteString("}\n")
		case EnumNodeKind:
			writeIndents(sb, depth)
			fmt.Fprintf(sb, "message %s enum {\n", node.Iden)
			WriteMemberList(sb, node.MemberKind(), node.Members, depth+1)
			writeIndents(sb, depth)
			sb.WriteString("}\n")
		case ServiceNodeKind:
			writeIndents(sb, depth)
			fmt.Fprintf(sb, "service %s {\n", node.Iden)
			WriteMemberList(sb, node.MemberKind(), node.Members, depth+1)
			WriteNodeList(sb, node.LocalDefs, depth+1)
			writeIndents(sb, depth)
			sb.WriteString("}\n")
		}
	}
}

func WriteType(sb *strings.Builder, node TypeNode) {
	for _, size := range node.Array {
		if size != 0 {
			fmt.Fprintf(sb, "[%d]", size)
		} else {
			sb.WriteString("[]")
		}
	}
	sb.WriteString(node.Iden)
	for i, t := range node.TypeArgs {
		if i == 0 {
			sb.WriteString("(")
		}
		WriteType(sb, t)
		if i < len(node.TypeArgs)-1 {
			sb.WriteString(", ")
		} else {
			sb.WriteString(")")
		}
	}
}

func WriteMemberList(sb *strings.Builder, kind NodeKind, nodes []MembNode, depth int) {
	for _, node := range nodes {
		switch kind {
		case FieldNodeKind:
			writeIndents(sb, depth)
			fmt.Fprintf(sb, "%s %s @%d ", node.Modifier, node.Iden, node.Ord)
			WriteType(sb, node.LType)
			sb.WriteString(";\n")
		case CaseNodeKind:
			writeIndents(sb, depth)
			fmt.Fprintf(sb, "@%d %s;\n", node.Ord, node.Iden)
		case OptionNodeKind:
			writeIndents(sb, depth)
			fmt.Fprintf(sb, "%s @%d ", node.Iden, node.Ord)
			WriteType(sb, node.LType)
			sb.WriteString(";\n")
		case RpcNodeKind:
			writeIndents(sb, depth)
			fmt.Fprintf(sb, "rpc @%d %s(", node.Ord, node.Iden)
			WriteType(sb, node.LType)
			sb.WriteString(") returns (")
			WriteType(sb, node.RType)
			sb.WriteString(");\n")
		}
	}
}
