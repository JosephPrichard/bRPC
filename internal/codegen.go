package internal

import (
	"strconv"
	"strings"
	"unicode"
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

func (b *CodeBuilder) buildNodes(nodes []DefNode) {
	for _, node := range nodes {
		b.build(node)
	}
}

func (b *CodeBuilder) build(node DefNode) {
	switch node.Kind {
	case StructNodeKind:
		b.buildStruct(node)
	case UnionNodeKind:
		b.buildUnion(node)
	case EnumNodeKind:
		b.buildEnum(node)
	case ServiceNodeKind:
		b.buildService(node)
	}
}

// this operation is common enough to extract it out to a utility function
func (b *CodeBuilder) write(s string) {
	b.sb.WriteString(s)
}

func (b *CodeBuilder) writeIden(s string) {
	for i, c := range s {
		if i == 0 {
			c = unicode.ToUpper(c)
		}
		b.sb.WriteRune(c)
	}
}

func (b *CodeBuilder) buildType(t TypeNode) {
	for _, size := range t.Array {
		b.write("[")
		if size > 0 {
			b.write(strconv.FormatUint(size, 10))
		}
		b.write("]")
	}
	b.write(t.Value.Native())
}

func (b *CodeBuilder) buildStruct(strct DefNode) {
	if strct.Poisoned {
		return
	}

	// build out the struct type definition
	b.write("type ")
	b.write(strct.Iden)
	b.write(" struct {\n")
	for _, field := range strct.Members {
		b.write("\t")
		b.writeIden(field.Iden)
		b.write("\t")
		b.buildType(field.LType)
		b.write("\n")
	}
	b.write("}\n\n")

	// build out the struct's serialize and deserialize methods
}

func (b *CodeBuilder) buildUnion(union DefNode) {
	if union.Poisoned {
		return
	}

	// build out the union type definition
	b.write("type ")
	b.write(union.Iden)
	b.write("Kind int\n\n")
	b.write("const (\n")
	for i, c := range union.Members {
		b.write("\t")
		b.write(union.Iden)
		b.write("Kind")
		b.writeIden(c.Iden)
		if i == 0 {
			b.write(" ")
			b.write(union.Iden)
			b.write("Kind = iota")
		}
		b.write("\n")
	}
	b.write(")\n\n")

	b.write("type ")
	b.write(union.Iden)
	b.write(" struct {\n")
	b.write("\tKind\t")
	b.write(union.Iden)
	b.write("Kind\n")
	for _, option := range union.Members {
		b.write("\t")
		b.writeIden(option.Iden)
		b.write("\t*")
		b.buildType(option.LType)
		b.write("\n")
	}
	b.write("}\n\n")

	// build out the union's serialize and deserialize methods
}

func (b *CodeBuilder) buildEnum(enum DefNode) {
	if enum.Poisoned {
		return
	}

	// build out the enum type definition and cases
	b.write("type ")
	b.write(enum.Iden)
	b.write(" int\n\n")
	b.write("const (\n")
	for i, c := range enum.Members {
		b.write("\t")
		b.write(enum.Iden)
		b.write(c.Iden)
		if i == 0 {
			b.write(" ")
			b.write(enum.Iden)
			b.write(" = iota")
		}
		b.write("\n")
	}
	b.write(")\n\n")

	// build out the enum's serialize and deserialize methods
}

func (b *CodeBuilder) buildService(svc DefNode) {
	if svc.Poisoned {
		return
	}

}

func (b *CodeBuilder) buildPackage(pack string) {
	b.write("package ")
	b.write(pack)
	b.write("\n\n")
}

func runCodeBuilder(program string, pack string, errs *[]error) string {
	nodes := runParser(program, errs)

	importTable := makeImportTable()
	propTable := makePropTable(nodes)

	tb := makeTransformer(errs)
	tb.transformNodeList(nodes, nil)
	tb.validateNodeList(nodes)

	if len(*errs) > 0 {
		return ""
	}

	cb := makeCodeBuilder(propTable, importTable, errs)
	cb.buildPackage(pack)
	cb.buildNodes(nodes)

	return cb.sb.String()
}
