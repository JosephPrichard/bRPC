package internal

import (
	"fmt"
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

// Pass One
// 1: prepares symbol tables (later used for type args)
// 2: check for redefined types
// 3: lower IDL types into golang types
func prepareTypeTables(nodes []Node, prev *TypeTable, errs *[]error) {
	table := &TypeTable{m: make(map[string]Node), prev: prev}

	insertTable := func(iden string, node Node) {
		_, exists := table.m[iden]
		// 2: redefined types lead to errors, but shadowing is legal
		if exists {
			*errs = append(*errs, &CodegenErr{node: node, msg: fmt.Sprintf("\"%s\" is redefined", iden)})
		}
		// 1: this may cause cycles if a child ast refers to a parent ast - this must be detected in a future pass
		table.m[iden] = node
	}

	for _, n := range nodes {
		n.SetTable(table)

		switch node := n.(type) {
		case *StructNode:
			// 3: lowering fields
			node.IterStruct(func(node *FieldNode) {
				lowerType(&node.Type)
			})
			// 1,2: preparing tables
			insertTable(node.Name, node)
			// recursive call for nested definitions
			prepareTypeTables(node.LocalDefs, table, errs)
		case *UnionNode:
			// 3: lowering options
			node.IterOptions(func(node *OptionNode) {
				lowerType(&node.Type)
			})
			// 1,2: preparing tables
			insertTable(node.Name, node)
			// recursive call for nested definitions
			prepareTypeTables(node.LocalDefs, table, errs)
		case *EnumNode:
			insertTable(node.Name, node)
		case *ServiceNode:
			// 3: lowering procedures
			node.IterProcedures(func(node *RpcNode) {
				lowerType(&node.Arg)
				lowerType(&node.Ret)
			})
			// 1,2: preparing tables
			insertTable(node.Name, node)
			// recursive call for nested definitions
			prepareTypeTables(node.LocalDefs, table, errs)
		}
	}
}

var BitLevels = []int{8, 16, 32, 64}

func lowerType(node *TypeRefNode) {
	// try to lower the iden to a primitive, or just lower it into itself
	index := strings.Index(node.Iden, "int")
	if index < 0 {
		return
	}
	bitsStr := node.Iden[index+2:]
	bits, err := strconv.Atoi(bitsStr)
	if err != nil {
		return
	}

	for _, bitLevel := range BitLevels {
		if bits <= bitLevel {
			node.Iden = fmt.Sprintf("int%d", bitLevel)
			node.Primitive = true
			return
		}
	}
	node.Iden = "big.Int"
	node.Primitive = true
}

// Pass Two
// 1: field orders start from one and are monotonically increasing by 1
// 2: identifier are resolved to a valid ast with valid type arguments
// 3: graph does not contain any cycles
func checkPreconditions(nodes []Node, errs *[]error) {

}

type PropTable = map[string]string

func makePropTable(nodes []Node) PropTable {
	propTable := make(PropTable)
	for _, node := range nodes {
		node, ok := node.(*PropertyNode)
		if ok && !node.Poisoned {
			propTable[node.Name] = node.Value
		}
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
	if n.GetPoisoned() {
		// ignore any ast that is poisoned, it is only left in the ast for pre-codegen validation
		return
	}
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

func (b *CodeBuilder) buildTypeRef(ref TypeRefNode) {
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

func (b *CodeBuilder) buildService(node *ServiceNode) {

}

func (b *CodeBuilder) buildPackage(pack string) {
	b.w("package ")
	b.w(pack)
	b.w("\n\n")
}

func runCodeBuilder(program string, pack string, errs *[]error) string {
	nodes := runParser(program, errs)

	// prepare metadata tables for performing the code generation. we stop if we encounter any errors before building code
	prepareTypeTables(nodes, nil, errs)
	checkPreconditions(nodes, errs)

	if len(*errs) > 0 {
		return ""
	}

	propTable := makePropTable(nodes)

	// build the program into a string
	builder := makeCodeBuilder(propTable, errs)
	builder.buildPackage(pack)
	for _, node := range nodes {
		builder.build(node)
	}

	return builder.sb.String()
}
