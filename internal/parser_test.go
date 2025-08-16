package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func clearAsts(asts []Ast) {
	for _, ast := range asts {
		ast.Clear()
	}
}

func TestParser_Properties(t *testing.T) {
	input := `
	import "/path/to/idl/file.brpc"

	package = "/hello/\\\"world\""
	constant = "value"
	`
	asts, errs := runParser(input)
	clearAsts(asts)

	expectedAsts := []Ast{
		&ImportAst{
			Path: "/path/to/idl/file.brpc",
			Markers: makeMarkers(TokImport, "import", TokString, "\"/path/to/idl/file.brpc\""),
		},
		&PropertyAst{
			Name:  "package",
			Value: "/hello/\\\"world\"",
			Markers: makeMarkers(TokIden, "package", TokString, "\"/hello/\\\\\\\"world\\\"\""),
		},
		&PropertyAst{
			Name:  "constant",
			Value: "value",
			Markers: makeMarkers(TokIden, "constant", TokString, "\"value\""),
		},
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
	clearAsts(asts)

	expectedAsts := []Ast{
		&TypRefAst{
			Alias: "Data",
			Iden:  "Data1",
			Markers: makeMarkers(TokImport, "import", TokString, "\"/path/to/idl/file.brpc\""),
		},
		&TypArrAst{
			Type: &TypRefAst{
				Alias: "BinaryArray",
				Iden:  "b64",
				Markers: makeMarkers(TokImport, "import", TokString, "\"/path/to/idl/file.brpc\""),
			},
			Size: []uint64{6},
			Markers: makeMarkers(TokImport, "import", TokString, "\"/path/to/idl/file.brpc\""),
		},
		&TypArrAst{
			Type: &TypRefAst{
				Alias: "Object",
				Iden:  "Object1",
				TypeArgs: []Ast{
					&TypRefAst{
						Iden: "BinaryArray",
						Markers: makeMarkers(TokImport, "import", TokString, "\"/path/to/idl/file.brpc\""),
					},
					&TypArrAst{
						Type: &TypRefAst{
							Iden: "Object3",
							TypeArgs: []Ast{
								&TypRefAst{Iden: "b8"},
							},
							Markers: makeMarkers(TokImport, "import", TokString, "\"/path/to/idl/file.brpc\""),
						},
						Size: []uint64{0},
						Markers: makeMarkers(TokImport, "import", TokString, "\"/path/to/idl/file.brpc\""),
					},
					&TypArrAst{
						Type: &TypRefAst{Iden: "b8"},
						Size: []uint64{6},
						Markers: makeMarkers(TokImport, "import", TokString, "\"/path/to/idl/file.brpc\""),
					},
					&TypArrAst{
						Type: &TypRefAst{Iden: "b16"},
						Size: []uint64{0},
						Markers: makeMarkers(TokImport, "import", TokString, "\"/path/to/idl/file.brpc\""),
					},
				},
			},
			Size: []uint64{4},
			Markers: makeMarkers(TokImport, "import", TokString, "\"/path/to/idl/file.brpc\""),
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
		optional four @4 [][4][]b4;;;
		required five @5 struct{ 
			required one @1 b16;;
        };

		message Data2 struct {
			required one @1 Data3;
	
			message Data3 struct(A B) {
				required one @1 A;;;
				required two @2 B;
			}
		}
	}
	`
	asts, errs := runParser(input)
	clearAsts(asts)

	expectedAsts := []Ast{
		&StructAst{
			Name: "Data1",
			Fields: []FieldAst{
				{Modifier: Required, Name: "one", Ord: 1, Type: &TypRefAst{Iden: "b128"}},
				{
					Modifier: Required,
					Name:     "two",
					Ord:      2,
					Type:     &TypArrAst{Type: &TypRefAst{Iden: "b5"}, Size: []uint64{0}},
				},
				{
					Modifier: Optional,
					Name:     "three",
					Ord:      3,
					Type:     &TypArrAst{Type: &TypRefAst{Iden: "b4"}, Size: []uint64{16}},
				},
				{
					Modifier: Optional,
					Name:     "four",
					Ord:      4,
					Type:     &TypArrAst{Type: &TypRefAst{Iden: "b4"}, Size: []uint64{0, 4, 0}},
				},
				{
					Modifier: Required,
					Name:     "five",
					Ord:      5,
					Type: &StructAst{
						Fields: []FieldAst{{Modifier: Required, Name: "one", Ord: 1, Type: &TypRefAst{Iden: "b16"}}},
					},
				},
			},
			LocalDefs: []Ast{
				&StructAst{
					Name: "Data2",
					Fields: []FieldAst{
						{Modifier: Required, Name: "one", Ord: 1, Type: &TypRefAst{Iden: "Data3"}},
					},
					LocalDefs: []Ast{
						&StructAst{
							Name:       "Data3",
							TypeParams: []string{"A", "B"},
							Fields: []FieldAst{
								{Modifier: Required, Name: "one", Ord: 1, Type: &TypRefAst{Iden: "A"}},
								{Modifier: Required, Name: "two", Ord: 2, Type: &TypRefAst{Iden: "B"}},
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
		@1 One;;
		@2 Two;
		@3 Three;;;
	}
	`
	asts, errs := runParser(input)
	clearAsts(asts)

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
			@1 B;;;
			@2 C;
        }
	}
	`
	asts, errs := runParser(input)
	clearAsts(asts)

	expectedAsts := []Ast{
		&UnionAst{
			Name:       "Data",
			TypeParams: []string{"A", "B", "C"},
			Options: []OptionAst{
				{Ord: 1, Type: &StructAst{}},
				{
					Ord: 2,
					Type: &StructAst{
						Fields: []FieldAst{
							{Modifier: Required, Name: "one", Ord: 1, Type: &TypRefAst{Iden: "A"}},
						},
					},
				},
				{Ord: 3, Type: &TypRefAst{Iden: "Data"}},
			},
			LocalDefs: []Ast{
				&UnionAst{
					Name: "Data",
					Options: []OptionAst{
						{Ord: 1, Type: &TypRefAst{Iden: "B"}},
						{Ord: 2, Type: &TypRefAst{Iden: "C"}},
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
		rpc @1 Hello(Test) returns (Output)
		rpc @2 World(struct{}) returns (enum{ @1 One; @2 Two; @3 Three; })

		message Test struct {
			required one @1 b24;
		}
	}
	`
	asts, errs := runParser(input)
	clearAsts(asts)

	expectedAsts := []Ast{
		&ServiceAst{
			Name: "ServiceA",
			Procedures: []RpcAst{
				{Ord: 1, Name: "Hello", Arg: &TypRefAst{Iden: "Test"}, Ret: &TypRefAst{Iden: "Output"}},
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
					Name: "Test",
					Fields: []FieldAst{
						{Modifier: Required, Name: "one", Ord: 1, Type: &TypRefAst{Iden: "b24"}},
					},
				},
			},
		},
	}

	assert.Equal(t, expectedAsts, asts)
	assert.Nil(t, errs)
}

func TestParser_Errors(t *testing.T) {
	type Test struct {
		input string
		asts  []Ast
		errs  []error
	}

	tests := []Test{
		{
			input: `
			message Data1 struct {
				required one @1 b128;
			`,
		},
		{
			input: `
			message Data1 struct {
				message Data2 struct {
			`,
		},
		{
			input: `
			message Data1 struct {
				required one @1abc b128;
				two @2abc []b5;
		
				message Data2 struct {
					required one @1;
				}
			}
		
			message Data3 struct {
				required one @1;

				message Data4 union {
					@1 One;
					Two;
				}
			}

			message Data4 enum {
				@1 ONE;
				TWO;
			}
			`,
		},
		{
			input: `
			message Data struct {
				required one one one one one @1 b128;
				two two two two two @2 []b5;
				required three @3 b128
			}
			`,
		},
		{
			input: `
			service Data {
				rpc @1 Hello(Test) returns (Output)
				required one @1 b128;
				rpc @2 World(Test1) returns (Output1)
			}
			`,
		},
	}

	for _, test := range tests {
		asts, errs := runParser(test.input)

		printLine := func(err string) {
			t.Log(err)
		}
		printErrors(errs, "test.brpc", printLine)

		assert.Equal(t, test.asts, asts)
		assert.Equal(t, test.errs, errs)
	}
}
