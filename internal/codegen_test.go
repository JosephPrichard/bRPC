package internal

import (
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

	_ = runCodeBuilder(parseOrElse(input), "data")

	// assert.Equal(t, "", output)
	// assert.Empty(t, errs)
	t.Fail()
}

func TestCodegen_Union(t *testing.T) {
	input := `
	message Data union {
		two @2 int1;
		three @3 int2;
		one @1 int3;
		four @4 int4;
	}
	`

	_ = runCodeBuilder(parseOrElse(input), "data")

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
	_ = runCodeBuilder(parseOrElse(input), "data")

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
	_ = runCodeBuilder(parseOrElse(input), "data")

	// t.Logf("\n%s", output)
	// for _, err := range errs {
	// 	t.Logf("%v\n", err)
	// }

	// assert.Equal(t, "", output)
	// assert.Empty(t, errs)
	t.Fail()
}