package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func runLexer(input string) []TokVal {
	lex := newLexer(input)
	go lex.run()

	var tokens []TokVal
	for token := range lex.tokens {
		tokens = append(tokens, token.TokVal)
	}

	return tokens
}

func TestLexer_Properties(t *testing.T) {
	input := `
	package = "/hello/\\\"world\"";
	constant = "value";
	`

	tokens := runLexer(input)

	expTokens := []TokVal{
		{t: TokIden, value: "package"},
		{t: TokEqual, value: "="},
		{t: TokString, value: "\"/hello/\\\\\\\"world\\\"\""},
		{t: TokTerminal, value: ";"},
		{t: TokIden, value: "constant"},
		{t: TokEqual, value: "="},
		{t: TokString, value: "\"value\""},
		{t: TokTerminal, value: ";"},
		{t: TokEof},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_Struct(t *testing.T) {
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
	tokens := runLexer(input)

	expTokens := []TokVal{
		{t: TokMessage, value: "message"},
		{t: TokIden, value: "Data1"},
		{t: TokStruct, value: "struct"},
		{t: TokLBrace, value: "{"},
		{t: TokRequired, value: "required"},
		{t: TokIden, value: "one"},
		{t: TokOrd, value: "@1"},
		{t: TokIden, value: "b128"},
		{t: TokTerminal, value: ";"},
		{t: TokRequired, value: "required"},
		{t: TokIden, value: "two"},
		{t: TokOrd, value: "@2"},
		{t: TokLBrack, value: "["},
		{t: TokRBrack, value: "]"},
		{t: TokIden, value: "b5"},
		{t: TokTerminal, value: ";"},
		{t: TokOptional, value: "optional"},
		{t: TokIden, value: "three"},
		{t: TokOrd, value: "@3"},
		{t: TokLBrack, value: "["},
		{t: TokInteger, value: "16"},
		{t: TokRBrack, value: "]"},
		{t: TokIden, value: "b4"},
		{t: TokTerminal, value: ";"},
		{t: TokRBrace, value: "}"},
		{t: TokMessage, value: "message"},
		{t: TokIden, value: "Data2"},
		{t: TokStruct, value: "struct"},
		{t: TokLBrace, value: "{"},
		{t: TokRequired, value: "required"},
		{t: TokIden, value: "one"},
		{t: TokOrd, value: "@1"},
		{t: TokIden, value: "Data"},
		{t: TokTerminal, value: ";"},
		{t: TokRBrace, value: "}"},
		{t: TokEof},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_Union(t *testing.T) {
	input := `
	message Data3 enum {
		@1 One;
		@2 Two;
		@3 Three;
	}
	
	message Data4 union = Data1 | Data2;
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{t: TokMessage, value: "message"},
		{t: TokIden, value: "Data3"},
		{t: TokEnum, value: "enum"},
		{t: TokLBrace, value: "{"},
		{t: TokOrd, value: "@1"},
		{t: TokIden, value: "One"},
		{t: TokTerminal, value: ";"},
		{t: TokOrd, value: "@2"},
		{t: TokIden, value: "Two"},
		{t: TokTerminal, value: ";"},
		{t: TokOrd, value: "@3"},
		{t: TokIden, value: "Three"},
		{t: TokTerminal, value: ";"},
		{t: TokRBrace, value: "}"},
		{t: TokMessage, value: "message"},
		{t: TokIden, value: "Data4"},
		{t: TokUnion, value: "union"},
		{t: TokEqual, value: "="},
		{t: TokIden, value: "Data1"},
		{t: TokPipe, value: "|"},
		{t: TokIden, value: "Data2"},
		{t: TokTerminal, value: ";"},
		{t: TokEof},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_Service(t *testing.T) {
	input := `
	service ThingService {
    	rpc @1 DoThis (input) returns (Output)
		rpc @2 DoThat (In) returns (Out)
	}
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{t: TokService, value: "service"},
		{t: TokIden, value: "ThingService"},
		{t: TokLBrace, value: "{"},
		{t: TokRpc, value: "rpc"},
		{t: TokOrd, value: "@1"},
		{t: TokIden, value: "DoThis"},
		{t: TokLParen, value: "("},
		{t: TokIden, value: "input"},
		{t: TokRParen, value: ")"},
		{t: TokReturns, value: "returns"},
		{t: TokLParen, value: "("},
		{t: TokIden, value: "Output"},
		{t: TokRParen, value: ")"},
		{t: TokRpc, value: "rpc"},
		{t: TokOrd, value: "@2"},
		{t: TokIden, value: "DoThat"},
		{t: TokLParen, value: "("},
		{t: TokIden, value: "In"},
		{t: TokRParen, value: ")"},
		{t: TokReturns, value: "returns"},
		{t: TokLParen, value: "("},
		{t: TokIden, value: "Out"},
		{t: TokRParen, value: ")"},
		{t: TokRBrace, value: "}"},
		{t: TokEof},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_Empty(t *testing.T) {
	input := ""
	tokens := runLexer(input)
	assert.Nil(t, tokens)
}

func TestLexer_BadOrd(t *testing.T) {
	input := `
	message Data3 struct {
		required one @1abc b128;
	}
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{t: TokMessage, value: "message"},
		{t: TokIden, value: "Data3"},
		{t: TokStruct, value: "struct"},
		{t: TokLBrace, value: "{"},
		{t: TokRequired, value: "required"},
		{t: TokIden, value: "one"},
		{t: TokErr, value: "@1abc"},
		{t: TokIden, value: "b128"},
		{t: TokTerminal, value: ";"},
		{t: TokRBrace, value: "}"},
		{t: TokEof},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_BadComment(t *testing.T) {
	input := `
	message Data2 struct {
		required one @1 Data; /# bad
	}
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{t: TokMessage, value: "message"},
		{t: TokIden, value: "Data2"},
		{t: TokStruct, value: "struct"},
		{t: TokLBrace, value: "{"},
		{t: TokRequired, value: "required"},
		{t: TokIden, value: "one"},
		{t: TokOrd, value: "@1"},
		{t: TokIden, value: "Data"},
		{t: TokTerminal, value: ";"},
		{t: TokErr, value: "/#"},
		{t: TokIden, value: "bad"},
		{t: TokRBrace, value: "}"},
		{t: TokEof},
	}

	assert.Equal(t, expTokens, tokens)
}
