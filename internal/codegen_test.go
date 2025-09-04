package internal

import (
	"fmt"
	"testing"
	// "github.com/stretchr/testify/assert"
)

func TestCodegen_Structs(t *testing.T) {
	input := `
	message Data1 struct {
		required one @1 int128;
	}
	message Data struct {
		required one @1 Data1;
		required two @2 string;
		optional three @3 [16]int9;
		optional four @4 [][4][]int4;
	}
	`

	var errs []error
	output := runCodeBuilder(input, "data", &errs)

	t.Logf("\n%s", output)
	for _, err := range errs {
		t.Logf("%v\n", err)
	}

	// assert.Equal(t, "", output)
	// assert.Empty(t, errs)
	t.Fail()
}

func TestCodegen_Union(t *testing.T) {
	input := `
	message Data union {
		two @2 B;
		three @3 C;
		one @1 A;
		four @4 D;
	}
	`

	var errs []error
	output := runCodeBuilder(input, "data", &errs)

	t.Logf("\n%s", output)
	for _, err := range errs {
		t.Logf("%v\n", err)
	}

	// assert.Equal(t, "", output)
	// assert.Empty(t, errs)
	t.Fail()
}

func TestCodegen_Enum(t *testing.T) {
	input := `
	message Data enum {
		@1 One;
		@2 Two;
		@3 Three;
	}
	`

	var errs []error
	output := runCodeBuilder(input, "data", &errs)

	t.Logf("\n%s", output)
	for _, err := range errs {
		t.Logf("%v\n", err)
	}

	// assert.Equal(t, "", output)
	// assert.Empty(t, errs)
	t.Fail()
}

func TestCodegen_Service(t *testing.T) {
	input := `
	service Data {
		rpc @1 Do(Input) returns (Output)
	}
	`

	var errs []error
	output := runCodeBuilder(input, "data", &errs)

	t.Logf("\n%s", output)
	for _, err := range errs {
		t.Logf("%v\n", err)
	}

	// assert.Equal(t, "", output)
	// assert.Empty(t, errs)
	t.Fail()
}

func TestCodegen_Errors(t *testing.T) {
	type Test struct {
		name  string
		input string
		errs  []error
	}

	tests := []Test{
		// {
		// 	name: "DuplicateTypeIden",
		// 	input: `
		// 	message Data struct {}
		// 	message Data struct {
		// 		message Data1 struct {
		// 			message Data1 struct {}
		// 		}
		// 		message Data1 struct {}
		// 	}
		// 	`,
		// 	errs: []error{},
		// },
		// {
		// 	name: "DuplicateFieldIdens",
		// 	input: `
		// 	message Data1 struct {
		// 		required one @1 int16;
		// 		deprecated one @2 int16;
		// 	}
		// 	message Data2 enum {
		// 		@1 One;
		// 		@2 One;
		// 	}
		// 	message Data3 union {
		// 		@1 Data1;
		// 		@2 Data2;
		// 	}
		// 	`,
		// 	errs: []error{},
		// },
		// {
		// 	name: "InvalidOrds",
		// 	input: `
		// 	message Data1 struct {
		// 		required one @1 int16;
		// 		deprecated two @2 int16;
		// 		deprecated one @3 int16;
		// 	}
		// 	message Data2 enum {
		// 		@1 One;
		// 		@1 One;
		// 	}
		// 	message Data3 union {
		// 		@1 Data1;
		// 		@1 Data2;
		// 	}
		// 	message Data4 union {
		// 		@0 Data1;
		// 		@4 Data2;
		// 	}
		// 	`,
		// 	errs: []error{},
		// },
		{
			name: "UnresolvedIden",
			input: `
			message Data1 struct {
				required one @1 int16;
				deprecated two @2 int16;
				deprecated one @3 int16;

				message Data3 union {
					@1 Data2;
					@2 Invalid;
				}
			}
			message Data2 struct {
				required one @1 Data1;
				required two @2 Invalid;
			}
			`,
			errs: []error{},
		},
		{
			name: "RecursiveAst",
			input: `
			message Data1 struct {
				required one @1 int16;
				deprecated two @2 Data1;
				deprecated one @3 int16;

				message Data4 union {
					@1 Data1;
					@2 Data4;
				}
			}

			message Data2 struct {
				required one @1 Data3;

				message Data3 struct {
					required one @1 Data2;
				}
			}
			`,
			errs: []error{},
		},
		// {
		// 	name: "InvalidTypeArgs",
		// 	input: `
		// 	message Data1 struct {
		// 		required one @1 Data2;
		// 		required one @2 Data2(int8);
		// 		required one @3 Data2(int16, int18);
		// 	}

		// 	message Data2 union(A) {
		// 		@1 A;
		// 		@2 B;
		// 	}
		// 	`,
		// 	errs: []error{},
		// },
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("test/%s", test.name), func(t *testing.T) {
			var errs []error
			output := runCodeBuilder(test.input, "data", &errs)

			t.Logf("\n%s", output)
			for _, err := range errs {
				t.Logf("%v\n", err)
			}

			// assert.Equal(t, "", output)
			// assert.Equal(t, test.errs, errs)
			t.Fail()
		})
	}
}
