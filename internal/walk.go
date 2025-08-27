package internal

func WalkMeta(visit func(Node), n Node) {
	if n == nil {
		return
	}
	visit(n)
	switch node := n.(type) {
	case *StructNode:
		WalkMetaList(visit, node.LocalDefs)
		for _, field := range node.Fields {
			WalkMeta(visit, field)

		}
	case *UnionNode:
		WalkMetaList(visit, node.LocalDefs)
		for _, option := range node.Options {
			WalkMeta(visit, option)
		}
	case *ServiceNode:
		WalkMetaList(visit, node.LocalDefs)
		for _, proc := range node.Procedures {
			WalkMeta(visit, proc)
		}
	case *RpcNode:
		WalkMeta(visit, &node.Arg)
		WalkMeta(visit, &node.Ret)
	case *FieldNode:
		WalkMeta(visit, &node.Type)
	}
}

func WalkMetaList(visit func(Node), nodes []Node) {
	for _, node := range nodes {
		WalkMeta(visit, node)
	}
}
