package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParser_Properties(t *testing.T) {
	input := `
	package = "/hello/\\\"world\"";
	constant = "value";
	`
	asts, errs := runParser(input)

	expectedAsts := []Ast{}

	assert.Equal(t, expectedAsts, asts)
	assert.Nil(t, errs)
}

func TestParser_Struct(t *testing.T) {
	input := `
	message Data1 struct {
		required one @1 b128; // this is the first comment
		required two @2 []b5; // this is the second comment
		optional three @3 [16]b4;
		optional four @4 [][4][]b4;
		required five @5 struct{ 
			required one @1 b16; 
			required two @2 []enum{ ONE = 1; TWO = 2; THREE = 3; };	
        };
	}

	message Data2 struct {
		required one @1 Data3;

        message Data3 struct {
			required one @1 b24;
		}
	}
	`
	asts, errs := runParser(input)

	expectedAsts := []Ast{}

	assert.Equal(t, expectedAsts, asts)
	assert.Nil(t, errs)
}

func TestParser_Enum(t *testing.T) {
	input := `
	message Data1 enum {
		@1 One;
		@2 Two;
		@3 Three;
	}
	`
	asts, errs := runParser(input)

	expectedAsts := []Ast{}

	assert.Equal(t, expectedAsts, asts)
	assert.Nil(t, errs)
}

func TestParser_Union(t *testing.T) {
	input := `
	message Data1 union {
		@1 struct{};
		@2 struct{ required one @1 b16; };
		@3 Data1;

		message Data1 struct { 
			required one @1 b2; 
        }
	}
	`
	asts, errs := runParser(input)

	expectedAsts := []Ast{}

	assert.Equal(t, expectedAsts, asts)
	assert.Nil(t, errs)
}

func TestParser_Service(t *testing.T) {
	input := `
	service ServiceA {
		rpc @1 Hello(Input) returns (Output)
		rpc @2 World(struct{}) returns (enum{ One = 1; Two = 2; Three = 3; })

		message Input struct {
			required one @1 b24;
		}
	}
	`
	asts, errs := runParser(input)

	expectedAsts := []Ast{}

	assert.Equal(t, expectedAsts, asts)
	assert.Nil(t, errs)
}
