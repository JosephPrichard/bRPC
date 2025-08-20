package internal

import (
	"fmt"
	"strings"
)

// SymbolTable is used to look up the ast that an identifier may resolve to at any level in the ast
type SymbolTable struct {
	prev *SymbolTable
	m    map[string]Ast
}

func (t *SymbolTable) getAst(iden string) Ast {
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
func makeSymbolTables(asts []Ast, prev *SymbolTable, errs *[]error) {
	table := &SymbolTable{m: make(map[string]Ast), prev: prev}

	insertAst := func(iden string, ast Ast) {
		_, exists := table.m[iden]
		if exists {
			*errs = append(*errs, &AstErr{ast: ast, msg: fmt.Sprintf("'%s' is redefined", iden)})
		}
		table.m[iden] = ast
	}

	for _, ast := range asts {
		switch ast := ast.(type) {
		case *StructAst:
			ast.Table = table
			insertAst(ast.Name, ast)
			makeSymbolTables(ast.LocalDefs, table, errs)
		case *UnionAst:
			ast.Table = table
			insertAst(ast.Name, ast)
			makeSymbolTables(ast.LocalDefs, table, errs)
		case *EnumAst:
			ast.Table = table
		case *ServiceAst:
			ast.Table = table
			insertAst(ast.Name, ast)
			makeSymbolTables(ast.LocalDefs, table, errs)
		case *TypeRefAst:
			ast.Table = table
			insertAst(ast.Alias, ast)
		case *TypeArrAst:
			if ref, ok := ast.Type.(*TypeRefAst); ok {
				ref.Table = table
				insertAst(ref.Alias, ast)
			}
		}
	}
}

type PropTable = map[string]string

func makePropTable(asts []Ast) PropTable {
	propTable := make(PropTable)

	// build property table using root property asts. we can assume there are no property nodes below the root, since that is illegal
	for _, ast := range asts {
		ast, ok := ast.(*PropertyAst)
		if ok && !ast.Poisoned {
			propTable[ast.Name] = ast.Value
		}
	}

	return propTable
}

type ImportTable = map[string]Ast

func makeImportTable() ImportTable {
	importTable := make(ImportTable)
	return importTable
}

type CodeBuilder struct {
	sb          strings.Builder
	propTable   PropTable
	importTable ImportTable
	errs        []error
}

func makeCodeBuilder(propTable PropTable, importTable ImportTable) CodeBuilder {
	return CodeBuilder{propTable: propTable, importTable: importTable}
}

func (b *CodeBuilder) build(ast Ast) {
	switch ast := ast.(type) {
	case *StructAst:

	case *UnionAst:

	case *EnumAst:

	case *ServiceAst:
	case *ImportAst, *PropertyAst:
		// all property and import asts should already be resolved in the property and import tables
	default:
		panic(fmt.Sprintf("unsupported: ast type is not implemented: %T", ast))
	}
}

func runCodegen(asts []Ast) (string, []error) {
	var errs []error

	makeSymbolTables(asts, nil, &errs)
	propTable := makePropTable(asts)
	importTable := makeImportTable()

	if errs != nil {
		return "", errs
	}

	builder := makeCodeBuilder(propTable, importTable)
	for _, ast := range asts {
		builder.build(ast)
	}

	return builder.sb.String(), builder.errs
}
