package internal

import "fmt"

type Modifier int

const (
	Required Modifier = iota
	Optional
)

type Ast interface {
	Error(err error)
	Category() string
}

type PropertyAst struct {
	Name  string
	Value string
	Errs  []error
}

type ImportAst struct {
	Path string
	Err  error
}

type StructAst struct {
	Name      string // an empty string is an anonymous struct
	Fields    []FieldAst
	TypeArgs  []string
	LocalDefs []Ast
	Errs      []error
}

type EnumAst struct {
	Name  string // an empty string is an anonymous enum
	Cases []EnumCase
	Errs  []error
}

type EnumCase struct {
	Name string
	Ord  uint64
}

type UnionAst struct {
	Name      string // an empty string is an anonymous union
	Options   []OptionAst
	LocalDefs []Ast
	Errs      []error
}

type FieldAst struct {
	Modifier Modifier
	Name     string
	Type     Ast
	Ord      uint64
	Errs     []error
}

type OptionAst struct {
	Type Ast
	Ord  uint64
	Errs []error
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

func (ast *PropertyAst) Error(err error) { ast.Errs = append(ast.Errs, err) }
func (ast *ImportAst) Error(err error)   { ast.Err = err }
func (ast *StructAst) Error(err error)   { ast.Errs = append(ast.Errs, err) }
func (ast *EnumAst) Error(err error)     { ast.Errs = append(ast.Errs, err) }
func (ast *UnionAst) Error(err error)    { ast.Errs = append(ast.Errs, err) }
func (ast *ServiceAst) Error(err error)  { ast.errs = append(ast.errs, err) }
func (ast *RpcAst) Error(err error)      { ast.errs = append(ast.errs, err) }
func (ast *FieldAst) Error(err error)    { ast.Errs = append(ast.Errs, err) }
func (ast *OptionAst) Error(err error)   { ast.Errs = append(ast.Errs, err) }
func (ast *TypeRefAst) Error(err error) {
	panic(fmt.Sprintf("assertion error: type ref ast received an unhandled error: %v", err))
}
func (ast *TypeArrayAst) Error(err error) {
	panic(fmt.Sprintf("assertion error: type array ast received an unhandled error: %v", err))
}

func (ast *PropertyAst) Category() string  { return "<property>" }
func (ast *ImportAst) Category() string    { return "<import>" }
func (ast *StructAst) Category() string    { return "<struct>" }
func (ast *EnumAst) Category() string      { return "<enum>" }
func (ast *UnionAst) Category() string     { return "<union>" }
func (ast *ServiceAst) Category() string   { return "<service>" }
func (ast *RpcAst) Category() string       { return "<operation>" }
func (ast *FieldAst) Category() string     { return "<field>" }
func (ast *TypeRefAst) Category() string   { return "<type>" }
func (ast *TypeArrayAst) Category() string { return "<array>" }
