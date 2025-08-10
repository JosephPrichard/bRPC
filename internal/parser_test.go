package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParser_Properties(t *testing.T) {
	input := `
	import "/path/to/idl/file.brpc"

	package = "/hello/\\\"world\""
	constant = "value"
	`
	asts, errs := runParser(input)

	expectedAsts := []Ast{
		&ImportAst{Path: "/path/to/idl/file.brpc"},
		&PropertyAst{Name: "package", Value: "/hello/\\\"world\""},
		&PropertyAst{Name: "constant", Value: "value"},
	}

	assert.Equal(t, expectedAsts, asts)
	assert.Nil(t, errs)
}

func TestParser_Struct(t *testing.T) {
	input := `
	message Data1 struct {
		required one @1 b128; // this is the first comment
		required two @2 []b5; // this is the second comment
		optional three @3 [16]b4;
		optional four @4 [][4][]b4;
		required five @5 struct{ 
			required one @1 b16;
        };

		message Data2 struct {
			required one @1 Data3;
	
			message Data3 struct {
				required one @1 b24;
			}
		}
	}
	`
	asts, errs := runParser(input)

	expectedAsts := []Ast{
		&StructAst{
			Name: "Data1",
			Fields: []FieldAst{
				{Modifier: Required, Name: "one", Ord: 1, Type: &TypeRefAst{Name: "b128"}},
				{
					Modifier: Required,
					Name:     "two",
					Ord:      2,
					Type:     &TypeArrayAst{Type: &TypeRefAst{Name: "b5"}},
				},
				{
					Modifier: Optional,
					Name:     "three",
					Ord:      3,
					Type:     &TypeArrayAst{Type: &TypeRefAst{Name: "b4"}, Size: 16},
				},
				{
					Modifier: Optional,
					Name:     "four",
					Ord:      4,
					Type: &TypeArrayAst{
						Type: &TypeArrayAst{
							Type: &TypeArrayAst{Type: &TypeRefAst{Name: "b4"}},
							Size: 4,
						},
					},
				},
				{
					Modifier: Required,
					Name:     "five",
					Ord:      5,
					Type: &StructAst{
						Fields: []FieldAst{{Modifier: Required, Name: "one", Ord: 1, Type: &TypeRefAst{Name: "b16"}}},
					},
				},
			},
			LocalDefs: []Ast{
				&StructAst{
					Name: "Data2",
					Fields: []FieldAst{
						{Modifier: Required, Name: "one", Ord: 1, Type: &TypeRefAst{Name: "Data3"}},
					},
					LocalDefs: []Ast{
						&StructAst{
							Name: "Data3",
							Fields: []FieldAst{
								{Modifier: Required, Name: "one", Ord: 1, Type: &TypeRefAst{Name: "b24"}},
							},
						},
					},
				},
			},
		},
	}

	assert.Equal(t, expectedAsts, asts)
	assert.Nil(t, errs)
}

func TestParser_Enum(t *testing.T) {
	input := `
	message Data1 enum {
		@1 One;
		@2 Two;
		@3 Three;
	}
	`
	asts, errs := runParser(input)

	expectedAsts := []Ast{
		&EnumAst{
			Name: "Data1",
			Cases: []EnumCase{
				{Ord: 1, Name: "One"},
				{Ord: 2, Name: "Two"},
				{Ord: 3, Name: "Three"},
			},
		},
	}

	assert.Equal(t, expectedAsts, asts)
	assert.Nil(t, errs)
}

func TestParser_Union(t *testing.T) {
	input := `
	message Data union {
		@1 struct{};
		@2 struct{ required one @1 b16; };
		@3 Data;

		message Data struct { 
			required one @1 b2; 
        }
	}
	`
	asts, errs := runParser(input)

	expectedAsts := []Ast{
		&UnionAst{
			Name: "Data",
			Options: []OptionAst{
				{Ord: 1, Type: &StructAst{}},
				{
					Ord: 2,
					Type: &StructAst{
						Fields: []FieldAst{
							{Modifier: Required, Name: "one", Ord: 1, Type: &TypeRefAst{Name: "b16"}},
						},
					},
				},
				{Ord: 3, Type: &TypeRefAst{Name: "Data"}},
			},
			LocalDefs: []Ast{
				&StructAst{
					Name: "Data",
					Fields: []FieldAst{
						{Modifier: Required, Name: "one", Ord: 1, Type: &TypeRefAst{Name: "b2"}},
					},
				},
			},
		},
	}

	assert.Equal(t, expectedAsts, asts)
	assert.Nil(t, errs)
}

func TestParser_Service(t *testing.T) {
	input := `
	service ServiceA {
		rpc @1 Hello(Input) returns (Output)
		rpc @2 World(struct{}) returns (enum{ @1 One; @2 Two; @3 Three; })

		message Input struct {
			required one @1 b24;
		}
	}
	`
	asts, errs := runParser(input)

	expectedAsts := []Ast{
		&ServiceAst{
			Name: "ServiceA",
			Procedures: []RpcAst{
				{Ord: 1, Name: "Hello", Arg: &TypeRefAst{Name: "Input"}, Ret: &TypeRefAst{Name: "Output"}},
				{
					Ord:  2,
					Name: "World",
					Arg:  &StructAst{},
					Ret: &EnumAst{
						Cases: []EnumCase{
							{Ord: 1, Name: "One"},
							{Ord: 2, Name: "Two"},
							{Ord: 3, Name: "Three"},
						},
					},
				},
			},
			LocalDefs: []Ast{
				&StructAst{
					Name: "Input",
					Fields: []FieldAst{
						{Modifier: Required, Name: "one", Ord: 1, Type: &TypeRefAst{Name: "b24"}},
					},
				},
			},
		},
	}

	assert.Equal(t, expectedAsts, asts)
	assert.Nil(t, errs)
}
