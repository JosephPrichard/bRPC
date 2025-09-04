package internal

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParser_Properties(t *testing.T) {
	input := `
	import "/services/schemas/animals"

	package = "/hello/\\\"world\""
	constant = "Value"
	`

	var errs []error
	nodes := runParser(input, &errs)
	ClearNodeList(nodes)

	t.Logf("\n%s\n", WriteAst(nodes))

	expectedNodes := []DefNode{
		{Kind: ImportNodeKind, Value: "/services/schemas/animals"},
		{Kind: PropertyNodeKind, Iden: "package", Value: "/hello/\\\"world\""},
		{Kind: PropertyNodeKind, Iden: "constant", Value: "Value"},
	}

	assert.Equal(t, expectedNodes, nodes)
	assert.Empty(t, errs)
}

func TestParser_Struct(t *testing.T) {
	input := `
	message Data1 struct {
		required one @1 int128; // this is the first comment
		required two @2 []int5; // this is the second comment
		optional three @3 [16]int4;
		optional four @4 [][4][]int4;;;

		message Data2 struct() {
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
	ClearNodeList(nodes)

	t.Logf("\n%s\n", WriteAst(nodes))

	expectedNodes := []DefNode{
		{
			Kind: StructNodeKind,
			Iden: "Data1",
			Members: []MembNode{
				{Modifier: Required, Iden: "one", Ord: 1, LType: TypeNode{Iden: "int128"}},
				{Modifier: Required, Iden: "two", Ord: 2, LType: TypeNode{Iden: "int5", Array: []uint64{0}}},
				{Modifier: Optional, Iden: "three", Ord: 3, LType: TypeNode{Iden: "int4", Array: []uint64{16}}},
				{Modifier: Optional, Iden: "four", Ord: 4, LType: TypeNode{Iden: "int4", Array: []uint64{0, 4, 0}}},
			},
			LocalDefs: []DefNode{
				{
					Kind: StructNodeKind,
					Iden: "Data2",
					Members: []MembNode{
						{Modifier: Required, Iden: "one", Ord: 1, LType: TypeNode{Iden: "Data3"}},
					},
					LocalDefs: []DefNode{
						{
							Kind:       StructNodeKind,
							Iden:       "Data3",
							TypeParams: []string{"A", "B"},
							Members: []MembNode{
								{Modifier: Deprecated, Iden: "one", Ord: 1, LType: TypeNode{Iden: "A"}},
								{Modifier: Required, Iden: "two", Ord: 2, LType: TypeNode{Iden: "B"}},
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
	ClearNodeList(nodes)

	t.Logf("\n%s\n", WriteAst(nodes))

	expectedNodes := []DefNode{
		{
			Kind: EnumNodeKind,
			Iden: "Data1",
			Size: 16,
			Members: []MembNode{
				{Ord: 1, Iden: "One"},
				{Ord: 2, Iden: "Two"},
				{Ord: 3, Iden: "Three"},
			},
		},
	}

	assert.Equal(t, expectedNodes, nodes)
	assert.Empty(t, errs)
}

func TestParser_Union(t *testing.T) {
	input := `
	message Data [8]union(A, B, C) {
		one @1 Data2;
		two @2 Data1;
		three @3 []Data;;

		message Data union { 
			one @1 B;
			two @2 C;
        }
	}
	`
	var errs []error
	nodes := runParser(input, &errs)
	ClearNodeList(nodes)

	t.Logf("\n%s\n", WriteAst(nodes))

	expectedNodes := []DefNode{
		{
			Kind:       UnionNodeKind,
			Iden:       "Data",
			Size:       8,
			TypeParams: []string{"A", "B", "C"},
			Members: []MembNode{
				{Ord: 1, Iden: "one", LType: TypeNode{Iden: "Data2"}},
				{Ord: 2, Iden: "two", LType: TypeNode{Iden: "Data1"}},
				{Ord: 3, Iden: "three", LType: TypeNode{Iden: "Data", Array: []uint64{0}}},
			},
			LocalDefs: []DefNode{
				{
					Kind: UnionNodeKind,
					Iden: "Data",
					Size: 16,
					Members: []MembNode{
						{Ord: 1, Iden: "one", LType: TypeNode{Iden: "B"}},
						{Ord: 2, Iden: "two", LType: TypeNode{Iden: "C"}},
					},
				},
			},
		},
	}

	assert.Equal(t, expectedNodes, nodes)
	assert.Empty(t, errs)
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
	ClearNodeList(nodes)

	t.Logf("\n%s\n", WriteAst(nodes))

	expectedNodes := []DefNode{
		{
			Kind: ServiceNodeKind,
			Iden: "ServiceA",
			Members: []MembNode{
				{Ord: 1, Iden: "Hello", LType: TypeNode{Iden: "Test"}, RType: TypeNode{Iden: "Output"}},
				{
					Ord:   2,
					Iden:  "World",
					LType: TypeNode{Iden: "Test1", TypeArgs: []TypeNode{{Iden: "Arg1"}, {Iden: "Arg2"}, {Iden: "Arg3"}}},
					RType: TypeNode{Iden: "Output1", TypeArgs: []TypeNode{{Iden: "Arg1"}, {Iden: "Arg2"}}},
				},
			},
			LocalDefs: []DefNode{
				{
					Kind: StructNodeKind,
					Iden: "Test",
					Members: []MembNode{
						{Modifier: Required, Iden: "one", Ord: 1, LType: TypeNode{Iden: "b24"}},
					},
				},
			},
		},
	}

	assert.Equal(t, expectedNodes, nodes)
	assert.Empty(t, errs)
}

func TestParser_Errors(t *testing.T) {
	type Test struct {
		name  string
		input string
		nodes []DefNode
		errs  []error
	}

	tests := []Test{
		{
			name:  "UnclosedStruct",
			input: `message Data1 struct { required one @1 int128;`,
			nodes: []DefNode{
				{
					Kind:     StructNodeKind,
					Poisoned: true,
					Iden:     "Data1",
					Members: []MembNode{
						{Modifier: Required, Iden: "one", Ord: 1, LType: TypeNode{Iden: "int128"}},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokEof, Value: ""}, Positions{}},
					nodeKind: StructNodeKind,
					expected: []TokKind{TokField, TokMessage, TokRBrace},
				},
			},
		},
		{
			name:  "UnclosedNestedStruct",
			input: `message Data1 struct { message Data2 union { message Data3 struct {`,
			nodes: []DefNode{
				{
					Kind:     StructNodeKind,
					Poisoned: true,
					Iden:     "Data1",
					LocalDefs: []DefNode{
						{
							Kind:     UnionNodeKind,
							Poisoned: true,
							Size:     16,
							Iden:     "Data2",
							LocalDefs: []DefNode{
								{Kind: StructNodeKind, Poisoned: true, Iden: "Data3"},
							},
						},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokEof, Value: ""}, Positions{}},
					nodeKind: StructNodeKind,
					expected: []TokKind{TokField, TokMessage, TokRBrace},
				},
			},
		},
		{
			name:  "InvalidMessageSize",
			input: `message Data [5]struct { required one @1 int128; }`,
			nodes: []DefNode{
				{
					Kind: StructNodeKind,
					Iden: "Data",
					Members: []MembNode{
						{Modifier: Required, Iden: "one", Ord: 1, LType: TypeNode{Iden: "int128"}},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokInteger, Value: "5", Num: 5}, Positions{}},
					nodeKind: MessageNodeKind,
					errKind:  SizeErrKind,
				},
			},
		},
		{
			name:  "InvalidStruct",
			input: `message Data struct { required one @1 [5a]int128; } message Data_1 struct { required one @1 int128; }`,
			nodes: []DefNode{
				{
					Kind: StructNodeKind,
					Iden: "Data",
					Members: []MembNode{
						{Poisoned: true, Modifier: Required, Iden: "one", Ord: 1},
					},
				},
				{
					Kind:     StructNodeKind,
					Iden:     "Data_1",
					Poisoned: true,
					Members: []MembNode{
						{Modifier: Required, Iden: "one", Ord: 1, LType: TypeNode{Iden: "int128"}},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokErr, Value: "5a", Expected: TokInteger}, Positions{}},
					nodeKind: TypeNodeKind,
					expected: []TokKind{TokInteger, TokRBrack},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "Data_1"}, Positions{}},
					nodeKind: MessageNodeKind,
					errKind:  IdenErrKind,
				},
			},
		},
		{
			name:  "InvalidFields",
			input: `message Data1 struct { required one @1abc int128; two @2abc []int5; message Data2 struct { required one @1; } }`,
			nodes: []DefNode{
				{
					Kind:     StructNodeKind,
					Poisoned: true,
					Iden:     "Data1",
					Members: []MembNode{
						{Poisoned: true, Modifier: Required, Iden: "one"},
					},
					LocalDefs: []DefNode{
						{
							Kind: StructNodeKind,
							Iden: "Data2",
							Members: []MembNode{
								{Poisoned: true, Modifier: Required, Iden: "one", Ord: 1},
							},
						},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokErr, Value: "@1abc", Expected: TokOrd}, Positions{}},
					nodeKind: FieldNodeKind,
					expected: []TokKind{TokOrd},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "two"}, Positions{}},
					nodeKind: StructNodeKind,
					expected: []TokKind{TokField, TokMessage, TokRBrace},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokSemicolon, Value: ";"}, Positions{}},
					nodeKind: FieldNodeKind,
					expected: []TokKind{TokTypeRef},
				},
			},
		},
		{
			name:  "InvalidUnion",
			input: `message Data3 struct { required one @1; message Data4 union { @1 One; @2 5; Two; } }`,
			nodes: []DefNode{
				{
					Kind: StructNodeKind,
					Iden: "Data3",
					Members: []MembNode{
						{Poisoned: true, Modifier: Required, Iden: "one", Ord: 1},
					},
					LocalDefs: []DefNode{
						{
							Kind:     UnionNodeKind,
							Poisoned: true,
							Iden:     "Data4",
							Size:     16,
							Members: []MembNode{
								{Iden: "One", Ord: 1},
								{Poisoned: true, Ord: 2},
							},
						},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokSemicolon, Value: ";"}, Positions{}},
					nodeKind: FieldNodeKind,
					expected: []TokKind{TokTypeRef},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokInteger, Value: "5", Num: 5}, Positions{}},
					nodeKind: OptionNodeKind,
					expected: []TokKind{TokIden},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "Two"}, Positions{}},
					nodeKind: UnionNodeKind,
					expected: []TokKind{TokOption, TokMessage, TokRBrace},
				},
			},
		},
		{
			name:  "InvalidEnum",
			input: `message Data4 [4]enum { @1 ONE; @2 2 TWO; THREE; }`,
			nodes: []DefNode{
				{
					Kind:     EnumNodeKind,
					Poisoned: true,
					Iden:     "Data4",
					Size:     4,
					Members: []MembNode{
						{Iden: "ONE", Ord: 1},
						{Poisoned: true, Ord: 2},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokInteger, Value: "2", Num: 2}, Positions{}},
					nodeKind: CaseNodeKind,
					expected: []TokKind{TokIden},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "THREE"}, Positions{}},
					nodeKind: EnumNodeKind,
					expected: []TokKind{TokCase, TokRBrace},
				},
			},
		},
		{
			name:  "DuplicatedIdentifiers",
			input: `message Data struct { required one one one one one @1 int128; two two two two two @2 []int5; required three @3 int128 }`,
			nodes: []DefNode{
				{
					Kind:     StructNodeKind,
					Poisoned: true,
					Iden:     "Data",
					Members: []MembNode{
						{Poisoned: true, Modifier: Required, Iden: "one"},
						{Poisoned: true, Modifier: Required, Iden: "three", Ord: 3, LType: TypeNode{Iden: "int128"}},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "one"}, Positions{}},
					nodeKind: FieldNodeKind,
					expected: []TokKind{TokOrd},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokIden, Value: "two"}, Positions{}},
					nodeKind: StructNodeKind,
					expected: []TokKind{TokField, TokMessage, TokRBrace},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokRBrace, Value: "}"}, Positions{}},
					nodeKind: FieldNodeKind,
					expected: []TokKind{TokSemicolon},
				},
			},
		},
		{
			name:  "InvalidRpc",
			input: `service Data { rpc @1 Hello(Test) (Output) required one @1 int128; rpc @2 World(Test1) returns () }`,
			nodes: []DefNode{
				{
					Kind:     ServiceNodeKind,
					Poisoned: true,
					Iden:     "Data",
					Members: []MembNode{
						{Poisoned: true, Iden: "Hello", Ord: 1, LType: TypeNode{Iden: "Test"}},
						{Poisoned: true, Iden: "World", Ord: 2, LType: TypeNode{Iden: "Test1"}},
					},
				},
			},
			errs: []error{
				&ParseErr{
					actual:   Token{TokVal{Kind: TokLParen, Value: "("}, Positions{}},
					nodeKind: RpcNodeKind,
					expected: []TokKind{TokReturns},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokRequired, Value: "required"}, Positions{}},
					nodeKind: ServiceNodeKind,
					expected: []TokKind{TokRpc, TokMessage, TokRBrace},
				},
				&ParseErr{
					actual:   Token{TokVal{Kind: TokRParen, Value: ")"}, Positions{}},
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
			ClearNodeList(nodes)

			printLine := func(err string) { t.Log(err) }
			printErrors(errs, "test", printLine)
			clearErrors(errs)

			assert.Equal(t, test.nodes, nodes)
			assert.Equal(t, test.errs, errs)
		})
	}
}

func TestParser_Garbage(t *testing.T) {
	input := `hello world service struct field} lorem; ipsum 5a{ test 123 go there`
	done := make(chan struct{})

	go func() {
		var errs []error
		_ = runParser(input, &errs)
		close(done)
	}()

	select {
	case <-time.After(time.Second):
		t.Fatal("garbage parser test has timed out, is there an infinite loop?")
	case <-done:
		t.Log("finished garbage parser test")
	}
}
