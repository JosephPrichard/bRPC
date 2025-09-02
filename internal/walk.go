package internal

import (
	"fmt"
	"strings"
)

func ClearNode(n Node) {
	if n == nil {
		return
	}
	n.Clear()
	switch node := n.(type) {
	case *StructNode:
		ClearNodeList(node.LocalDefs)
		for _, field := range node.Fields {
			ClearNode(field)
		}
	case *UnionNode:
		ClearNodeList(node.LocalDefs)
		for _, option := range node.Options {
			ClearNode(option)
		}
	case *EnumNode:
		for _, c := range node.Cases {
			ClearNode(c)
		}
	case *ServiceNode:
		ClearNodeList(node.LocalDefs)
		for _, proc := range node.Procedures {
			ClearNode(proc)
		}
	case *RpcNode:
		ClearNode(&node.Arg)
		ClearNode(&node.Ret)
	case *FieldNode:
		ClearNode(&node.Type)
	}
}

func ClearNodeList(nodes []Node) {
	for _, node := range nodes {
		ClearNode(node)
	}
}

func StringifyNode(sb *strings.Builder, n Node, depth int) {
	if n == nil {
		return
	}

	write := func(s string) {
		sb.WriteString(s)
	}
	indents := func() {
		for range depth {
			write("\t")
		}
	}

	nextDepth := depth + 1

	switch node := n.(type) {
	case *ImportNode:
		write(fmt.Sprintf("import \"%s\"\n", node.Path))
	case *PropertyNode:
		write(fmt.Sprintf("%s \"%s\"\n", node.Iden, node.Value))
	case *StructNode:
		indents()
		write(fmt.Sprintf("message %s struct {\n", node.Iden))
		for _, field := range node.Fields {
			StringifyNode(sb, field, nextDepth)
		}
		StringifyNodeList(sb, node.LocalDefs, nextDepth)
		indents()
		write("}\n")
	case *UnionNode:
		indents()
		write(fmt.Sprintf("message %s union {\n", node.Iden))
		for _, option := range node.Options {
			StringifyNode(sb, option, nextDepth)
		}
		StringifyNodeList(sb, node.LocalDefs, nextDepth)
		indents()
		write("}\n")
	case *EnumNode:
		indents()
		write(fmt.Sprintf("message %s enum {\n", node.Iden))
		for _, c := range node.Cases {
			StringifyNode(sb, c, nextDepth)
		}
		indents()
		write("}\n")
	case *FieldNode:
		indents()
		write(fmt.Sprintf("%s %s @%d ", node.Modifier, node.Iden, node.Ord))
		StringifyNode(sb, &node.Type, nextDepth)
		write(";\n")
	case *CaseNode:
		indents()
		write(fmt.Sprintf("@%d %s;\n", node.Ord, node.Iden))
	case *OptionNode:
		indents()
		write(fmt.Sprintf("@%d %s;\n", node.Ord, node.Iden))
	case *ServiceNode:
		indents()
		write(fmt.Sprintf("service %s {\n", node.Iden))
		for _, proc := range node.Procedures {
			StringifyNode(sb, proc, nextDepth)
		}
		StringifyNodeList(sb, node.LocalDefs, nextDepth)
		indents()
		write("}\n")
	case *RpcNode:
		indents()
		write(fmt.Sprintf("rpc @%d %s(", node.Ord, node.Iden))
		StringifyNode(sb, &node.Arg, depth)
		write(") returns (")
		StringifyNode(sb, &node.Ret, depth)
		write(");\n")
	case *TypeNode:
		for _, size := range node.Array {
			if size != 0 {
				write(fmt.Sprintf("[%d]", size))
			} else {
				write("[]")
			}
		}
		write(node.Iden)
		for i, t := range node.TypeArgs {
			if i == 0 {
				write("(")
			}
			StringifyNode(sb, &t, depth)
			if i < len(node.TypeArgs)-1 {
				write(", ")
			} else {
				write(")")
			}
		}
	}
}

func StringifyNodeList(sb *strings.Builder, nodes []Node, depth int) {
	for _, node := range nodes {
		if depth != 0 {
			switch node.(type) {
			case *StructNode, *UnionNode, *EnumNode:
				sb.WriteString("\n")
			}
		}
		StringifyNode(sb, node, depth)
	}
}

func StringifyAst(nodes []Node) string {
	var sb strings.Builder
	StringifyNodeList(&sb, nodes, 0)
	return sb.String()
}
