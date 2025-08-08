package internal

import "fmt"

type Modifier int

const (
	Required Modifier = iota
	Optional
)

type Ast interface {
	Error(err error)
	String() string
}

type PropertyAst struct {
	Name  string
	Value string
	errs  []error
}

type StructAst struct {
	Name      string // an empty string is an anonymous struct
	Fields    []FieldAst
	errs      []error
	LocalDefs []Ast
}

type EnumAst struct {
	Name      string
	Cases     []EnumCase
	errs      []error
	LocalDefs []Ast
}

type EnumCase struct {
	Name string
	Ord  uint64
}

type UnionAst struct {
	Name      string
	Options   []OptionAst
	errs      []error
	LocalDefs []Ast
}

type FieldAst struct {
	Modifier Modifier
	Name     string
	Type     Ast
	Ord      uint64
	errs     []error
}

type OptionAst struct {
	Type Ast
	Ord  uint64
	errs []error
}

type ServiceAst struct {
	Name       string
	Procedures []RpcAst
	errs       []error
	LocalDefs  []Ast
}

type RpcAst struct {
	Name string
	Ord  uint64
	Arg  Ast
	Ret  Ast
	errs []error
}

type TypeRefAst struct {
	Name string
}

type TypeArrayAst struct {
	Type Ast
	Size uint64 // 0 means the array is a dynamic array
}

func (ast *PropertyAst) Error(err error) { ast.errs = append(ast.errs, err) }
func (ast *StructAst) Error(err error)   { ast.errs = append(ast.errs, err) }
func (ast *EnumAst) Error(err error)     { ast.errs = append(ast.errs, err) }
func (ast *UnionAst) Error(err error)    { ast.errs = append(ast.errs, err) }
func (ast *ServiceAst) Error(err error)  { ast.errs = append(ast.errs, err) }
func (ast *RpcAst) Error(err error)      { ast.errs = append(ast.errs, err) }
func (ast *FieldAst) Error(err error)    { ast.errs = append(ast.errs, err) }
func (ast *OptionAst) Error(err error)   { ast.errs = append(ast.errs, err) }
func (ast *TypeRefAst) Error(err error) {
	panic(fmt.Sprintf("assertion error: type ref ast received an unhandled error: %v", err))
}
func (ast *TypeArrayAst) Error(err error) {
	panic(fmt.Sprintf("assertion error: type array ast received an unhandled error: %v", err))
}

func (ast *PropertyAst) String() string  { return "<property>" }
func (ast *StructAst) String() string    { return "<struct>" }
func (ast *EnumAst) String() string      { return "<enum>" }
func (ast *UnionAst) String() string     { return "<union>" }
func (ast *ServiceAst) String() string   { return "<service>" }
func (ast *RpcAst) String() string       { return "<operation>" }
func (ast *FieldAst) String() string     { return "<field>" }
func (ast *TypeRefAst) String() string   { return "<type>" }
func (ast *TypeArrayAst) String() string { return "<array>" }
