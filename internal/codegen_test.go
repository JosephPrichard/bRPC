package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCodegen_Structs(t *testing.T) {
	input := `
	message Data struct {
		required one @1 b128;
		required two @2 []b5;
		optional three @3 [16]b4;
		optional four @4 [][4][]b4;
	}
	`

	var errs []error
	output := runCodeBuilder(input, "data", &errs)

	t.Logf("\n%s", output)

	assert.Equal(t, "", output)
	assert.Empty(t, errs)
}

func TestCodegen_Union(t *testing.T) {
	input := `
	message Data union {
		@2 B;
		@3 C;
		@1 A;
		@4 D;
	}
	`

	var errs []error
	output := runCodeBuilder(input, "data", &errs)

	t.Logf("\n%s", output)

	assert.Equal(t, "", output)
	assert.Empty(t, errs)
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

	assert.Equal(t, "", output)
	assert.Empty(t, errs)
}
