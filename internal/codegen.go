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

// invariant: an abstract syntax tree is well-formed
func makeTables(asts []Ast, prev *SymbolTable, errs *[]error) {
	table := &SymbolTable{m: make(map[string]Ast), prev: prev}

	insertAst := func(iden string, ast Ast) {
		_, ok := table.m[iden]
		if ok {
			*errs = append(*errs, &AstErr{ast: ast, msg: fmt.Sprintf("'%s' is redefined", iden)})
		}
		table.m[iden] = ast
	}

	for _, ast := range asts {
		switch ast := ast.(type) {
		case *StructAst:
			ast.Table = table
			insertAst(ast.Name, ast)
			makeTables(ast.LocalDefs, table, errs)
		case *UnionAst:
			ast.Table = table
			insertAst(ast.Name, ast)
			makeTables(ast.LocalDefs, table, errs)
		case *EnumAst:
			ast.Table = table
		case *ServiceAst:
			ast.Table = table
			insertAst(ast.Name, ast)
			makeTables(ast.LocalDefs, table, errs)
		case *TypeRefAst:
			ast.Table = table
			insertAst(ast.Alias, ast)
		case *TypeArrAst:
			if ref, ok := ast.Type.(*TypeRefAst); ok {
				ref.Table = table
				insertAst(ref.Alias, ast)
			} else {
				panic(fmt.Sprintf("assertion error: array must have type as children, got: %T", ast.Type))
			}
		}
	}
}

// invariant: an abstract syntax tree is well-formed, but the data contained inside it may be invalid
func validateAsts(asts []Ast, errs *[]error) {
	for _, ast := range asts {
		switch ast := ast.(type) {
		case *StructAst:
			validateAsts(ast.LocalDefs, errs)
		case *UnionAst:
			validateAsts(ast.LocalDefs, errs)
		case *EnumAst:
			//ast := ast.(*EnumAst)
		case *ServiceAst:
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

	makeTables(asts, nil, &builder.errs)
	validateAsts(asts, &builder.errs)

	if builder.errs != nil {
		return "", builder.errs
	}

	return builder.sb.String(), builder.errs
}
