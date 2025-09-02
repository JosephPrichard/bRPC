package internal

import (
	"fmt"
	"strconv"
	"strings"
)

type CodeBuilder struct {
	sb          strings.Builder
	propTable   PropTable
	importTable ImportTable
	errs        *[]error
}

func makeCodeBuilder(propTable PropTable, importTable ImportTable, errs *[]error) CodeBuilder {
	return CodeBuilder{propTable: propTable, importTable: importTable, errs: errs}
}

func (b *CodeBuilder) buildNodes(nodes []Node) {
	for _, node := range nodes {
		b.build(node)
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
	b.w(ref.TIden)
}

func (b *CodeBuilder) buildStruct(strct *StructNode) {
	if strct.Poisoned {
		return
	}

	// build out the struct type definition
	b.w("type ")
	b.w(strct.Iden)
	b.w(" struct {\n")
	for _, field := range strct.Fields {
		b.w("\t")
		b.w(field.Iden)
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
		b.w(union.Iden)
		b.w(option.TIden)
		b.w("Opt")
	}

	// build out the union type definition
	b.w("type ")
	b.w(union.Iden)
	b.w(" interface {\n")
	for _, option := range union.Options {
		b.w("\t")
		b.w(option.TIden)
		b.w("() *")
		writeOptionStruct(option)
		b.w("\n")
	}
	b.w("}\n\n")

	defaultName := union.Iden + "Unimplemented"

	// each union option contains a default implementation for each of the interface methods
	b.w("type ")
	b.w(defaultName)
	b.w(" struct {}\n\n")
	for _, option := range union.Options {
		b.w("func (_ *")
		b.w(defaultName)
		b.w(") ")
		b.w(option.TIden)
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
		b.w(option.TIden)
		b.w("\n\t")
		b.w(defaultName)
		b.w("\n}\n\n")

		b.w("func (o *")
		writeOptionStruct(option)
		b.w(") ")
		b.w(option.TIden)
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
	b.w(enum.Iden)
	b.w(" int\n\n")
	b.w("const (\n")
	for i, c := range enum.Cases {
		b.w("\t")
		b.w(enum.Iden)
		b.w(c.Iden)
		if i == 0 {
			b.w(" ")
			b.w(enum.Iden)
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

	tb := makeTransformer(errs)
	tb.makePropTable(nodes)
	tb.transformTypes(nodes, nil)
	tb.validateTypes(nodes)

	if len(*errs) > 0 {
		return ""
	}

	cb := makeCodeBuilder(tb.propTable, tb.importTable, errs)
	cb.buildPackage(pack)
	cb.buildNodes(nodes)

	return cb.sb.String()
}
