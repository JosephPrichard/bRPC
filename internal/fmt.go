package internal

import (
	"fmt"
	"strings"
)

func FmtAst(nodes []DefNode) string {
	var sb strings.Builder
	FmtDefList(&sb, nodes, 0)
	return sb.String()
}

func fmtIndents(sb *strings.Builder, depth int) {
	for range depth {
		sb.WriteString("\t")
	}
}

func FmtDefList(sb *strings.Builder, nodes []DefNode, depth int) {
	for _, node := range nodes {
		if depth != 0 && node.Kind.isTypeDecl() {
			sb.WriteString("\n")
		}
		switch node.Kind {
		case ImportNodeKind:
			fmt.Fprintf(sb, "import \"%s\"\n", node.Value)
		case PropertyNodeKind:
			fmt.Fprintf(sb, "%s \"%s\"\n", node.Iden, node.Value)
		case StructNodeKind:
			fmtIndents(sb, depth)
			fmt.Fprintf(sb, "message %s struct ", node.Iden)
			FmtTypeParams(sb, node.TypeParams)
			sb.WriteString("{\n")
			FmtMemberList(sb, node.Kind.MemberKind(), node.Members, depth+1)
			FmtDefList(sb, node.LocalDefs, depth+1)
			fmtIndents(sb, depth)
			sb.WriteString("}\n")
		case UnionNodeKind:
			fmtIndents(sb, depth)
			fmt.Fprintf(sb, "message %s union ", node.Iden)
			FmtTypeParams(sb, node.TypeParams)
			sb.WriteString("{\n")
			FmtMemberList(sb, node.Kind.MemberKind(), node.Members, depth+1)
			FmtDefList(sb, node.LocalDefs, depth+1)
			fmtIndents(sb, depth)
			sb.WriteString("}\n")
		case EnumNodeKind:
			fmtIndents(sb, depth)
			fmt.Fprintf(sb, "message %s enum ", node.Iden)
			FmtTypeParams(sb, node.TypeParams)
			sb.WriteString("{\n")
			FmtMemberList(sb, node.Kind.MemberKind(), node.Members, depth+1)
			fmtIndents(sb, depth)
			sb.WriteString("}\n")
		case ServiceNodeKind:
			fmtIndents(sb, depth)
			fmt.Fprintf(sb, "service %s {\n", node.Iden)
			FmtMemberList(sb, node.Kind.MemberKind(), node.Members, depth+1)
			FmtDefList(sb, node.LocalDefs, depth+1)
			fmtIndents(sb, depth)
			sb.WriteString("}\n")
		}
	}
}

func FmtTypeParams(sb *strings.Builder, typeParams []string) {
	for i, param := range typeParams {
		if i == 0 {
			sb.WriteString("(")
		}
		fmt.Fprintf(sb, "%s", param)
		if i == len(typeParams)-1 {
			sb.WriteString(") ")
		} else {
			sb.WriteString(", ")
		}
	}
}

func FmtType(sb *strings.Builder, node TypeNode) {
	for _, size := range node.Array {
		if size != 0 {
			fmt.Fprintf(sb, "[%d]", size)
		} else {
			sb.WriteString("[]")
		}
	}
	sb.WriteString(node.Iden)
	FmtTypeArgs(sb, node.TypeArgs)
}

func FmtTypeArgs(sb *strings.Builder, typeArgs []TypeNode) {
	for i, arg := range typeArgs {
		if i == 0 {
			sb.WriteString("(")
		}
		FmtType(sb, arg)
		if i == len(typeArgs)-1 {
			sb.WriteString(")")
		} else {
			sb.WriteString(", ")
		}
	}
}

func FmtMemberList(sb *strings.Builder, kind NodeKind, nodes []MemberNode, depth int) {
	for _, node := range nodes {
		switch kind {
		case FieldNodeKind:
			fmtIndents(sb, depth)
			fmt.Fprintf(sb, "%s %s @%d ", node.Modifier, node.Iden, node.Tag)
			FmtType(sb, node.LeftType)
			sb.WriteString(";\n")
		case CaseNodeKind:
			fmtIndents(sb, depth)
			fmt.Fprintf(sb, "@%d %s;\n", node.Tag, node.Iden)
		case OptionNodeKind:
			fmtIndents(sb, depth)
			fmt.Fprintf(sb, "%s @%d ", node.Iden, node.Tag)
			FmtType(sb, node.LeftType)
			sb.WriteString(";\n")
		case RpcNodeKind:
			fmtIndents(sb, depth)
			fmt.Fprintf(sb, "rpc @%d %s(", node.Tag, node.Iden)
			FmtType(sb, node.LeftType)
			sb.WriteString(") returns (")
			FmtType(sb, node.RightType)
			sb.WriteString(");\n")
		}
	}
}

func clearNodeList(nodes []DefNode) {
	for i := range nodes {
		node := &nodes[i]
		node.ClearPositions()
		for i := range node.Members {
			node := &node.Members[i]
			node.ClearPositions()
			node.DefaultValue.ClearPositions()
			clearTypeNode(&node.LeftType)
			clearTypeNode(&node.RightType)
		}
		clearNodeList(node.LocalDefs)
	}
}

func clearTypeNode(node *TypeNode) {
	node.ClearPositions()
	for i := range node.TypeArgs {
		clearTypeNode(&node.TypeArgs[i])
	}
}
