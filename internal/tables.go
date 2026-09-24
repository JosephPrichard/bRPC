package internal

type TypeParamStack struct {
	prev *TypeParamStack
	m    map[string]*TypeNode
}

func makeTypeParamStack(prev *TypeParamStack) *TypeParamStack {
	return &TypeParamStack{m: make(map[string]*TypeNode), prev: prev}
}

func (t *TypeParamStack) insert(iden string, node *TypeNode) error {
	if _, exists := t.m[iden]; exists {
		return makeRedefErr(TypeNodeKind, node.Positions, iden)
	}
	t.m[iden] = node
	return nil
}

func (t *TypeParamStack) resolve(iden string) *TypeNode {
	table := t
	for table != nil {
		node, ok := table.m[iden]
		if ok {
			return node
		}
		table = table.prev
	}
	return nil
}

type TypeDefStack struct {
	prev *TypeDefStack
	m    map[string]*DefNode
}

func makeTypeDefStack(prev *TypeDefStack) *TypeDefStack {
	return &TypeDefStack{m: make(map[string]*DefNode), prev: prev}
}

func (t *TypeDefStack) insert(iden string, node *DefNode) error {
	if _, exists := t.m[iden]; exists {
		return makeRedefErr(node.Kind, node.Positions, iden)
	}
	t.m[iden] = node
	return nil
}

func (t *TypeDefStack) resolve(iden string) *DefNode {
	table := t
	for table != nil {
		node, ok := table.m[iden]
		if ok {
			return node
		}
		table = table.prev
	}
	return nil
}

// resolving the identifier relative to the definition stack and then local definitions
func resolveIden(node *DefNode, iden string) *DefNode {
	ret := node.DefStack.resolve(iden)
	if ret != nil {
		return ret
	}
	for i := range node.LocalDefs {
		def := &node.LocalDefs[i]
		if def.Iden == iden {
			return def
		}
	}
	return nil
}

type PropTable = map[string]string

func makePropTable(nodes []DefNode) PropTable {
	propTable := make(PropTable)
	for _, node := range nodes {
		if node.Kind != PropertyNodeKind {
			continue
		}
		if !node.Poisoned {
			propTable[node.Iden] = node.Value
		}
	}
	return propTable
}

type ImportTable = map[string][]DefNode

func makeImportTable() ImportTable {
	importTable := make(ImportTable)
	return importTable
}
