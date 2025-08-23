package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCodegen_Structs(t *testing.T) {
	input := `
	package = "data"

	text Data1 struct {
		required one @1 b128;
		required two @2 []b5;
		optional three @3 [16]b4;
		optional four @4 [][4][]b4;
	}
	`

	var errs []error
	output := runCodeBuilder(input, &errs)

	expectedErrs := []error{}

	assert.Equal(t, "", output)
	assert.Equal(t, expectedErrs, errs)
}
