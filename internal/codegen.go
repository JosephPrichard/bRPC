package internal

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type PropTable = map[string]string

type ImportTable = map[string]Node

type CodeBuilder struct {
	sb          strings.Builder
	propTable   PropTable
	importTable ImportTable
	errs        *[]error
}

func makeCodeBuilder(errs *[]error) CodeBuilder {
	return CodeBuilder{errs: errs}
}

func (b *CodeBuilder) buildPropTable(nodes []Node) {
	b.propTable = make(PropTable)
	for _, n := range nodes {
		node, ok := n.(*PropertyNode)
		if !ok || node.Poisoned {
			continue
		}
		b.propTable[node.Name] = node.Value
	}
}

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

func (b *CodeBuilder) prepareTypes(nodes []Node, prev *TypeTable) {
	table := &TypeTable{m: make(map[string]Node), prev: prev}

	insertTable := func(iden string, node Node) {
		_, exists := table.m[iden]
		if exists {
			err := makeRedefinedErr(node, iden)
			*b.errs = append(*b.errs, err)
			return
		}
		table.m[iden] = node
	}

	for _, n := range nodes {
		switch node := n.(type) {
		case *StructNode:
			for _, field := range node.Fields {
				field.Type.RType = makeRType(field.Type.Iden)
			}
			node.Table = table
			insertTable(node.Name, node)
			b.prepareTypes(node.LocalDefs, table)
		case *UnionNode:
			for _, option := range node.Options {
				option.RType = makeRType(option.Iden)
			}
			node.Table = table
			insertTable(node.Name, node)
			b.prepareTypes(node.LocalDefs, table)
		case *EnumNode:
			node.Table = table
			insertTable(node.Name, node)
		case *ServiceNode:
			for _, rpc := range node.Procedures {
				rpc.Arg.RType = makeRType(rpc.Arg.Iden)
				rpc.Ret.RType = makeRType(rpc.Ret.Iden)
			}
			node.Table = table
			insertTable(node.Name, node)
			b.prepareTypes(node.LocalDefs, table)
		}
	}
}

func checkNodeOrder[N Node](nodes []N, errs *[]error) {
	prev := uint64(0)
	for i, node := range nodes {
		ord := node.Order()
		var err error
		if i == 0 && ord != 1 {
			err = makeFstOrdErr(node)
		} else if ord != prev+1 {
			err = makeOrdErr(node, ord, prev+1)
		}
		if err != nil {
			*errs = append(*errs, err)
			break
		}
		prev++
	}
}

func (b *CodeBuilder) validateTypes(nodes []Node) {
	for _, n := range nodes {
		switch node := n.(type) {
		case *StructNode:
			slices.SortFunc(node.Fields, cmpFieldOrd)
			checkNodeOrder(node.Fields, b.errs)
		case *UnionNode:
			slices.SortFunc(node.Options, cmpOptionOrd)
			checkNodeOrder(node.Options, b.errs)
		case *EnumNode:
			slices.SortFunc(node.Cases, cmpCaseOrd)
			checkNodeOrder(node.Cases, b.errs)
		case *ServiceNode:
			slices.SortFunc(node.Procedures, cmpRpcOrd)
			checkNodeOrder(node.Procedures, b.errs)
		}
	}
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

// this operation is common enough to extract it out to a utility function
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
	b.w(ref.RType.Iden)
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
		b.w("\t")
		b.w(field.Name)
		b.w("\t")
		b.buildTypeRef(field.Type)
		b.w("\n")
	}
	b.w("}\n\n")

	// build out the struct's serialize and deserialize methods
}

func (b *CodeBuilder) buildUnion(union *UnionNode) {
	if union.Poisoned {
		return
	}

	writeOptionStruct := func(option *OptionNode) {
		b.w(union.Name)
		b.w(option.RType.Iden)
		b.w("Opt")
	}

	// build out the union type definition
	b.w("type ")
	b.w(union.Name)
	b.w(" interface {\n")
	for _, option := range union.Options {
		b.w("\t")
		b.w(option.RType.Iden)
		b.w("() *")
		writeOptionStruct(option)
		b.w("\n")
	}
	b.w("}\n\n")

	defaultName := union.Name + "Unimplemented"

	// each union option contains a default implementation for each of the interface methods
	b.w("type ")
	b.w(defaultName)
	b.w(" struct {}\n\n")
	for _, option := range union.Options {
		b.w("func (_ *")
		b.w(defaultName)
		b.w(") ")
		b.w(option.RType.Iden)
		b.w("() *")
		writeOptionStruct(option)
		b.w("{ return nil } \n")
	}
	b.w("\n")

	// each union option is a struct that contains the option as a single field
	for _, option := range union.Options {
		b.w("type ")
		writeOptionStruct(option)
		b.w(" struct {\n")
		b.w("\t")
		b.w(option.RType.Iden)
		b.w("\n\t")
		b.w(defaultName)
		b.w("\n}\n\n")

		b.w("func (o *")
		writeOptionStruct(option)
		b.w(") ")
		b.w(option.RType.Iden)
		b.w("() *")
		writeOptionStruct(option)
		b.w(" { return o }\n\n")
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
		b.w("\t")
		b.w(enum.Name)
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

	builder := makeCodeBuilder(errs)

	builder.buildPropTable(nodes)
	builder.prepareTypes(nodes, nil)
	builder.validateTypes(nodes)

	if len(*errs) > 0 {
		return ""
	}

	builder.buildPackage(pack)
	for _, node := range nodes {
		builder.build(node)
	}

	return builder.sb.String()
}
