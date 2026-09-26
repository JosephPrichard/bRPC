package internal

import (
	"math/big"
	"testing"

	// "time"

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
		{Kind: TokImport, Str: "import"},
		{Kind: TokString, Str: "\"/path/to/idl/idl.brpc\""},
		{Kind: TokIden, Str: "package"},
		{Kind: TokEqual, Str: "="},
		{Kind: TokString, Str: "\"/hello/\\\\\\\"world\\\"\""},
		{Kind: TokIden, Str: "constant"},
		{Kind: TokEqual, Str: "="},
		{Kind: TokString, Str: "\"typValue\""},
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

	message Data2 struct(A, B) {
		deprecated one @1 A;
        required two @2 B;
	}
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{Kind: TokMessage, Str: "message"},
		{Kind: TokIden, Str: "Data1"},
		{Kind: TokStruct, Str: "struct"},
		{Kind: TokLBrace, Str: "{"},
		{Kind: TokRequired, Str: "required"},
		{Kind: TokIden, Str: "one"},
		{Kind: TokTag, Str: "@1", Int: *big.NewInt(1)},
		{Kind: TokIden, Str: "b128"},
		{Kind: TokSemicolon, Str: ";"},
		{Kind: TokRequired, Str: "required"},
		{Kind: TokIden, Str: "two"},
		{Kind: TokTag, Str: "@2", Int: *big.NewInt(2)},
		{Kind: TokLBrack, Str: "["},
		{Kind: TokRBrack, Str: "]"},
		{Kind: TokIden, Str: "b5"},
		{Kind: TokSemicolon, Str: ";"},
		{Kind: TokOptional, Str: "optional"},
		{Kind: TokIden, Str: "three"},
		{Kind: TokTag, Str: "@3", Int: *big.NewInt(3)},
		{Kind: TokLBrack, Str: "["},
		{Kind: TokInteger, Str: "16", Int: *big.NewInt(16)},
		{Kind: TokRBrack, Str: "]"},
		{Kind: TokIden, Str: "b4"},
		{Kind: TokSemicolon, Str: ";"},
		{Kind: TokRBrace, Str: "}"},
		{Kind: TokMessage, Str: "message"},
		{Kind: TokIden, Str: "Data2"},
		{Kind: TokStruct, Str: "struct"},
		{Kind: TokLParen, Str: "("},
		{Kind: TokIden, Str: "A"},
		{Kind: TokComma, Str: ","},
		{Kind: TokIden, Str: "B"},
		{Kind: TokRParen, Str: ")"},
		{Kind: TokLBrace, Str: "{"},
		{Kind: TokDeprecated, Str: "deprecated"},
		{Kind: TokIden, Str: "one"},
		{Kind: TokTag, Str: "@1", Int: *big.NewInt(1)},
		{Kind: TokIden, Str: "A"},
		{Kind: TokSemicolon, Str: ";"},
		{Kind: TokRequired, Str: "required"},
		{Kind: TokIden, Str: "two"},
		{Kind: TokTag, Str: "@2", Int: *big.NewInt(2)},
		{Kind: TokIden, Str: "B"},
		{Kind: TokSemicolon, Str: ";"},
		{Kind: TokRBrace, Str: "}"},
		{Kind: TokEof},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_Union(t *testing.T) {
	input := `
	message Data3 enum {
		one @1 One;
		two @2 Two;
		three @3 Three;
	}
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{Kind: TokMessage, Str: "message"},
		{Kind: TokIden, Str: "Data3"},
		{Kind: TokEnum, Str: "enum"},
		{Kind: TokLBrace, Str: "{"},
		{Kind: TokIden, Str: "one"},
		{Kind: TokTag, Str: "@1", Int: *big.NewInt(1)},
		{Kind: TokIden, Str: "One"},
		{Kind: TokSemicolon, Str: ";"},
		{Kind: TokIden, Str: "two"},
		{Kind: TokTag, Str: "@2", Int: *big.NewInt(2)},
		{Kind: TokIden, Str: "Two"},
		{Kind: TokSemicolon, Str: ";"},
		{Kind: TokIden, Str: "three"},
		{Kind: TokTag, Str: "@3", Int: *big.NewInt(3)},
		{Kind: TokIden, Str: "Three"},
		{Kind: TokSemicolon, Str: ";"},
		{Kind: TokRBrace, Str: "}"},
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
		{Kind: TokService, Str: "service"},
		{Kind: TokIden, Str: "ThingService"},
		{Kind: TokLBrace, Str: "{"},
		{Kind: TokRpc, Str: "rpc"},
		{Kind: TokTag, Str: "@1", Int: *big.NewInt(1)},
		{Kind: TokIden, Str: "DoThis"},
		{Kind: TokLParen, Str: "("},
		{Kind: TokIden, Str: "input"},
		{Kind: TokRParen, Str: ")"},
		{Kind: TokReturns, Str: "returns"},
		{Kind: TokLParen, Str: "("},
		{Kind: TokIden, Str: "Output"},
		{Kind: TokRParen, Str: ")"},
		{Kind: TokRpc, Str: "rpc"},
		{Kind: TokTag, Str: "@2", Int: *big.NewInt(2)},
		{Kind: TokIden, Str: "DoThat"},
		{Kind: TokLParen, Str: "("},
		{Kind: TokIden, Str: "In"},
		{Kind: TokRParen, Str: ")"},
		{Kind: TokReturns, Str: "returns"},
		{Kind: TokLParen, Str: "("},
		{Kind: TokIden, Str: "Out"},
		{Kind: TokRParen, Str: ")"},
		{Kind: TokRBrace, Str: "}"},
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
		{Kind: TokMessage, Str: "message"},
		{Kind: TokIden, Str: "Data3"},
		{Kind: TokStruct, Str: "struct"},
		{Kind: TokLBrace, Str: "{"},
		{Kind: TokRequired, Str: "required"},
		{Kind: TokIden, Str: "one"},
		{Kind: TokErr, Str: "@1abc", Expected: TokTag},
		{Kind: TokIden, Str: "b128"},
		{Kind: TokSemicolon, Str: ";"},
		{Kind: TokRBrace, Str: "}"},
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
		{Kind: TokMessage, Str: "message"},
		{Kind: TokIden, Str: "Data2"},
		{Kind: TokStruct, Str: "struct"},
		{Kind: TokLBrace, Str: "{"},
		{Kind: TokRequired, Str: "required"},
		{Kind: TokIden, Str: "one"},
		{Kind: TokTag, Str: "@1", Int: *big.NewInt(1)},
		{Kind: TokIden, Str: "Data"},
		{Kind: TokSemicolon, Str: ";"},
		{Kind: TokErr, Str: "/# bad", Expected: TokComment},
		{Kind: TokRBrace, Str: "}"},
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
		{Kind: TokMessage, Str: "message"},
		{Kind: TokIden, Str: "Data2"},
		{Kind: TokStruct, Str: "struct"},
		{Kind: TokLBrace, Str: "{"},
		{Kind: TokRequired, Str: "required"},
		{Kind: TokIden, Str: "one"},
		{Kind: TokTag, Str: "@1", Int: *big.NewInt(1)},
		{Kind: TokLBrack, Str: "["},
		{Kind: TokErr, Str: "5a", Expected: TokInteger},
		{Kind: TokRBrack, Str: "]"},
		{Kind: TokIden, Str: "Data"},
		{Kind: TokSemicolon, Str: ";"},
		{Kind: TokRBrace, Str: "}"},
		{Kind: TokEof},
	}

	assert.Equal(t, expTokens, tokens)
}

func TestLexer_BadTag(t *testing.T) {
	input := `
	message Data struct {
		required one @ Data;
	}
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{Kind: TokMessage, Str: "message"},
		{Kind: TokIden, Str: "Data"},
		{Kind: TokStruct, Str: "struct"},
		{Kind: TokLBrace, Str: "{"},
		{Kind: TokRequired, Str: "required"},
		{Kind: TokIden, Str: "one"},
		{Kind: TokErr, Str: "@", Expected: TokTag},
		{Kind: TokIden, Str: "Data"},
		{Kind: TokSemicolon, Str: ";"},
		{Kind: TokRBrace, Str: "}"},
		{Kind: TokEof},
	}

	assert.Equal(t, expTokens, tokens)
}