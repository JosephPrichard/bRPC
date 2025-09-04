package internal

type TypeTable struct {
	prev *TypeTable
	m    map[string]*DefNode
}

func makeTypeTable(prev *TypeTable) *TypeTable {
	return &TypeTable{m: make(map[string]*DefNode), prev: prev}
}

func (t *TypeTable) insert(iden string, node *DefNode) error {
	if _, exists := t.m[iden]; exists {
		return makeRedefErr(node.Kind, node.Positions, iden)
	}
	t.m[iden] = node
	return nil
}

func (t *TypeTable) resolve(iden string) *DefNode {
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