package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func runLexer(input string) []TokVal {
	lex := makeLexer(input)
	lex.run()

	var tokens []TokVal
	for _, token := range lex.tokens {
		tokens = append(tokens, token.TokVal)
	}
	return tokens
}

func TestLexer_Properties(t *testing.T) {
	input := `
	import "/path/to/idl/idl.brpc"

	package = "/hello/\\\"world\""
	constant = "typValue"
	`

	tokens := runLexer(input)

	expTokens := []TokVal{
		{Kind: TokImport, Value: "import"},
		{Kind: TokString, Value: "\"/path/to/idl/idl.brpc\""},
		{Kind: TokIden, Value: "package"},
		{Kind: TokEqual, Value: "="},
		{Kind: TokString, Value: "\"/hello/\\\\\\\"world\\\"\""},
		{Kind: TokIden, Value: "constant"},
		{Kind: TokEqual, Value: "="},
		{Kind: TokString, Value: "\"typValue\""},
		{Kind: TokEof},
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

	message Data2 struct[A, B] {
		deprecated one @1 A;
        required two @2 B;
	}
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{Kind: TokMessage, Value: "message"},
		{Kind: TokIden, Value: "Data1"},
		{Kind: TokStruct, Value: "struct"},
		{Kind: TokLBrace, Value: "{"},
		{Kind: TokRequired, Value: "required"},
		{Kind: TokIden, Value: "one"},
		{Kind: TokOrd, Value: "@1"},
		{Kind: TokIden, Value: "b128"},
		{Kind: TokSemicolon, Value: ";"},
		{Kind: TokRequired, Value: "required"},
		{Kind: TokIden, Value: "two"},
		{Kind: TokOrd, Value: "@2"},
		{Kind: TokLBrack, Value: "["},
		{Kind: TokRBrack, Value: "]"},
		{Kind: TokIden, Value: "b5"},
		{Kind: TokSemicolon, Value: ";"},
		{Kind: TokOptional, Value: "optional"},
		{Kind: TokIden, Value: "three"},
		{Kind: TokOrd, Value: "@3"},
		{Kind: TokLBrack, Value: "["},
		{Kind: TokInteger, Value: "16"},
		{Kind: TokRBrack, Value: "]"},
		{Kind: TokIden, Value: "b4"},
		{Kind: TokSemicolon, Value: ";"},
		{Kind: TokRBrace, Value: "}"},
		{Kind: TokMessage, Value: "message"},
		{Kind: TokIden, Value: "Data2"},
		{Kind: TokStruct, Value: "struct"},
		{Kind: TokLBrack, Value: "["},
		{Kind: TokIden, Value: "A"},
		{Kind: TokComma, Value: ","},
		{Kind: TokIden, Value: "B"},
		{Kind: TokRBrack, Value: "]"},
		{Kind: TokLBrace, Value: "{"},
		{Kind: TokDeprecated, Value: "deprecated"},
		{Kind: TokIden, Value: "one"},
		{Kind: TokOrd, Value: "@1"},
		{Kind: TokIden, Value: "A"},
		{Kind: TokSemicolon, Value: ";"},
		{Kind: TokRequired, Value: "required"},
		{Kind: TokIden, Value: "two"},
		{Kind: TokOrd, Value: "@2"},
		{Kind: TokIden, Value: "B"},
		{Kind: TokSemicolon, Value: ";"},
		{Kind: TokRBrace, Value: "}"},
		{Kind: TokEof},
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
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{Kind: TokMessage, Value: "message"},
		{Kind: TokIden, Value: "Data3"},
		{Kind: TokEnum, Value: "enum"},
		{Kind: TokLBrace, Value: "{"},
		{Kind: TokOrd, Value: "@1"},
		{Kind: TokIden, Value: "One"},
		{Kind: TokSemicolon, Value: ";"},
		{Kind: TokOrd, Value: "@2"},
		{Kind: TokIden, Value: "Two"},
		{Kind: TokSemicolon, Value: ";"},
		{Kind: TokOrd, Value: "@3"},
		{Kind: TokIden, Value: "Three"},
		{Kind: TokSemicolon, Value: ";"},
		{Kind: TokRBrace, Value: "}"},
		{Kind: TokEof},
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
		{Kind: TokService, Value: "service"},
		{Kind: TokIden, Value: "ThingService"},
		{Kind: TokLBrace, Value: "{"},
		{Kind: TokRpc, Value: "rpc"},
		{Kind: TokOrd, Value: "@1"},
		{Kind: TokIden, Value: "DoThis"},
		{Kind: TokLParen, Value: "("},
		{Kind: TokIden, Value: "input"},
		{Kind: TokRParen, Value: ")"},
		{Kind: TokReturns, Value: "returns"},
		{Kind: TokLParen, Value: "("},
		{Kind: TokIden, Value: "Output"},
		{Kind: TokRParen, Value: ")"},
		{Kind: TokRpc, Value: "rpc"},
		{Kind: TokOrd, Value: "@2"},
		{Kind: TokIden, Value: "DoThat"},
		{Kind: TokLParen, Value: "("},
		{Kind: TokIden, Value: "In"},
		{Kind: TokRParen, Value: ")"},
		{Kind: TokReturns, Value: "returns"},
		{Kind: TokLParen, Value: "("},
		{Kind: TokIden, Value: "Out"},
		{Kind: TokRParen, Value: ")"},
		{Kind: TokRBrace, Value: "}"},
		{Kind: TokEof},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_Empty(t *testing.T) {
	input := ""
	tokens := runLexer(input)
	assert.Equal(t, []TokVal{{Kind: TokEof}}, tokens)
}

func TestLexer_BadOrd(t *testing.T) {
	input := `
	message Data3 struct {
		required one @1abc b128;
	}
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{Kind: TokMessage, Value: "message"},
		{Kind: TokIden, Value: "Data3"},
		{Kind: TokStruct, Value: "struct"},
		{Kind: TokLBrace, Value: "{"},
		{Kind: TokRequired, Value: "required"},
		{Kind: TokIden, Value: "one"},
		{Kind: TokErr, Value: "@1abc", Expected: TokOrd},
		{Kind: TokIden, Value: "b128"},
		{Kind: TokSemicolon, Value: ";"},
		{Kind: TokRBrace, Value: "}"},
		{Kind: TokEof},
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
		{Kind: TokMessage, Value: "message"},
		{Kind: TokIden, Value: "Data2"},
		{Kind: TokStruct, Value: "struct"},
		{Kind: TokLBrace, Value: "{"},
		{Kind: TokRequired, Value: "required"},
		{Kind: TokIden, Value: "one"},
		{Kind: TokOrd, Value: "@1"},
		{Kind: TokIden, Value: "Data"},
		{Kind: TokSemicolon, Value: ";"},
		{Kind: TokErr, Value: "/# bad", Expected: TokComment},
		{Kind: TokRBrace, Value: "}"},
		{Kind: TokEof},
	}

	assert.Equal(t, expTokens, tokens)
}

func TestLexer_BadInteger(t *testing.T) {
	input := `
	message Data2 struct {
		required one @1 [5a]Data;
	}
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{Kind: TokMessage, Value: "message"},
		{Kind: TokIden, Value: "Data2"},
		{Kind: TokStruct, Value: "struct"},
		{Kind: TokLBrace, Value: "{"},
		{Kind: TokRequired, Value: "required"},
		{Kind: TokIden, Value: "one"},
		{Kind: TokOrd, Value: "@1"},
		{Kind: TokLBrack, Value: "["},
		{Kind: TokErr, Value: "5a", Expected: TokInteger},
		{Kind: TokRBrack, Value: "]"},
		{Kind: TokIden, Value: "Data"},
		{Kind: TokSemicolon, Value: ";"},
		{Kind: TokRBrace, Value: "}"},
		{Kind: TokEof},
	}

	assert.Equal(t, expTokens, tokens)
}
