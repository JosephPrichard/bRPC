package internal

type Modifier int

const (
	Required Modifier = iota
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
	Ord      int
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

type TypeAst struct {
	Name string
}

func (ast *StructAst) Error(err error)  { ast.err = err }
func (ast *EnumAst) Error(err error)    { ast.err = err }
func (ast *UnionAst) Error(err error)   { ast.err = err }
func (ast *ServiceAst) Error(err error) { ast.err = err }
func (ast *RpcAst) Error(err error)     { ast.err = err }
func (ast *FieldAst) Error(err error)   { ast.err = err }
func (ast *TypeAst) Error(_ error)   {}

func (ast *StructAst) String() string  { return "<struct>" }
func (ast *EnumAst) String() string    { return "<enum>" }
func (ast *UnionAst) String() string   { return "<union>" }
func (ast *ServiceAst) String() string { return "<service>" }
func (ast *RpcAst) String() string     { return "<operation>" }
func (ast *FieldAst) String() string   { return "<field>" }
func (ast *TypeAst) String() string   { return "<type>" }
