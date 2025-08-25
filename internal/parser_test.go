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
	ClearAll(nodes)

	expectedNodes := []Node{
		&ImportNode{Path: "/services/schemas/animals"},
		&PropertyNode{Name: "package", Value: "/hello/\\\"world\""},
		&PropertyNode{Name: "constant", Value: "Value"},
	}

	assert.Equal(t, expectedNodes, nodes)
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
	
			message Data3 struct(A, B) {
				deprecated one @1 A;;;
				required two @2 B;
			}
		}
	}
	`
	var errs []error
	nodes := runParser(input, &errs)
	ClearAll(nodes)

	expectedNodes := []Node{
		&StructNode{
			Name: "Data1",
			Fields: []FieldNode{
				{Modifier: Required, Name: "one", Ord: 1, Type: TypeNode{TypeVal: TypeVal{Iden: "b128"}}},
				{
					Modifier: Required,
					Name:     "two",
					Ord:      2,
					Type:     TypeNode{TypeVal: TypeVal{Iden: "b5"}, Array: []uint64{0}},
				},
				{
					Modifier: Optional,
					Name:     "three",
					Ord:      3,
					Type:     TypeNode{TypeVal: TypeVal{Iden: "b4"}, Array: []uint64{16}},
				},
				{
					Modifier: Optional,
					Name:     "four",
					Ord:      4,
					Type:     TypeNode{TypeVal: TypeVal{Iden: "b4"}, Array: []uint64{0, 4, 0}},
				},
			},
			LocalDefs: []Node{
				&StructNode{
					Name: "Data2",
					Fields: []FieldNode{
						{Modifier: Required, Name: "one", Ord: 1, Type: TypeNode{TypeVal: TypeVal{Iden: "Data3"}}},
					},
					LocalDefs: []Node{
						&StructNode{
							Name:       "Data3",
							TypeParams: []string{"A", "B"},
							Fields: []FieldNode{
								{Modifier: Deprecated, Name: "one", Ord: 1, Type: TypeNode{TypeVal: TypeVal{Iden: "A"}}},
								{Modifier: Required, Name: "two", Ord: 2, Type: TypeNode{TypeVal: TypeVal{Iden: "B"}}},
							},
						},
					},
				},
			},
		},
	}

	assert.Equal(t, expectedNodes, nodes)
	assert.Empty(t, errs)
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
	ClearAll(nodes)

	expectedNodes := []Node{
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

	assert.Equal(t, expectedNodes, nodes)
	assert.Nil(t, errs)
}

func TestParser_Union(t *testing.T) {
	input := `
	message Data [8]union(A, B, C) {
		@1 Data2;
		@2 Data1;
		@3 Data;

		message Data union() { 
			@1 B;;;
			@2 C;
        }
	}
	`
	var errs []error
	nodes := runParser(input, &errs)
	ClearAll(nodes)

	expectedNodes := []Node{
		&UnionNode{
			Name:       "Data",
			Size:       8,
			TypeParams: []string{"A", "B", "C"},
			Options: []OptionNode{
				{Ord: 1, Type: TypeVal{Iden: "Data2"}},
				{Ord: 2, Type: TypeVal{Iden: "Data1"}},
				{Ord: 3, Type: TypeVal{Iden: "Data"}},
			},
			LocalDefs: []Node{
				&UnionNode{
					Name: "Data",
					Size: 16,
					Options: []OptionNode{
						{Ord: 1, Type: TypeVal{Iden: "B"}},
						{Ord: 2, Type: TypeVal{Iden: "C"}},
					},
				},
			},
		},
	}

	assert.Equal(t, expectedNodes, nodes)
	assert.Nil(t, errs)
}

func TestParser_Service(t *testing.T) {
	input := `
	service ServiceA {
		rpc @1 Hello(Test) returns (Output)
		rpc @2 World(Test1(Arg1, Arg2, Arg3)) returns (Output1(Arg1, Arg2))

		message Test struct {
			required one @1 b24;
		}
	}
	`
	var errs []error
	nodes := runParser(input, &errs)
	ClearAll(nodes)

	expectedNodes := []Node{
		&ServiceNode{
			Name: "ServiceA",
			Procedures: []RpcNode{
				{
					Ord:  1,
					Name: "Hello",
					Arg:  TypeNode{TypeVal: TypeVal{Iden: "Test"}},
					Ret:  TypeNode{TypeVal: TypeVal{Iden: "Output"}},
				},
				{
					Ord:  2,
					Name: "World",
					Arg: TypeNode{
						TypeVal:  TypeVal{Iden: "Test1"},
						TypeArgs: []TypeNode{{TypeVal: TypeVal{Iden: "Arg1"}}, {TypeVal: TypeVal{Iden: "Arg2"}}, {TypeVal: TypeVal{Iden: "Arg3"}}},
					},
					Ret: TypeNode{
						TypeVal:  TypeVal{Iden: "Output1"},
						TypeArgs: []TypeNode{{TypeVal: TypeVal{Iden: "Arg1"}}, {TypeVal: TypeVal{Iden: "Arg2"}}},
					},
				},
			},
			LocalDefs: []Node{
				&StructNode{
					Name: "Test",
					Fields: []FieldNode{
						{Modifier: Required, Name: "one", Ord: 1, Type: TypeNode{TypeVal: TypeVal{Iden: "b24"}}},
					},
				},
			},
		},
	}

	assert.Equal(t, expectedNodes, nodes)
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
					Poisoned: true,
					Name:     "Data1",
					Fields: []FieldNode{
						{Modifier: Required, Name: "one", Ord: 1, Type: TypeNode{TypeVal: TypeVal{Iden: "b128"}}},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokEof, Value: ""}, Positions{B: 56, E: 56}},
					nodeKind: StructNodeKind,
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
					Poisoned: true,
					Name:     "Data1",
					LocalDefs: []Node{
						&UnionNode{
							Poisoned: true,
							Size:     16,
							Name:     "Data2",
							LocalDefs: []Node{
								&StructNode{Poisoned: true, Name: "Data3"},
							},
						},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokEof, Value: ""}, Positions{B: 84, E: 84}},
					nodeKind: StructNodeKind,
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
						{Modifier: Required, Name: "one", Ord: 1, Type: TypeNode{TypeVal: TypeVal{Iden: "b128"}}},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokInteger, Value: "5"}, Positions{B: 18, E: 19}},
					nodeKind: MessageNodeKind,
					errKind:  SizeErrKind,
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
						{Poisoned: true, Modifier: Required, Name: "one", Ord: 1},
					},
				},
				&StructNode{
					Name:     "Data_1",
					Poisoned: true,
					Fields: []FieldNode{
						{Modifier: Required, Name: "one", Ord: 1, Type: TypeNode{TypeVal: TypeVal{Iden: "b128"}}},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokErr, Value: "5a", Expected: TokInteger}, Positions{B: 47, E: 49}},
					nodeKind: TypeNodeKind,
					expected: []TokKind{TokInteger, TokRBrack},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "Data_1"}, Positions{B: 72, E: 78}},
					nodeKind: MessageNodeKind,
					errKind:  IdenErrKind,
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
					Poisoned: true,
					Name:     "Data1",
					Fields: []FieldNode{
						{Poisoned: true, Modifier: Required, Name: "one"},
					},
					LocalDefs: []Node{
						&StructNode{
							Name: "Data2",
							Fields: []FieldNode{
								{Poisoned: true, Modifier: Required, Name: "one", Ord: 1},
							},
						},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokErr, Value: "@1abc", Expected: TokOrd}, Positions{B: 44, E: 49}},
					nodeKind: FieldNodeKind,
					expected: []TokKind{TokOrd},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "two"}, Positions{B: 60, E: 63}},
					nodeKind: StructNodeKind,
					expected: []TokKind{TokField, TokMessage, TokRBrace},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokSemicolon, Value: ";"}, Positions{B: 126, E: 127}},
					nodeKind: FieldNodeKind,
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
						{Poisoned: true, Modifier: Required, Name: "one", Ord: 1},
					},
					LocalDefs: []Node{
						&UnionNode{
							Poisoned: true,
							Name:     "Data4",
							Size:     16,
							Options: []OptionNode{
								{Type: TypeVal{Iden: "One"}, Ord: 1},
								{Poisoned: true, Ord: 2},
							},
						},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokSemicolon, Value: ";"}, Positions{B: 46, E: 47}},
					nodeKind: FieldNodeKind,
					expected: []TokKind{TokTypeRef},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokInteger, Value: "5"}, Positions{B: 98, E: 99}},
					nodeKind: OptionNodeKind,
					expected: []TokKind{TokIden},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "Two"}, Positions{B: 106, E: 109}},
					nodeKind: UnionNodeKind,
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
					Poisoned: true,
					Name:     "Data4",
					Size:     4,
					Cases: []CaseNode{
						{Name: "ONE", Ord: 1},
						{Poisoned: true, Ord: 2},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokInteger, Value: "2"}, Positions{B: 47, E: 48}},
					nodeKind: CaseNodeKind,
					expected: []TokKind{TokIden},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "THREE"}, Positions{B: 58, E: 63}},
					nodeKind: EnumNodeKind,
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
					Poisoned: true,
					Name:     "Data",
					Fields: []FieldNode{
						{Poisoned: true, Modifier: Required, Name: "one"},
						{Poisoned: true, Modifier: Required, Name: "three", Ord: 3, Type: TypeNode{TypeVal: TypeVal{Iden: "b128"}}},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "one"}, Positions{B: 43, E: 46}},
					nodeKind: FieldNodeKind,
					expected: []TokKind{TokOrd},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "two"}, Positions{B: 72, E: 75}},
					nodeKind: StructNodeKind,
					expected: []TokKind{TokField, TokMessage, TokRBrace},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokRBrace, Value: "}"}, Positions{B: 131, E: 132}},
					nodeKind: FieldNodeKind,
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
					Poisoned: true,
					Name:     "Data",
					Procedures: []RpcNode{
						{Poisoned: true, Name: "Hello", Ord: 1, Arg: TypeNode{TypeVal: TypeVal{Iden: "Test"}}},
						{Poisoned: true, Name: "World", Ord: 2, Arg: TypeNode{TypeVal: TypeVal{Iden: "Test1"}}},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokLParen, Value: "("}, Positions{B: 42, E: 43}},
					nodeKind: RpcNodeKind,
					expected: []TokKind{TokReturns},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokRequired, Value: "required"}, Positions{B: 55, E: 63}},
					nodeKind: ServiceNodeKind,
					expected: []TokKind{TokRpc, TokMessage, TokRBrace},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokRParen, Value: ")"}, Positions{B: 110, E: 111}},
					nodeKind: RpcNodeKind,
					expected: []TokKind{TokTypeRef},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("test/%s", test.name), func(t *testing.T) {
			var errs []error
			nodes := runParser(test.input, &errs)
			WalkMetaList(Node.Clear, nodes)

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
