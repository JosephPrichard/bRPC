package internal

import (
	"fmt"
	"strings"
)

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

func makeSymbolTables(asts []Ast, prev *SymbolTable, errs *[]error) {
	table := &SymbolTable{m: make(map[string]Ast), prev: prev}

	insertAst := func(name string, ast Ast) {
		_, ok := table.m[name]
		if ok {
			*errs = append(*errs, fmt.Errorf("duplicate symbol table name: %s", name))
		}
		table.m[name] = ast
	}

	for _, a := range asts {
		switch a.(type) {
		case *StructAst:
			ast := a.(*StructAst)
			ast.Table = table
			insertAst(ast.Name, ast)
			makeSymbolTables(ast.LocalDefs, table, errs)
		case *UnionAst:
			ast := a.(*UnionAst)
			ast.Table = table
			insertAst(ast.Name, ast)
			makeSymbolTables(ast.LocalDefs, table, errs)
		case *EnumAst:
			ast := a.(*EnumAst)
			ast.Table = table
		case *ServiceAst:
			ast := a.(*ServiceAst)
			ast.Table = table
			insertAst(ast.Name, ast)
			makeSymbolTables(ast.LocalDefs, table, errs)
		case *TypeAst:
			ast := a.(*TypeAst)
			ast.Table = table
			insertAst(ast.Alias, ast)
		case *ArrayAst:
			ast := a.(*ArrayAst)
			typ, ok := ast.Type.(*TypeAst)
			if ok {
				typ.Table = table
				insertAst(typ.Alias, ast)
			}
		}
	}
}

func validateAsts(asts []Ast, errs *[]error) {
	for _, ast := range asts {
		switch ast.(type) {
		case *StructAst:
			ast := ast.(*StructAst)
			validateAsts(ast.LocalDefs, errs)
		case *UnionAst:
			ast := ast.(*UnionAst)
			validateAsts(ast.LocalDefs, errs)
		case *EnumAst:
			//ast := ast.(*EnumAst)
		case *ServiceAst:
			ast := ast.(*ServiceAst)
			validateAsts(ast.LocalDefs, errs)
		}
	}
}

type CodeBuilder struct {
	sb   strings.Builder
	errs []error
}

func makeProgram(asts []Ast) (string, []error) {
	var builder CodeBuilder

	makeSymbolTables(asts, nil, &builder.errs)
	validateAsts(asts, &builder.errs)

	if builder.errs != nil {
		return "", builder.errs
	}

	return builder.sb.String(), builder.errs
}
