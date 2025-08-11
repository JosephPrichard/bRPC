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

func TestParser_Alias(t *testing.T) {
	input := `
	message Data Data1
	message BinaryArray [6]b64
	message Object [4]Object1(BinaryArray []Object3(b8) [6]b8 []b16)
	`
	asts, errs := runParser(input)

	expectedAsts := []Ast{
		&TypeAst{Alias: "Data", Value: "Data1"},
		&ArrayAst{Type: &TypeAst{Alias: "BinaryArray", Value: "b64"}, Size: 6},
		&ArrayAst{
			Type: &TypeAst{
				Alias: "Object",
				Value: "Object1",
				TypeArgs: []Ast{
					&TypeAst{Value: "BinaryArray"},
					&ArrayAst{
						Type: &TypeAst{Value: "Object3",
							TypeArgs: []Ast{&TypeAst{Value: "b8"}},
						},
					},
					&ArrayAst{Type: &TypeAst{Value: "b8"}, Size: 6},
					&ArrayAst{Type: &TypeAst{Value: "b16"}},
				},
			},
			Size: 4,
		},
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
	
			message Data3 struct(A B) {
				required one @1 A;
				required two @2 B;
			}
		}
	}
	`
	asts, errs := runParser(input)

	expectedAsts := []Ast{
		&StructAst{
			Name: "Data1",
			Fields: []FieldAst{
				{Modifier: Required, Name: "one", Ord: 1, Type: &TypeAst{Value: "b128"}},
				{
					Modifier: Required,
					Name:     "two",
					Ord:      2,
					Type:     &ArrayAst{Type: &TypeAst{Value: "b5"}},
				},
				{
					Modifier: Optional,
					Name:     "three",
					Ord:      3,
					Type:     &ArrayAst{Type: &TypeAst{Value: "b4"}, Size: 16},
				},
				{
					Modifier: Optional,
					Name:     "four",
					Ord:      4,
					Type: &ArrayAst{
						Type: &ArrayAst{
							Type: &ArrayAst{Type: &TypeAst{Value: "b4"}},
							Size: 4,
						},
					},
				},
				{
					Modifier: Required,
					Name:     "five",
					Ord:      5,
					Type: &StructAst{
						Fields: []FieldAst{{Modifier: Required, Name: "one", Ord: 1, Type: &TypeAst{Value: "b16"}}},
					},
				},
			},
			LocalDefs: []Ast{
				&StructAst{
					Name: "Data2",
					Fields: []FieldAst{
						{Modifier: Required, Name: "one", Ord: 1, Type: &TypeAst{Value: "Data3"}},
					},
					LocalDefs: []Ast{
						&StructAst{
							Name:     "Data3",
							TypeArgs: []string{"A", "B"},
							Fields: []FieldAst{
								{Modifier: Required, Name: "one", Ord: 1, Type: &TypeAst{Value: "A"}},
								{Modifier: Required, Name: "two", Ord: 2, Type: &TypeAst{Value: "B"}},
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
	message Data union(A B C) {
		@1 struct{};
		@2 struct{ required one @1 A; };
		@3 Data;

		message Data union() { 
			@1 B;
			@2 C;
        }
	}
	`
	asts, errs := runParser(input)

	expectedAsts := []Ast{
		&UnionAst{
			Name:     "Data",
			TypeArgs: []string{"A", "B", "C"},
			Options: []OptionAst{
				{Ord: 1, Type: &StructAst{}},
				{
					Ord: 2,
					Type: &StructAst{
						Fields: []FieldAst{
							{Modifier: Required, Name: "one", Ord: 1, Type: &TypeAst{Value: "A"}},
						},
					},
				},
				{Ord: 3, Type: &TypeAst{Value: "Data"}},
			},
			LocalDefs: []Ast{
				&UnionAst{
					Name: "Data",
					Options: []OptionAst{
						{Ord: 1, Type: &TypeAst{Value: "B"}},
						{Ord: 2, Type: &TypeAst{Value: "C"}},
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
				{Ord: 1, Name: "Hello", Arg: &TypeAst{Value: "Input"}, Ret: &TypeAst{Value: "Output"}},
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
						{Modifier: Required, Name: "one", Ord: 1, Type: &TypeAst{Value: "b24"}},
					},
				},
			},
		},
	}

	assert.Equal(t, expectedAsts, asts)
	assert.Nil(t, errs)
}
