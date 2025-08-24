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

	var errs []error
	nodes := runParser(input, &errs)
	WalkMetaList(Node.ClearPos, nodes)

	expectednodes := []Node{
		&ImportNode{Path: "/services/schemas/animals"},
		&PropertyNode{Name: "package", Value: "/hello/\\\"world\""},
		&PropertyNode{Name: "constant", Value: "Value"},
	}

	assert.Equal(t, expectednodes, nodes)
	assert.Nil(t, errs)
}

func TestParser_Struct(t *testing.T) {
	input := `
	message Data1 struct {
		required one @1 b128; // this is the first comment
		required two @2 []b5; // this is the second comment
		optional three @3 [16]b4;
		optional four @4 [][4][]b4;;;

		message Data2 struct {
			required one @1 Data3;
	
			message Data3 struct(A B) {
				deprecated one @1 A;;;
				required two @2 B;
			}
		}
	}
	`
	var errs []error
	nodes := runParser(input, &errs)
	WalkMetaList(Node.ClearPos, nodes)

	expectednodes := []Node{
		&StructNode{
			Name: "Data1",
			Fields: []FieldNode{
				{Modifier: Required, Name: "one", Ord: 1, Type: TypeRefNode{Iden: "b128"}},
				{
					Modifier: Required,
					Name:     "two",
					Ord:      2,
					Type:     TypeRefNode{Iden: "b5", Array: []uint64{0}},
				},
				{
					Modifier: Optional,
					Name:     "three",
					Ord:      3,
					Type:     TypeRefNode{Iden: "b4", Array: []uint64{16}},
				},
				{
					Modifier: Optional,
					Name:     "four",
					Ord:      4,
					Type:     TypeRefNode{Iden: "b4", Array: []uint64{0, 4, 0}},
				},
			},
			LocalDefs: []Node{
				&StructNode{
					Name: "Data2",
					Fields: []FieldNode{
						{Modifier: Required, Name: "one", Ord: 1, Type: TypeRefNode{Iden: "Data3"}},
					},
					LocalDefs: []Node{
						&StructNode{
							Name:       "Data3",
							TypeParams: []string{"A", "B"},
							Fields: []FieldNode{
								{Modifier: Deprecated, Name: "one", Ord: 1, Type: TypeRefNode{Iden: "A"}},
								{Modifier: Required, Name: "two", Ord: 2, Type: TypeRefNode{Iden: "B"}},
							},
						},
					},
				},
			},
		},
	}

	assert.Equal(t, expectednodes, nodes)
	assert.Nil(t, errs)
}

func TestParser_Enum(t *testing.T) {
	input := `
	message Data1 [16]enum {
		@1 One;;
		@2 Two;
		@3 Three;;;
	}
	`
	var errs []error
	nodes := runParser(input, &errs)
	WalkMetaList(Node.ClearPos, nodes)

	expectednodes := []Node{
		&EnumNode{
			Name: "Data1",
			Size: 16,
			Cases: []CaseNode{
				{Ord: 1, Name: "One"},
				{Ord: 2, Name: "Two"},
				{Ord: 3, Name: "Three"},
			},
		},
	}

	assert.Equal(t, expectednodes, nodes)
	assert.Nil(t, errs)
}

func TestParser_Union(t *testing.T) {
	input := `
	message Data [8]union(A B C) {
		@1 Data2(Data);
		@2 []Data1;
		@3 Data;

		message Data union() { 
			@1 B;;;
			@2 C;
        }
	}
	`
	var errs []error
	nodes := runParser(input, &errs)
	WalkMetaList(Node.ClearPos, nodes)

	expectednodes := []Node{
		&UnionNode{
			Name:       "Data",
			Size:       8,
			TypeParams: []string{"A", "B", "C"},
			Options: []OptionNode{
				{
					Ord: 1,
					Type: TypeRefNode{
						Iden:     "Data2",
						TypeArgs: []TypeRefNode{{Iden: "Data"}},
					},
				},
				{
					Ord:  2,
					Type: TypeRefNode{Iden: "Data1", Array: []uint64{0}},
				},
				{Ord: 3, Type: TypeRefNode{Iden: "Data"}},
			},
			LocalDefs: []Node{
				&UnionNode{
					Name: "Data",
					Size: 16,
					Options: []OptionNode{
						{Ord: 1, Type: TypeRefNode{Iden: "B"}},
						{Ord: 2, Type: TypeRefNode{Iden: "C"}},
					},
				},
			},
		},
	}

	assert.Equal(t, expectednodes, nodes)
	assert.Nil(t, errs)
}

func TestParser_Service(t *testing.T) {
	input := `
	service ServiceA {
		rpc @1 Hello(Test) returns (Output)
		rpc @2 World(Test1(Arg1 Arg2 Arg3)) returns (Output1(Arg1 Arg2))

		message Test struct {
			required one @1 b24;
		}
	}
	`
	var errs []error
	nodes := runParser(input, &errs)
	WalkMetaList(Node.ClearPos, nodes)

	expectednodes := []Node{
		&ServiceNode{
			Name: "ServiceA",
			Procedures: []RpcNode{
				{Ord: 1, Name: "Hello", Arg: TypeRefNode{Iden: "Test"}, Ret: TypeRefNode{Iden: "Output"}},
				{
					Ord:  2,
					Name: "World",
					Arg: TypeRefNode{
						Iden:     "Test1",
						TypeArgs: []TypeRefNode{{Iden: "Arg1"}, {Iden: "Arg2"}, {Iden: "Arg3"}},
					},
					Ret: TypeRefNode{
						Iden:     "Output1",
						TypeArgs: []TypeRefNode{{Iden: "Arg1"}, {Iden: "Arg2"}},
					},
				},
			},
			LocalDefs: []Node{
				&StructNode{
					Name: "Test",
					Fields: []FieldNode{
						{Modifier: Required, Name: "one", Ord: 1, Type: TypeRefNode{Iden: "b24"}},
					},
				},
			},
		},
	}

	assert.Equal(t, expectednodes, nodes)
	assert.Nil(t, errs)
}

func TestParser_Errors(t *testing.T) {
	type Test struct {
		name  string
		input string
		nodes []Node
		errs  []error
	}

	tests := []Test{
		{
			name: "UnclosedStruct",
			input: `
			message Data1 struct {
				required one @1 b128;
			`,
			nodes: []Node{
				&StructNode{
					Tags: Tags{Poisoned: true},
					Name: "Data1",
					Fields: []FieldNode{
						{Modifier: Required, Name: "one", Ord: 1, Type: TypeRefNode{Iden: "b128"}},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokEof, Value: ""}, Positions{B: 56, E: 56}},
					kind:     StructNodeKind,
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
			nodes: []Node{
				&StructNode{
					Tags: Tags{Poisoned: true},
					Name: "Data1",
					LocalDefs: []Node{
						&UnionNode{
							Tags: Tags{Poisoned: true},
							Size: 16,
							Name: "Data2",
							LocalDefs: []Node{
								&StructNode{Tags: Tags{Poisoned: true}, Name: "Data3"},
							},
						},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokEof, Value: ""}, Positions{B: 84, E: 84}},
					kind:     StructNodeKind,
					expected: []TokKind{TokField, TokMessage, TokRBrace},
				},
			},
		},
		{
			name: "InvalidMessageSize",
			input: `
			message Data [5]struct {
				required one @1 b128;
			}`,
			nodes: []Node{
				&StructNode{
					Name: "Data",
					Fields: []FieldNode{
						{Modifier: Required, Name: "one", Ord: 1, Type: TypeRefNode{Iden: "b128"}},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual: Token{TokVal{Kind: TokInteger, Value: "5"}, Positions{B: 18, E: 19}},
					kind:   MessageNodeKind,
					text:   "struct does not allow a size argument",
				},
			},
		},
		{
			name: "InvalidStruct",
			input: `
			message Data struct {
				required one @1 [5a]b128;
			}
			message Data_1 struct {
				required one @1 b128;
			}`,
			nodes: []Node{
				&StructNode{
					Name: "Data",
					Fields: []FieldNode{
						{Tags: Tags{Poisoned: true}, Modifier: Required, Name: "one", Ord: 1},
					},
				},
				&StructNode{
					Name: "Data_1",
					Tags: Tags{Poisoned: true},
					Fields: []FieldNode{
						{Modifier: Required, Name: "one", Ord: 1, Type: TypeRefNode{Iden: "b128"}},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokErr, Value: "5a", Expected: TokInteger}, Positions{B: 47, E: 49}},
					kind:     TypeRefNodeKind,
					expected: []TokKind{TokInteger, TokRBrack},
				},
				&ParsingErr{
					actual: Token{TokVal{Kind: TokIden, Value: "Data_1"}, Positions{B: 72, E: 78}},
					kind:   MessageNodeKind,
					text:   "iden must only contain alphanumeric characters",
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
			nodes: []Node{
				&StructNode{
					Tags: Tags{Poisoned: true},
					Name: "Data1",
					Fields: []FieldNode{
						{Tags: Tags{Poisoned: true}, Modifier: Required, Name: "one"},
					},
					LocalDefs: []Node{
						&StructNode{
							Name: "Data2",
							Fields: []FieldNode{
								{Tags: Tags{Poisoned: true}, Modifier: Required, Name: "one", Ord: 1},
							},
						},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokErr, Value: "@1abc", Expected: TokOrd}, Positions{B: 44, E: 49}},
					kind:     FieldNodeKind,
					expected: []TokKind{TokOrd},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "two"}, Positions{B: 60, E: 63}},
					kind:     StructNodeKind,
					expected: []TokKind{TokField, TokMessage, TokRBrace},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokSemicolon, Value: ";"}, Positions{B: 126, E: 127}},
					kind:     FieldNodeKind,
					expected: []TokKind{TokTypeRef},
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
			nodes: []Node{
				&StructNode{
					Name: "Data3",
					Fields: []FieldNode{
						{Tags: Tags{Poisoned: true}, Modifier: Required, Name: "one", Ord: 1},
					},
					LocalDefs: []Node{
						&UnionNode{
							Tags: Tags{Poisoned: true},
							Name: "Data4",
							Size: 16,
							Options: []OptionNode{
								{Type: TypeRefNode{Iden: "One"}, Ord: 1},
								{Tags: Tags{Poisoned: true}, Ord: 2},
							},
						},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokSemicolon, Value: ";"}, Positions{B: 46, E: 47}},
					kind:     FieldNodeKind,
					expected: []TokKind{TokTypeRef},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokInteger, Value: "5"}, Positions{B: 98, E: 99}},
					kind:     OptionNodeKind,
					expected: []TokKind{TokTypeRef},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "Two"}, Positions{B: 106, E: 109}},
					kind:     UnionNodeKind,
					expected: []TokKind{TokOption, TokMessage, TokRBrace},
				},
			},
		},
		{
			name: "InvalidEnum",
			input: `
			message Data4 [4]enum {
				@1 ONE;
				@2 2 TWO;
				THREE;
			}`,
			nodes: []Node{
				&EnumNode{
					Tags: Tags{Poisoned: true},
					Name: "Data4",
					Size: 4,
					Cases: []CaseNode{
						{Name: "ONE", Ord: 1},
						{Tags: Tags{Poisoned: true}, Ord: 2},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokInteger, Value: "2"}, Positions{B: 47, E: 48}},
					kind:     CaseNodeKind,
					expected: []TokKind{TokIden},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "THREE"}, Positions{B: 58, E: 63}},
					kind:     EnumNodeKind,
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
			nodes: []Node{
				&StructNode{
					Tags: Tags{Poisoned: true},
					Name: "Data",
					Fields: []FieldNode{
						{Tags: Tags{Poisoned: true}, Modifier: Required, Name: "one"},
						{Tags: Tags{Poisoned: true}, Modifier: Required, Name: "three", Ord: 3, Type: TypeRefNode{Iden: "b128"}},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "one"}, Positions{B: 43, E: 46}},
					kind:     FieldNodeKind,
					expected: []TokKind{TokOrd},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "two"}, Positions{B: 72, E: 75}},
					kind:     StructNodeKind,
					expected: []TokKind{TokField, TokMessage, TokRBrace},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokRBrace, Value: "}"}, Positions{B: 131, E: 132}},
					kind:     FieldNodeKind,
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
			nodes: []Node{
				&ServiceNode{
					Tags: Tags{Poisoned: true},
					Name: "Data",
					Procedures: []RpcNode{
						{Tags: Tags{Poisoned: true}, Name: "Hello", Ord: 1, Arg: TypeRefNode{Iden: "Test"}},
						{Tags: Tags{Poisoned: true}, Name: "World", Ord: 2, Arg: TypeRefNode{Iden: "Test1"}},
					},
				},
			},
			errs: []error{
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokLParen, Value: "("}, Positions{B: 42, E: 43}},
					kind:     RpcNodeKind,
					expected: []TokKind{TokReturns},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokRequired, Value: "required"}, Positions{B: 55, E: 63}},
					kind:     ServiceNodeKind,
					expected: []TokKind{TokRpc, TokMessage, TokRBrace},
				},
				&ParsingErr{
					actual:   Token{TokVal{Kind: TokRParen, Value: ")"}, Positions{B: 110, E: 111}},
					kind:     RpcNodeKind,
					expected: []TokKind{TokTypeRef},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("test/%s", test.name), func(t *testing.T) {
			var errs []error
			nodes := runParser(test.input, &errs)
			WalkMetaList(Node.ClearPos, nodes)

			printLine := func(err string) {
				t.Log(err)
			}
			printErrors(errs, "test", printLine)

			assert.Equal(t, test.nodes, nodes)
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
		var errs []error
		_ = runParser(input, &errs)
		// we throw away asts and don't do anything with the errs - just check that this terminates
		close(done)
	}()

	select {
	case <-time.After(time.Second):
		t.Fatal("garbage parser test has timed out, is there an infinite loop?")
	case <-done:
		t.Log("finished garbage parser test")
	}
}
