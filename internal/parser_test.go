package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func runParser(input string) ([]Ast, error) {
	lex := NewLexer(input)
	go lex.Run()

	p := NewParser(lex.Tokens)
	err := p.Parse()

	return p.Asts, err
}

func TestParser_Struct(t *testing.T) {
	input := `
	message Data1 struct {
		required one @1 b128; // this is the first comment
		required two @2 []b5; // this is the second comment
		optional three @3 [16]b4;
	}

	message Data2 struct {
		required one @1 Data;
	}
	`
	asts, err := runParser(input)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	expectedAsts := []Ast{}

	assert.Equal(t, expectedAsts, asts)
}
