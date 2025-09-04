package internal

import (
	"slices"
)

type Transformer struct {
	errs *[]error
}

func makeTransformer(errs *[]error) Transformer {
	return Transformer{errs: errs}
}

func (t *Transformer) emitError(err error) {
	*t.errs = append(*t.errs, err)
}

func (t *Transformer) transformNodeList(nodes []DefNode, prev *TypeTable) {
	table := makeTypeTable(prev)
	for i := range nodes {
		node := &nodes[i]
		if node.Kind != StructNodeKind && node.Kind != UnionNodeKind && node.Kind != EnumNodeKind && node.Kind != ServiceNodeKind {
			continue
		}

		if err := table.insert(node.Iden, node); err != nil {
			t.emitError(err)
		}
		node.TypeTable = table

		mKind := node.MemberKind()
		for i := range node.Members {
			node := &node.Members[i]
			switch mKind {
			case FieldNodeKind, OptionNodeKind:
				node.LType.Value = makeType(node.LType.Iden)
			case RpcNodeKind:
				node.LType.Value = makeType(node.LType.Iden)
				node.RType.Value = makeType(node.RType.Iden)
			}
		}

		t.transformNodeList(node.LocalDefs, table)
	}
}

func sortMembers(fields []MembNode) {
	slices.SortFunc(fields, func(n1, n2 MembNode) int { return int(n1.Ord - n2.Ord) })
}

func (t *Transformer) checkMemberOrder(kind NodeKind, nodes []MembNode) {
	expOrd := uint64(1)
	for _, node := range nodes {
		ord := node.Ord
		if ord == expOrd {
			expOrd++
			continue
		}
		err := makeOrdErr(kind, node.Positions, expOrd, ord)
		t.emitError(err)
		break
	}
}

func (t *Transformer) checkDupMembers(kind NodeKind, nodes []MembNode) {
	for i, node := range nodes {
		name := node.Iden
		for j := i - 1; j >= 0; j-- {
			leftName := nodes[j].Iden
			if name != leftName {
				continue
			}
			err := makeRedefErr(kind, node.Positions, name)
			t.emitError(err)
			break
		}
	}
}

func (t *Transformer) checkMemberTypes(kind NodeKind, nodes []MembNode, table *TypeTable) {
	if table == nil {
		return
	}
	for _, node := range nodes {
		checkType := func(typeVal Type) {
			if typeVal.Primitive {
				return
			}
			refNode := table.resolve(typeVal.Iden)
			if refNode == nil {
				err := makeUndefErr(kind, node.Positions, typeVal.Iden)
				t.emitError(err)
				return
			}
		}
		switch kind {
		case FieldNodeKind, OptionNodeKind:
			checkType(node.LType.Value)
		case RpcNodeKind:
			checkType(node.LType.Value)
			checkType(node.RType.Value)
		}
	}
}

func (t *Transformer) validateNodeList(nodes []DefNode) {
	for i := range nodes {
		node := &nodes[i]
		if node.Kind != StructNodeKind && node.Kind != UnionNodeKind && node.Kind != EnumNodeKind && node.Kind != ServiceNodeKind {
			// skip non definition nodes (import and property)
			continue
		}

		mKind := node.MemberKind()
		sortMembers(node.Members)
		t.checkMemberOrder(mKind, node.Members)
		t.checkDupMembers(mKind, node.Members)

		if node.Kind != EnumNodeKind {
			// enum nodes will never have LocalDefs or non-nil Type
			continue
		}

		t.checkMemberTypes(mKind, node.Members, node.TypeTable)
		t.validateNodeList(node.LocalDefs)
	}
}
