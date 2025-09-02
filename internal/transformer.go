package internal

import (
	"slices"
)

type PropTable = map[string]string

type ImportTable = map[string]Node

type Transformer struct {
	propTable   PropTable
	importTable ImportTable
	errs        *[]error
}

func makeTransformer(errs *[]error) Transformer {
	return Transformer{errs: errs}
}

func (t *Transformer) emitError(err error) {
	*t.errs = append(*t.errs, err)
}

func (t *Transformer) makePropTable(nodes []Node) {
	t.propTable = make(PropTable)
	for _, n := range nodes {
		node, ok := n.(*PropertyNode)
		if !ok {
			continue
		}
		if !node.Poisoned {
			t.propTable[node.Iden] = node.Value
		}
	}
}

func (t *Transformer) transformTypes(nodes []Node, prev *TypeTable) {
	table := &TypeTable{m: make(map[string]Node), prev: prev}

	insertTable := func(iden string, node Node) {
		if _, exists := table.m[iden]; exists {
			t.emitError(makeRedefErr(node, iden))
			return
		}
		table.m[iden] = node
	}

	makeFieldTypes := func(node *StructNode) {
		for _, field := range node.Fields {
			field.Type.Type = makeType(field.Type.Iden)
		}
	}

	makeUnionTypes := func(node *UnionNode) {
		for _, option := range node.Options {
			option.Type = makeType(option.Iden)
		}
	}

	makeSvcTypes := func(node *ServiceNode) {
		for _, proc := range node.Procedures {
			proc.Arg.Type = makeType(proc.Arg.Iden)
			proc.Ret.Type = makeType(proc.Ret.Iden)
		}
	}

	for _, n := range nodes {
		switch node := n.(type) {
		case *StructNode:
			node.TypeTable = table
			makeFieldTypes(node)
			insertTable(node.Iden, node)
			t.transformTypes(node.LocalDefs, table)
		case *UnionNode:
			node.TypeTable = table
			insertTable(node.Iden, node)
			makeUnionTypes(node)
			t.transformTypes(node.LocalDefs, table)
		case *EnumNode:
			node.TypeTable = table
			insertTable(node.Iden, node)
		case *ServiceNode:
			node.TypeTable = table
			insertTable(node.Iden, node)
			makeSvcTypes(node)
			t.transformTypes(node.LocalDefs, table)
		}
	}
}

func sortFields(fields []*FieldNode) {
	slices.SortFunc(fields, func(n1, n2 *FieldNode) int { return int(n1.Ord - n2.Ord) })
}

func sortOptions(options []*OptionNode) {
	slices.SortFunc(options, func(n1, n2 *OptionNode) int { return int(n1.Ord - n2.Ord) })
}

func sortCases(cases []*CaseNode) {
	slices.SortFunc(cases, func(n1, n2 *CaseNode) int { return int(n1.Ord - n2.Ord) })
}

func sortProcedures(procs []*RpcNode) {
	slices.SortFunc(procs, func(n1, n2 *RpcNode) int { return int(n1.Ord - n2.Ord) })
}

func checkNodeOrder[M Member](nodes []M, f func(error)) {
	expOrd := uint64(1)
	for _, node := range nodes {
		ord := node.Order()
		var err error
		if ord != expOrd {
			err = makeOrdErr(node, expOrd, ord)
		}
		if err != nil {
			f(err)
			break
		}
		expOrd++
	}
}

func checkDupNodes[M Member](nodes []M, f func(error)) {
	for i, node := range nodes {
		for j := i - 1; j >= 0; j-- {
			n := nodes[j]
			if node.Name() == n.Name() {
				f(makeRedefErr(node, node.Name()))
				break
			}
		}
	}
}

func checkNodeTypes[M TypedMember](table *TypeTable, nodes []M, f func(error)) {
	for _, node := range nodes {
		visitType := func(typ Type) {
			if typ.Primitive {
				return
			}
			refNode := table.resolve(typ.TIden)
			if refNode == nil {
				f(makeUndefErr(node, typ.TIden))
				return
			}
		}
		node.Types(visitType)
	}
}

func (t *Transformer) validateTypes(nodes []Node) {
	for _, n := range nodes {
		switch node := n.(type) {
		case *StructNode:
			sortFields(node.Fields)
			checkNodeOrder(node.Fields, t.emitError)
			checkDupNodes(node.Fields, t.emitError)
			checkNodeTypes(node.TypeTable, node.Fields, t.emitError)
			t.validateTypes(node.LocalDefs)
		case *UnionNode:
			sortOptions(node.Options)
			checkNodeOrder(node.Options, t.emitError)
			checkDupNodes(node.Options, t.emitError)
			checkNodeTypes(node.TypeTable, node.Options, t.emitError)
			t.validateTypes(node.LocalDefs)
		case *EnumNode:
			sortCases(node.Cases)
			checkNodeOrder(node.Cases, t.emitError)
			checkDupNodes(node.Cases, t.emitError)
		case *ServiceNode:
			sortProcedures(node.Procedures)
			checkNodeOrder(node.Procedures, t.emitError)
			checkDupNodes(node.Procedures, t.emitError)
			checkNodeTypes(node.TypeTable, node.Procedures, t.emitError)
			t.validateTypes(node.LocalDefs)
		}
	}
}
