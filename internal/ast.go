package internal

type Modifier int

const (
	Undefined Modifier = iota
	Required
	Optional
)

type Ast interface {
	Error(err error)
	String() string
}

type StructAst struct {
	Name      string // an empty string is an anonymous struct
	Fields    []*FieldAst
	err       error
	LocalDefs []Ast
}

type EnumAst struct {
	Name      string
	Cases     []EnumCase
	err       error
	LocalDefs []Ast
}

type EnumCase struct {
	Name  string
	Value int64
}

type UnionAst struct {
	Name      string
	Options   []string
	err       error
	LocalDefs []Ast
}

type FieldAst struct {
	Modifier Modifier
	Name     string
	Type     Ast
	Ord      uint64
	err      error
}

type ServiceAst struct {
	Name       string
	Operations []*RpcAst
	err        error
	LocalDefs  []Ast
}

type RpcAst struct {
	Name string
	Arg  Ast
	Ret  Ast
	err  error
}

type TypeRefAst struct {
	Name string
}

type TypeArrayAst struct {
	Type Ast
	Size uint64 // 0 means the array is a dynamic array
}

func (ast *StructAst) Error(err error)    { ast.err = err }
func (ast *EnumAst) Error(err error)      { ast.err = err }
func (ast *UnionAst) Error(err error)     { ast.err = err }
func (ast *ServiceAst) Error(err error)   { ast.err = err }
func (ast *RpcAst) Error(err error)       { ast.err = err }
func (ast *FieldAst) Error(err error)     { ast.err = err }
func (ast *TypeRefAst) Error(_ error)     {}
func (ast *TypeArrayAst) Error(err error) {}

func (ast *StructAst) String() string    { return "<struct>" }
func (ast *EnumAst) String() string      { return "<enum>" }
func (ast *UnionAst) String() string     { return "<union>" }
func (ast *ServiceAst) String() string   { return "<service>" }
func (ast *RpcAst) String() string       { return "<operation>" }
func (ast *FieldAst) String() string     { return "<field>" }
func (ast *TypeRefAst) String() string   { return "<type>" }
func (ast *TypeArrayAst) String() string { return "<array>" }
