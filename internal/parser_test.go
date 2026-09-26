package internal

import (
	"fmt"
	"math/big"
	"strings"
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

	nodes, errs := Parse(input)
	clearNodeList(nodes)

	t.Logf("\n%s\n", FmtAst(nodes))

	expectedNodes := []DefNode{
		{Kind: ImportNodeKind, StrValue: "/services/schemas/animals"},
		{Kind: PropertyNodeKind, Iden: "package", StrValue: "/hello/\\\"world\""},
		{Kind: PropertyNodeKind, Iden: "constant", StrValue: "Value"},
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
	nodes, errs := Parse(input)
	clearNodeList(nodes)

	t.Logf("\n%s\n", FmtAst(nodes))

	expectedNodes := []DefNode{
		{
			Kind: StructNodeKind,
			Iden: "Data1",
			Members: []MemberNode{
				{Modifier: Required, Iden: "one", Tag: 1, LeftType: TypeNode{Iden: "int128"}},
				{Modifier: Required, Iden: "two", Tag: 2, LeftType: TypeNode{Iden: "int5", Array: []uint64{0}}},
				{Modifier: Optional, Iden: "three", Tag: 3, LeftType: TypeNode{Iden: "int4", Array: []uint64{16}}},
				{Modifier: Optional, Iden: "four", Tag: 4, LeftType: TypeNode{Iden: "int4", Array: []uint64{0, 4, 0}}},
			},
			LocalDefs: []DefNode{
				{
					Kind: StructNodeKind,
					Iden: "Data2",
					Members: []MemberNode{
						{Modifier: Required, Iden: "one", Tag: 1, LeftType: TypeNode{Iden: "Data3"}},
					},
					LocalDefs: []DefNode{
						{
							Kind:       StructNodeKind,
							Iden:       "Data3",
							TypeParams: []string{"A", "B"},
							Members: []MemberNode{
								{Modifier: Deprecated, Iden: "one", Tag: 1, LeftType: TypeNode{Iden: "A"}},
								{Modifier: Required, Iden: "two", Tag: 2, LeftType: TypeNode{Iden: "B"}},
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

func TestParser_Struct_DefaultFields(t *testing.T) {
	input := `
	message Data struct {
		required one @1 string = "default";
		required two @2 int128 = 1;
		required three @3 float64 = 0.1;
	}
	`
	nodes, errs := Parse(input)
	clearNodeList(nodes)

	t.Logf("\n%s\n", FmtAst(nodes))

	expectedNodes := []DefNode{
		{
			Kind: StructNodeKind,
			Iden: "Data",
			Members: []MemberNode{
				{
					Modifier:     Required,
					Iden:         "one",
					Tag:          1,
					LeftType:     TypeNode{Iden: "string"},
					DefaultValue: ValueNode{Kind: StringInstanceKind, Str: "default"},
				},
				{
					Modifier:     Required,
					Iden:         "two",
					Tag:          2,
					LeftType:     TypeNode{Iden: "int128"},
					DefaultValue: ValueNode{Kind: IntInstanceKind, Int: *big.NewInt(1)},
				},
				{
					Modifier:     Required,
					Iden:         "three",
					Tag:          3,
					LeftType:     TypeNode{Iden: "float64"},
					DefaultValue: ValueNode{Kind: Float64InstanceKind, Float64: 0.1},
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
	nodes, errs := Parse(input)
	clearNodeList(nodes)

	t.Logf("\n%s\n", FmtAst(nodes))

	expectedNodes := []DefNode{
		{
			Kind: EnumNodeKind,
			Iden: "Data1",
			Size: 16,
			Members: []MemberNode{
				{Tag: 1, Iden: "One"},
				{Tag: 2, Iden: "Two"},
				{Tag: 3, Iden: "Three"},
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
	nodes, errs := Parse(input)
	clearNodeList(nodes)

	t.Logf("\n%s\n", FmtAst(nodes))

	expectedNodes := []DefNode{
		{
			Kind:       UnionNodeKind,
			Iden:       "Data",
			Size:       8,
			TypeParams: []string{"A", "B", "C"},
			Members: []MemberNode{
				{Tag: 1, Iden: "one", LeftType: TypeNode{Iden: "Data2"}},
				{Tag: 2, Iden: "two", LeftType: TypeNode{Iden: "Data1"}},
				{Tag: 3, Iden: "three", LeftType: TypeNode{Iden: "Data", Array: []uint64{0}}},
			},
			LocalDefs: []DefNode{
				{
					Kind: UnionNodeKind,
					Iden: "Data",
					Size: 16,
					Members: []MemberNode{
						{Tag: 1, Iden: "one", LeftType: TypeNode{Iden: "B"}},
						{Tag: 2, Iden: "two", LeftType: TypeNode{Iden: "C"}},
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
	nodes, errs := Parse(input)
	clearNodeList(nodes)

	t.Logf("\n%s\n", FmtAst(nodes))

	expectedNodes := []DefNode{
		{
			Kind: ServiceNodeKind,
			Iden: "ServiceA",
			Members: []MemberNode{
				{Tag: 1, Iden: "Hello", LeftType: TypeNode{Iden: "Test"}, RightType: TypeNode{Iden: "Output"}},
				{
					Tag:       2,
					Iden:      "World",
					LeftType:  TypeNode{Iden: "Test1", TypeArgs: []TypeNode{{Iden: "Arg1"}, {Iden: "Arg2"}, {Iden: "Arg3"}}},
					RightType: TypeNode{Iden: "Output1", TypeArgs: []TypeNode{{Iden: "Arg1"}, {Iden: "Arg2"}}},
				},
			},
			LocalDefs: []DefNode{
				{
					Kind: StructNodeKind,
					Iden: "Test",
					Members: []MemberNode{
						{Modifier: Required, Iden: "one", Tag: 1, LeftType: TypeNode{Iden: "b24"}},
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
		errs  []ParseError
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
					Members: []MemberNode{
						{Modifier: Required, Iden: "one", Tag: 1, LeftType: TypeNode{Iden: "int128"}},
					},
				},
			},
			errs: []ParseError{
				{
					actualToken: Token{TokVal{Kind: TokEof, Str: ""}, Positions{}},
					nodeKind:    StructNodeKind,
					expected:    []TokKind{TokField, TokMessage, TokRBrace},
					errKind:     ExpectErrKind,
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
			errs: []ParseError{
				{
					actualToken: Token{TokVal{Kind: TokEof, Str: ""}, Positions{}},
					nodeKind:    StructNodeKind,
					expected:    []TokKind{TokField, TokMessage, TokRBrace},
					errKind:     ExpectErrKind,
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
					Members: []MemberNode{
						{Modifier: Required, Iden: "one", Tag: 1, LeftType: TypeNode{Iden: "int128"}},
					},
				},
			},
			errs: []ParseError{
				{
					actualToken: Token{TokVal{Kind: TokInteger, Str: "5", Int: *big.NewInt(5)}, Positions{}},
					nodeKind:    MessageNodeKind,
					errKind:     SizeErrKind,
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
					Members: []MemberNode{
						{Poisoned: true, Modifier: Required, Iden: "one", Tag: 1},
					},
				},
				{
					Kind: StructNodeKind,
					Iden: "Data_1",
					Members: []MemberNode{
						{Modifier: Required, Iden: "one", Tag: 1, LeftType: TypeNode{Iden: "int128"}},
					},
				},
			},
			errs: []ParseError{
				{
					actualToken: Token{TokVal{Kind: TokErr, Str: "5a", Expected: TokInteger}, Positions{}},
					nodeKind:    TypeNodeKind,
					expected:    []TokKind{TokInteger, TokRBrack},
					errKind:     ExpectErrKind,
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
					Members: []MemberNode{
						{Poisoned: true, Modifier: Required, Iden: "one"},
					},
					LocalDefs: []DefNode{
						{
							Kind: StructNodeKind,
							Iden: "Data2",
							Members: []MemberNode{
								{Poisoned: true, Modifier: Required, Iden: "one", Tag: 1},
							},
						},
					},
				},
			},
			errs: []ParseError{
				{
					actualToken: Token{TokVal{Kind: TokErr, Str: "@1abc", Expected: TokTag}, Positions{}},
					nodeKind:    FieldNodeKind,
					expected:    []TokKind{TokTag},
					errKind:     ExpectErrKind,
				},
				{
					actualToken: Token{TokVal{Kind: TokIden, Str: "two"}, Positions{}},
					nodeKind:    StructNodeKind,
					expected:    []TokKind{TokField, TokMessage, TokRBrace},
					errKind:     ExpectErrKind,
				},
				{
					actualToken: Token{TokVal{Kind: TokSemicolon, Str: ";"}, Positions{}},
					nodeKind:    FieldNodeKind,
					expected:    []TokKind{TokTypeRef},
					errKind:     ExpectErrKind,
				},
			},
		},
		{
			name: "InvalidUnion",
			input: `
			message Data3 struct { 
				required one @1; 
				message Data4 union { 
					one @1 One; 
					two @2 5; 
					three Two; 
				}
			}`,
			nodes: []DefNode{
				{
					Kind: StructNodeKind,
					Iden: "Data3",
					Members: []MemberNode{
						{Poisoned: true, Modifier: Required, Iden: "one", Tag: 1},
					},
					LocalDefs: []DefNode{
						{
							Kind:     UnionNodeKind,
							Poisoned: false,
							Iden:     "Data4",
							Size:     16,
							Members: []MemberNode{
								{Iden: "one", Tag: 1, LeftType: TypeNode{Iden: "One"}},
								{Poisoned: true, Iden: "two", Tag: 2},
								{Poisoned: true, Iden: "three"},
							},
						},
					},
				},
			},
			errs: []ParseError{
				{
					actualToken: Token{TokVal{Kind: TokSemicolon, Str: ";"}, Positions{}},
					nodeKind:    FieldNodeKind,
					expected:    []TokKind{TokTypeRef},
					errKind:     ExpectErrKind,
				},
				{
					actualToken: Token{TokVal{Kind: TokInteger, Str: "5", Int: *big.NewInt(5)}, Positions{}},
					nodeKind:    OptionNodeKind,
					expected:    []TokKind{TokTypeRef},
					errKind:     ExpectErrKind,
				},
				{
					actualToken: Token{TokVal{Kind: TokIden, Str: "Two"}, Positions{}},
					nodeKind:    OptionNodeKind,
					expected:    []TokKind{TokTag},
					errKind:     ExpectErrKind,
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
					Members: []MemberNode{
						{Iden: "ONE", Tag: 1},
						{Poisoned: true, Tag: 2},
					},
				},
			},
			errs: []ParseError{
				{
					actualToken: Token{TokVal{Kind: TokInteger, Str: "2", Int: *big.NewInt(2)}, Positions{}},
					nodeKind:    CaseNodeKind,
					expected:    []TokKind{TokIden},
					errKind:     ExpectErrKind,
				},
				{
					actualToken: Token{TokVal{Kind: TokIden, Str: "THREE"}, Positions{}},
					nodeKind:    EnumNodeKind,
					expected:    []TokKind{TokCase, TokRBrace},
					errKind:     ExpectErrKind,
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
					Members: []MemberNode{
						{Poisoned: true, Modifier: Required, Iden: "one"},
						{Poisoned: true, Modifier: Required, Iden: "three", Tag: 3, LeftType: TypeNode{Iden: "int128"}},
					},
				},
			},
			errs: []ParseError{
				{
					actualToken: Token{TokVal{Kind: TokIden, Str: "one"}, Positions{}},
					nodeKind:    FieldNodeKind,
					expected:    []TokKind{TokTag},
					errKind:     ExpectErrKind,
				},
				{
					actualToken: Token{TokVal{Kind: TokIden, Str: "two"}, Positions{}},
					nodeKind:    StructNodeKind,
					expected:    []TokKind{TokField, TokMessage, TokRBrace},
					errKind:     ExpectErrKind,
				},
				{
					actualToken: Token{TokVal{Kind: TokRBrace, Str: "}"}, Positions{}},
					nodeKind:    FieldNodeKind,
					expected:    []TokKind{TokEqual, TokSemicolon},
					errKind:     ExpectErrKind,
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
					Members: []MemberNode{
						{Poisoned: true, Iden: "Hello", Tag: 1, LeftType: TypeNode{Iden: "Test"}},
						{Poisoned: true, Iden: "World", Tag: 2, LeftType: TypeNode{Iden: "Test1"}},
					},
				},
			},
			errs: []ParseError{
				{
					actualToken: Token{TokVal{Kind: TokLParen, Str: "("}, Positions{}},
					nodeKind:    RpcNodeKind,
					expected:    []TokKind{TokReturns},
					errKind:     ExpectErrKind,
				},
				{
					actualToken: Token{TokVal{Kind: TokRequired, Str: "required"}, Positions{}},
					nodeKind:    ServiceNodeKind,
					expected:    []TokKind{TokRpc, TokMessage, TokRBrace},
					errKind:     ExpectErrKind,
				},
				{
					actualToken: Token{TokVal{Kind: TokRParen, Str: ")"}, Positions{}},
					nodeKind:    RpcNodeKind,
					expected:    []TokKind{TokTypeRef},
					errKind:     ExpectErrKind,
				},
			},
		},
		{
			name:  "InvalidDefaultFields",
			input: `message Data struct { required one @1 string "default"; required two @2 string = invalid; }`,
			nodes: []DefNode{
				{
					Kind: StructNodeKind,
					Iden: "Data",
					Members: []MemberNode{
						{Poisoned: true, Modifier: Required, Iden: "one", Tag: 1, LeftType: TypeNode{Iden: "string"}},
						{
							Poisoned:     false,
							Modifier:     Required,
							Iden:         "two",
							Tag:          2,
							LeftType:     TypeNode{Iden: "string"},
							DefaultValue: ValueNode{Poisoned: true},
						},
					},
				},
			},
			errs: []ParseError{
				{
					actualToken: Token{TokVal{Kind: TokString, Str: "\"default\""}, Positions{}},
					nodeKind:    FieldNodeKind,
					expected:    []TokKind{TokEqual, TokSemicolon},
					errKind:     ExpectErrKind,
				},
				{
					actualToken: Token{TokVal{Kind: TokIden, Str: "invalid"}, Positions{}},
					nodeKind:    ValueNodeKind,
					expected:    []TokKind{TokInteger, TokFloat, TokString},
					errKind:     ExpectErrKind,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("test/%s", test.name), func(t *testing.T) {
			nodes, errs := Parse(test.input)
			clearNodeList(nodes)

			printLine := func(err string) { t.Log(err) }
			PrintErrors(errs, "test", printLine)
			clearParseErrors(errs)

			assert.Equal(t, test.nodes, nodes)
			assert.Equal(t, test.errs, errs)
		})
	}
}

func TestParser_Garbage(t *testing.T) {
	// note(Joseph): This is not a replacement for a fuzzer, just a sanity check
	input := `hello world service struct field} lorem; ipsum 5a{ test 123 go there`
	done := make(chan struct{})

	go func() {
		_, _ = Parse(input)
		close(done)
	}()

	select {
	case <-time.After(time.Second):
		t.Fatal("garbage parser test has timed out, is there an infinite loop?")
	case <-done:
		t.Log("finished garbage parser test")
	}
}

func Benchmark_ParseLargeAst(b *testing.B) {
	astString, err := loadSampleFile()
	if err != nil {
		b.Fatal(err.Error())
	}

	for b.Loop() {
		b.StopTimer()
		// fmt.Printf("%v\n\n", FmtAstWithConfig(randomNodes, &FmtAstConfig{ShouldPrintLines: true}))

		startTime := time.Now()
		b.StartTimer()
		resultNodes := MustParse(astString)

		b.StopTimer()
		endTime := time.Now()

		lineCount := strings.Count(astString, "\n") + 1
		totalTime := endTime.Sub(startTime)

		fmt.Printf("AstLineCount: %d\nDuration: %v\nLinesPerSecond: %f\n\n",
			lineCount,
			totalTime,
			linesPerUnit(lineCount, totalTime.Seconds()),
		)

		assert.Equal(b, astString, FmtAst(resultNodes))
		b.StartTimer()
	}
}

func Fuzz_Parser(f *testing.F) {
	astGenConfig := &AstGenerationConfig{
		maxDefNodes:    100,
		maxMemberNodes: 25,
		maxStrLength:   25,
		maxDepth:       3,
		maxArrayDim:    3,
		arrayChance:    10,
	}

	var testCases []string

	testCount := 10
	for range testCount {
		ast := generateAstWithConfig(astGenConfig)
		testCases = append(testCases, FmtAst(ast))
	}

	// fmt.Printf("Fuzz_Parser: %+v test cases generated\n", testCases)

	for _, tc := range testCases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, spec string) {
		_, _ = Parse(spec)
	})
}
