package internal

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestParser_Properties(t *testing.T) {
	input := `
	import "/services/schemas/animals"

	package = "/hello/\\\"world\""
	constant = "Value"
	`
	asts, errs := runParser(input)
	WalkList(Ast.ClearPos, asts)

	expectedAsts := []Ast{
		&ImportAst{Path: "/services/schemas/animals"},
		&PropertyAst{Name: "package", Value: "/hello/\\\"world\""},
		&PropertyAst{Name: "constant", Value: "Value"},
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
	WalkList(Ast.ClearPos, asts)

	expectedAsts := []Ast{
		&TypeRefAst{Alias: "Data", Iden: "Data1"},
		&TypeArrAst{Type: &TypeRefAst{Alias: "BinaryArray", Iden: "b64"}, Size: []uint64{6}},
		&TypeArrAst{
			Type: &TypeRefAst{
				Alias: "Object",
				Iden:  "Object1",
				TypeArgs: []Ast{
					&TypeRefAst{Iden: "BinaryArray"},
					&TypeArrAst{
						Type: &TypeRefAst{Iden: "Object3", TypeArgs: []Ast{&TypeRefAst{Iden: "b8"}}},
						Size: []uint64{0},
					},
					&TypeArrAst{Type: &TypeRefAst{Iden: "b8"}, Size: []uint64{6}},
					&TypeArrAst{Type: &TypeRefAst{Iden: "b16"}, Size: []uint64{0}},
				},
			},
			Size: []uint64{4},
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
	WalkList(Ast.ClearPos, asts)

	expectedAsts := []Ast{
		&StructAst{
			Name: "Data1",
			Fields: []FieldAst{
				{Modifier: Required, Name: "one", Ord: 1, Type: &TypeRefAst{Iden: "b128"}},
				{
					Modifier: Required,
					Name:     "two",
					Ord:      2,
					Type:     &TypeArrAst{Type: &TypeRefAst{Iden: "b5"}, Size: []uint64{0}},
				},
				{
					Modifier: Optional,
					Name:     "three",
					Ord:      3,
					Type:     &TypeArrAst{Type: &TypeRefAst{Iden: "b4"}, Size: []uint64{16}},
				},
				{
					Modifier: Optional,
					Name:     "four",
					Ord:      4,
					Type:     &TypeArrAst{Type: &TypeRefAst{Iden: "b4"}, Size: []uint64{0, 4, 0}},
				},
				{
					Modifier: Required,
					Name:     "five",
					Ord:      5,
					Type: &StructAst{
						Fields: []FieldAst{{Modifier: Required, Name: "one", Ord: 1, Type: &TypeRefAst{Iden: "b16"}}},
					},
				},
			},
			LocalDefs: []Ast{
				&StructAst{
					Name: "Data2",
					Fields: []FieldAst{
						{Modifier: Required, Name: "one", Ord: 1, Type: &TypeRefAst{Iden: "Data3"}},
					},
					LocalDefs: []Ast{
						&StructAst{
							Name:       "Data3",
							TypeParams: []string{"A", "B"},
							Fields: []FieldAst{
								{Modifier: Required, Name: "one", Ord: 1, Type: &TypeRefAst{Iden: "A"}},
								{Modifier: Required, Name: "two", Ord: 2, Type: &TypeRefAst{Iden: "B"}},
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
	WalkList(Ast.ClearPos, asts)

	expectedAsts := []Ast{
		&EnumAst{
			Name: "Data1",
			Cases: []CaseAst{
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
	WalkList(Ast.ClearPos, asts)

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
							{Modifier: Required, Name: "one", Ord: 1, Type: &TypeRefAst{Iden: "A"}},
						},
					},
				},
				{Ord: 3, Type: &TypeRefAst{Iden: "Data"}},
			},
			LocalDefs: []Ast{
				&UnionAst{
					Name: "Data",
					Options: []OptionAst{
						{Ord: 1, Type: &TypeRefAst{Iden: "B"}},
						{Ord: 2, Type: &TypeRefAst{Iden: "C"}},
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
	WalkList(Ast.ClearPos, asts)

	expectedAsts := []Ast{
		&ServiceAst{
			Name: "ServiceA",
			Procedures: []RpcAst{
				{Ord: 1, Name: "Hello", Arg: &TypeRefAst{Iden: "Test"}, Ret: &TypeRefAst{Iden: "Output"}},
				{
					Ord:  2,
					Name: "World",
					Arg:  &StructAst{},
					Ret: &EnumAst{
						Cases: []CaseAst{
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
						{Modifier: Required, Name: "one", Ord: 1, Type: &TypeRefAst{Iden: "b24"}},
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
		name  string
		input string
		asts  []Ast
		errs  []error
	}

	tests := []Test{
		{
			name: "UnclosedStruct",
			input: `
			message Data1 struct {
				required one @1 b128;
			`,
			asts: []Ast{
				&StructAst{
					Tags: Tags{Poisoned: true},
					Name: "Data1",
					Fields: []FieldAst{
						{Modifier: Required, Name: "one", Ord: 1, Type: &TypeRefAst{Iden: "b128"}},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokEof, Value: ""}, Range{B: 56, E: 56}},
					kind:     StructAstKind,
					expected: []TokKind{TokField, TokMessage, TokRBrace},
				},
			},
		},
		{
			name: "UnclosedNestedStruct",
			input: `
			message Data1 struct {
				message Data2 union {
					message Data3 struct {
			`,
			asts: []Ast{
				&StructAst{
					Tags: Tags{Poisoned: true},
					Name: "Data1",
					LocalDefs: []Ast{
						&UnionAst{
							Tags: Tags{Poisoned: true},
							Name: "Data2",
							LocalDefs: []Ast{
								&StructAst{Tags: Tags{Poisoned: true}, Name: "Data3"},
							},
						},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokEof, Value: ""}, Range{B: 84, E: 84}},
					kind:     StructAstKind,
					expected: []TokKind{TokField, TokMessage, TokRBrace},
				},
			},
		},
		{
			name: "InvalidArraySize",
			input: `
			message Data struct {
				required one @1 [5a]b128;
			}`,
			asts: []Ast{
				&StructAst{
					Name: "Data",
					Fields: []FieldAst{
						{Tags: Tags{Poisoned: true}, Modifier: Required, Name: "one", Ord: 1},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokErr, Value: "5a", Expected: TokInteger}, Range{B: 47, E: 49}},
					kind:     ArrayAstKind,
					expected: []TokKind{TokInteger, TokRBrack},
				},
			},
		},
		{
			name: "InvalidFields",
			input: `
			message Data1 struct {
				required one @1abc b128;
				two @2abc []b5;
		
				message Data2 struct {
					required one @1;
				}
			}`,
			asts: []Ast{
				&StructAst{
					Tags: Tags{Poisoned: true},
					Name: "Data1",
					Fields: []FieldAst{
						{Tags: Tags{Poisoned: true}, Modifier: Required, Name: "one"},
					},
					LocalDefs: []Ast{
						&StructAst{
							Name: "Data2",
							Fields: []FieldAst{
								{Tags: Tags{Poisoned: true}, Modifier: Required, Name: "one", Ord: 1},
							},
						},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokErr, Value: "@1abc", Expected: TokOrd}, Range{B: 44, E: 49}},
					kind:     FieldAstKind,
					expected: []TokKind{TokOrd},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "two"}, Range{B: 60, E: 63}},
					kind:     StructAstKind,
					expected: []TokKind{TokField, TokMessage, TokRBrace},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokSemicolon, Value: ";"}, Range{B: 126, E: 127}},
					kind:     FieldAstKind,
					expected: []TokKind{TokType},
				},
			},
		},
		{
			name: "InvalidUnion",
			input: `
			message Data3 struct {
				required one @1;
		
				message Data4 union {
					@1 One;
					@2 5;
					Two;
				}
			}`,
			asts: []Ast{
				&StructAst{
					Name: "Data3",
					Fields: []FieldAst{
						{Tags: Tags{Poisoned: true}, Modifier: Required, Name: "one", Ord: 1},
					},
					LocalDefs: []Ast{
						&UnionAst{
							Tags: Tags{Poisoned: true},
							Name: "Data4",
							Options: []OptionAst{
								{Type: &TypeRefAst{Iden: "One"}, Ord: 1},
								{Tags: Tags{Poisoned: true}, Ord: 2},
							},
						},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokSemicolon, Value: ";"}, Range{B: 46, E: 47}},
					kind:     FieldAstKind,
					expected: []TokKind{TokType},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokInteger, Value: "5"}, Range{B: 98, E: 99}},
					kind:     OptionAstKind,
					expected: []TokKind{TokType},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "Two"}, Range{B: 106, E: 109}},
					kind:     UnionAstKind,
					expected: []TokKind{TokOption, TokMessage, TokRBrace},
				},
			},
		},
		{
			name: "InvalidEnum",
			input: `
			message Data4 enum {
				@1 ONE;
				@2 2 TWO;
				THREE;
			}`,
			asts: []Ast{
				&EnumAst{
					Tags: Tags{Poisoned: true},
					Name: "Data4",
					Cases: []CaseAst{
						{Name: "ONE", Ord: 1},
						{Tags: Tags{Poisoned: true}, Ord: 2},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokInteger, Value: "2"}, Range{B: 44, E: 45}},
					kind:     CaseAstKind,
					expected: []TokKind{TokIden},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "THREE"}, Range{B: 55, E: 60}},
					kind:     EnumAstKind,
					expected: []TokKind{TokCase, TokRBrace},
				},
			},
		},
		{
			name: "DuplicatedIdentifiers",
			input: `
			message Data struct {
				required one one one one one @1 b128;
				two two two two two @2 []b5;
				required three @3 b128
			}`,
			asts: []Ast{
				&StructAst{
					Tags: Tags{Poisoned: true},
					Name: "Data",
					Fields: []FieldAst{
						{Tags: Tags{Poisoned: true}, Modifier: Required, Name: "one"},
						{Tags: Tags{Poisoned: true}, Modifier: Required, Name: "three", Ord: 3, Type: &TypeRefAst{Iden: "b128"}},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "one"}, Range{B: 43, E: 46}},
					kind:     FieldAstKind,
					expected: []TokKind{TokOrd},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "two"}, Range{B: 72, E: 75}},
					kind:     StructAstKind,
					expected: []TokKind{TokField, TokMessage, TokRBrace},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokRBrace, Value: "}"}, Range{B: 131, E: 132}},
					kind:     FieldAstKind,
					expected: []TokKind{TokSemicolon},
				},
			},
		},
		{
			name: "InvalidRpc",
			input: `
			service Data {
				rpc @1 Hello(Test) (Output)
				required one @1 b128;
				rpc @2 World(Test1) returns ()
			}`,
			asts: []Ast{
				&ServiceAst{
					Tags: Tags{Poisoned: true},
					Name: "Data",
					Procedures: []RpcAst{
						{Tags: Tags{Poisoned: true}, Name: "Hello", Ord: 1, Arg: &TypeRefAst{Iden: "Test"}},
						{Tags: Tags{Poisoned: true}, Name: "World", Ord: 2, Arg: &TypeRefAst{Iden: "Test1"}},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokLParen, Value: "("}, Range{B: 42, E: 43}},
					kind:     RpcAstKind,
					expected: []TokKind{TokReturns},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokRequired, Value: "required"}, Range{B: 55, E: 63}},
					kind:     ServiceAstKind,
					expected: []TokKind{TokRpc, TokMessage, TokRBrace},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokRParen, Value: ")"}, Range{B: 110, E: 111}},
					kind:     RpcAstKind,
					expected: []TokKind{TokType},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("test/%s", test.name), func(t *testing.T) {
			asts, errs := runParser(test.input)
			WalkList(Ast.ClearPos, asts)

			printLine := func(err string) {
				t.Log(err)
			}
			printErrors(errs, "test", printLine)

			assert.Equal(t, test.asts, asts)
			assert.Equal(t, test.errs, errs)
		})
	}
}

func TestParser_Garbage(t *testing.T) {
	input := `
	hello world service struct field}
	lorem; ipsum 5a{ test 123 go there
	`

	done := make(chan struct{})

	go func() {
		runParser(input)
		close(done)
	}()

	select {
	case <-time.After(time.Second):
		t.Fatal("garbage parser test has timed out, is there an infinite loop?")
	case <-done:
		t.Log("finished garbage parser test")
	}
}
