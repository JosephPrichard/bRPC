package internal

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// TypeTable is used to look up the node that an identifier may resolve to at any level in the node
type TypeTable struct {
	prev *TypeTable
	m    map[string]Node
}

func (t *TypeTable) getNode(iden string) Node {
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

func prepareTypes(nodes []Node, prev *TypeTable, errs *[]error) {
	table := &TypeTable{m: make(map[string]Node), prev: prev}

	insertTable := func(iden string, node Node) {
		if _, exists := table.m[iden]; exists {
			*errs = append(*errs, makeRedefinedErr(node, iden))
		}
		table.m[iden] = node
	}

	for _, n := range nodes {
		switch node := n.(type) {
		case *StructNode:
			node.Table = table
			for i := range node.Fields {
				convertTypeNode(&node.Fields[i].Type)
			}
			insertTable(node.Name, node)
			prepareTypes(node.LocalDefs, table, errs)
		case *UnionNode:
			node.Table = table
			for i := range node.Options {
				option := &node.Options[i]
				option.Type = convertType(option.Type)
			}
			insertTable(node.Name, node)
			prepareTypes(node.LocalDefs, table, errs)
		case *EnumNode:
			node.Table = table
			insertTable(node.Name, node)
		case *ServiceNode:
			node.Table = table
			for i := range node.Procedures {
				proc := &node.Procedures[i]
				convertTypeNode(&proc.Arg)
				convertTypeNode(&proc.Ret)
			}
			insertTable(node.Name, node)
			prepareTypes(node.LocalDefs, table, errs)
		}
	}
}

var IntSizes = []int{8, 16, 32, 64}

func convertTypeNode(node *TypeNode) {
	node.TypeVal = convertType(node.TypeVal)
}

func convertType(t TypeVal) TypeVal {
	// attempt to convert to a primitive first, otherwise keep the type as itself
	index := strings.Index(t.Iden, "int")
	if index < 0 {
		return t
	}
	bitsStr := t.Iden[index+2:]
	bits, err := strconv.Atoi(bitsStr)
	if err != nil {
		return t
	}

	// map to a fix sized primitive, or a big integer if that is not possible (>= 64 bits)
	for _, size := range IntSizes {
		if bits <= size {
			iden := fmt.Sprintf("int%d", size)
			return TypeVal{Iden: iden, Primitive: true}
		}
	}
	return TypeVal{Iden: "big.Int", Primitive: true}
}

type iterOrdElem struct {
	ord  uint64
	node Node
}

func checkNodeOrder(errs *[]error, count int, elemAt func(int) iterOrdElem) {
	prev := uint64(0)
	for i := range count {
		e := elemAt(i)
		var err error
		if i == 0 && e.ord != 1 {
			err = makeFirstOrdErr(e.node)
		} else if e.ord != prev+1 {
			err = makeOrdErr(e.node, e.ord, prev+1)
		}
		if err != nil {
			*errs = append(*errs, err)
			break
		}
		prev++
	}
}

func checkFieldOrder(node *StructNode, errs *[]error) {
	fields := node.Fields
	slices.SortFunc(fields, func(n1, n2 FieldNode) int { return cmpOrd(n1.Ord, n2.Ord) })
	checkNodeOrder(errs, len(fields), func(i int) iterOrdElem { return iterOrdElem{ord: fields[i].Ord, node: &fields[i]} })
}

func checkUnionOrder(node *UnionNode, errs *[]error) {
	options := node.Options
	slices.SortFunc(options, func(n1, n2 OptionNode) int { return cmpOrd(n1.Ord, n2.Ord) })
	checkNodeOrder(errs, len(options), func(i int) iterOrdElem { return iterOrdElem{ord: options[i].Ord, node: &options[i]} })
}

func checkEnumOrder(node *EnumNode, errs *[]error) {
	cases := node.Cases
	slices.SortFunc(cases, func(n1, n2 CaseNode) int { return cmpOrd(n1.Ord, n2.Ord) })
	checkNodeOrder(errs, len(cases), func(i int) iterOrdElem { return iterOrdElem{ord: cases[i].Ord, node: &cases[i]} })
}

func checkProcOrder(node *ServiceNode, errs *[]error) {
	procs := node.Procedures
	slices.SortFunc(procs, func(n1, n2 RpcNode) int { return cmpOrd(n1.Ord, n2.Ord) })
	checkNodeOrder(errs, len(procs), func(i int) iterOrdElem { return iterOrdElem{ord: procs[i].Ord, node: &procs[i]} })
}

func validateTypes(nodes []Node, errs *[]error) {
	for _, n := range nodes {
		switch node := n.(type) {
		case *StructNode:
			checkFieldOrder(node, errs)
		case *UnionNode:
			checkUnionOrder(node, errs)
		case *EnumNode:
			checkEnumOrder(node, errs)
		case *ServiceNode:
			checkProcOrder(node, errs)
		}
	}
}

type PropTable = map[string]string

func makePropTable(nodes []Node) PropTable {
	propTable := make(PropTable)
	for _, n := range nodes {
		node, ok := n.(*PropertyNode)
		if !ok || node.Poisoned {
			continue
		}
		propTable[node.Name] = node.Value
	}
	return propTable
}

type ImportTable = map[string]Node

func makeImportTable() ImportTable {
	importTable := make(ImportTable)
	return importTable
}

type CodeBuilder struct {
	sb          strings.Builder
	propTable   PropTable
	importTable ImportTable
	errs        *[]error
}

func makeCodeBuilder(propTable PropTable, errs *[]error) CodeBuilder {
	return CodeBuilder{propTable: propTable, errs: errs}
}

func (b *CodeBuilder) build(n Node) {
	switch node := n.(type) {
	case *StructNode:
		b.buildStruct(node)
	case *UnionNode:
		b.buildUnion(node)
	case *EnumNode:
		b.buildEnum(node)
	case *ServiceNode:
		b.buildService(node)
	case *ImportNode, *PropertyNode:
		// noop: all property and import asts should be resolved already
	default:
		// this will fail for field and rpc asts, we assume they are handled in the builder for structs and services
		panic(fmt.Sprintf("unsupported: build call for n type is not implemented: %T", node))
	}
}

// write this operation is common enough to extract it out to a utility function
func (b *CodeBuilder) w(s string) {
	b.sb.WriteString(s)
}

func (b *CodeBuilder) buildTypeRef(ref TypeNode) {
	for _, size := range ref.Array {
		b.w("[")
		if size > 0 {
			b.w(strconv.FormatUint(size, 10))
		}
		b.w("]")
	}
	b.w(ref.Iden)
}

func (b *CodeBuilder) buildStruct(strct *StructNode) {
	if strct.Poisoned {
		return
	}

	// build out the struct type definition
	b.w("type ")
	b.w(strct.Name)
	b.w(" struct {\n")
	for _, field := range strct.Fields {
		b.w(field.Name)
		b.buildTypeRef(field.Type)
	}
	b.w("}\n\n")

	// build out the struct's serialize and deserialize methods
}

func (b *CodeBuilder) buildUnion(union *UnionNode) {
	if union.Poisoned {
		return
	}

	// build out the union type definition
	b.w("type ")
	b.w(union.Name)
	b.w(" interface {\n")
	for _, option := range union.Options {
		b.w(option.Type.Iden)
		b.w("() *")
		b.w(option.Type.Iden)
		b.w("\n")
	}
	b.w("}\n")

	// each union case is a struct that contains the option as a single field
	for _, option := range union.Options {
		b.w("type ")
		b.w(union.Name)
		b.w(option.Type.Iden)
		b.w("Option struct {\n")
		b.w(option.Type.Iden)
		b.w("\n")
		b.w("}\n\n")
	}

	// build out the union's serialize and deserialize methods
}

func (b *CodeBuilder) buildEnum(enum *EnumNode) {
	if enum.Poisoned {
		return
	}

	// build out the enum type definition and cases
	b.w("type ")
	b.w(enum.Name)
	b.w(" int\n\n")
	b.w("const (\n")
	for i, c := range enum.Cases {
		b.w(enum.Name)
		b.w("_")
		b.w(c.Name)
		if i == 0 {
			b.w(" ")
			b.w(enum.Name)
			b.w(" = iota")
		}
		b.w("\n")
	}
	b.w(")\n\n")

	// build out the enum's serialize and deserialize methods
}

func (b *CodeBuilder) buildService(svc *ServiceNode) {
	if svc.Poisoned {
		return
	}
}

func (b *CodeBuilder) buildPackage(pack string) {
	b.w("package ")
	b.w(pack)
	b.w("\n\n")
}

func runCodeBuilder(program string, pack string, errs *[]error) string {
	nodes := runParser(program, errs)

	prepareTypes(nodes, nil, errs)
	validateTypes(nodes, errs)

	if len(*errs) > 0 {
		return ""
	}

	propTable := makePropTable(nodes)

	builder := makeCodeBuilder(propTable, errs)
	builder.buildPackage(pack)
	for _, node := range nodes {
		builder.build(node)
	}

	return builder.sb.String()
}
