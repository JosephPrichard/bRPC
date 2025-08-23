package internal

import (
	"fmt"
	"strings"
)

// SymbolTable is used to look up the ast that an identifier may resolve to at any level in the ast
type SymbolTable struct {
	prev *SymbolTable
	m    map[string]Node
}

func (t *SymbolTable) getAst(iden string) Node {
	table := t
	for table != nil {
		ast, ok := table.m[iden]
		if ok {
			return ast
		}
		table = table.prev
	}
	return nil
}

// symbol tables are attached to all asts to resolve all identifiers into other asts at any other level
// this may cause cycles if a child ast refers to a parent ast - this must be detected in a future codegen pass
func makeSymbolTables(asts []Node, prev *SymbolTable, errs *[]error) {
	table := &SymbolTable{m: make(map[string]Node), prev: prev}

	insertAst := func(iden string, ast Node) {
		_, exists := table.m[iden]
		if exists {
			*errs = append(*errs, &CodegenErr{ast: ast, msg: fmt.Sprintf("\"%s\" is redefined", iden)})
		}
		table.m[iden] = ast
	}

	for _, ast := range asts {
		switch ast := ast.(type) {
		case *StructNode:
			ast.Table = table
			insertAst(ast.Name, ast)
			makeSymbolTables(ast.LocalDefs, table, errs)
		case *UnionNode:
			ast.Table = table
			insertAst(ast.Name, ast)
			makeSymbolTables(ast.LocalDefs, table, errs)
		case *EnumNode:
			ast.Table = table
		case *ServiceNode:
			ast.Table = table
			insertAst(ast.Name, ast)
			makeSymbolTables(ast.LocalDefs, table, errs)
		}
	}
}

func checkPreconditions(asts []Node, errs *[]error) {
	// ast field orders start from one and are monotonically increasing by 1
	// all identifiers are resolved to a valid ast with valid type arguments
	// ast does not contain any cycles through the symbol table
}

type PropTable = map[string]string

func makePropTable(asts []Node) PropTable {
	propTable := make(PropTable)

	// build property table using root property asts. we can assume there are no property nodes below the root, since that is illegal
	for _, ast := range asts {
		ast, ok := ast.(*PropertyNode)
		if ok && !ast.Poisoned {
			propTable[ast.Name] = ast.Value
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

func (b *CodeBuilder) build(ast Node) {
	if ast.IsPoisoned() {
		// ignore any ast that is poisoned, it is only left in the ast for pre-codegen validation
		return
	}
	switch ast := ast.(type) {
	case *StructNode:
		b.buildStruct(ast)
	case *UnionNode:
		b.buildUnion(ast)
	case *EnumNode:
		b.buildEnum(ast)
	case *ServiceNode:
		b.buildService(ast)
	case *ImportNode, *PropertyNode:
		// noop: all property and import asts should be resolved already
	default:
		// this will fail for field and rpc asts, we assume they are handled in the builder for structs and services
		panic(fmt.Sprintf("unsupported: build call for ast type is not implemented: %T", ast))
	}
}

func (b *CodeBuilder) buildStruct(ast *StructNode) {
	// build out the struct type definition

	// build out the struct's serialize and deserialize methods
}

func (b *CodeBuilder) buildUnion(ast *UnionNode) {
	// build out the union type definition

	// build out the union's serialize and deserialize methods
}

func (b *CodeBuilder) buildEnum(ast *EnumNode) {
	// build out the enum type definition

	// build out the enum's serialize and deserialize methods
}

func (b *CodeBuilder) buildService(ast *ServiceNode) {

}

func (b *CodeBuilder) buildProperties() bool {
	pack, ok := b.propTable["package"]
	if !ok {
		*b.errs = append(*b.errs, &CodegenErr{msg: "\"package\" property is not defined"})
		return false
	}
	b.sb.WriteString("package")
	b.sb.WriteString(pack)
	return true
}

func runCodeBuilder(program string, errs *[]error) string {
	asts := runParser(program, errs)

	// prepare metadata tables for performing the code generation. we stop if we encounter any errors before building code
	makeSymbolTables(asts, nil, errs)
	propTable := makePropTable(asts)

	checkPreconditions(asts, errs)
	if len(*errs) > 0 {
		return ""
	}

	// build the program into a string
	builder := makeCodeBuilder(propTable, errs)
	if !builder.buildProperties() {
		return ""
	}
	for _, ast := range asts {
		builder.build(ast)
	}

	return builder.sb.String()
}
