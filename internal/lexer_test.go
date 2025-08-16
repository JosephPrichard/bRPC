package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
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
		{T: TokImport, Value: "import"},
		{T: TokString, Value: "\"/path/to/idl/idl.brpc\""},
		{T: TokIden, Value: "package"},
		{T: TokEqual, Value: "="},
		{T: TokString, Value: "\"/hello/\\\\\\\"world\\\"\""},
		{T: TokIden, Value: "constant"},
		{T: TokEqual, Value: "="},
		{T: TokString, Value: "\"typValue\""},
		{T: TokEof},
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
		required one @1 A;
        required two @2 B;
	}
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data1"},
		{T: TokStruct, Value: "struct"},
		{T: TokLBrace, Value: "{"},
		{T: TokRequired, Value: "required"},
		{T: TokIden, Value: "one"},
		{T: TokOrd, Value: "@1"},
		{T: TokIden, Value: "b128"},
		{T: TokSemicolon, Value: ";"},
		{T: TokRequired, Value: "required"},
		{T: TokIden, Value: "two"},
		{T: TokOrd, Value: "@2"},
		{T: TokLBrack, Value: "["},
		{T: TokRBrack, Value: "]"},
		{T: TokIden, Value: "b5"},
		{T: TokSemicolon, Value: ";"},
		{T: TokOptional, Value: "optional"},
		{T: TokIden, Value: "three"},
		{T: TokOrd, Value: "@3"},
		{T: TokLBrack, Value: "["},
		{T: TokInteger, Value: "16"},
		{T: TokRBrack, Value: "]"},
		{T: TokIden, Value: "b4"},
		{T: TokSemicolon, Value: ";"},
		{T: TokRBrace, Value: "}"},
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data2"},
		{T: TokStruct, Value: "struct"},
		{T: TokLBrack, Value: "["},
		{T: TokIden, Value: "A"},
		{T: TokComma, Value: ","},
		{T: TokIden, Value: "B"},
		{T: TokRBrack, Value: "]"},
		{T: TokLBrace, Value: "{"},
		{T: TokRequired, Value: "required"},
		{T: TokIden, Value: "one"},
		{T: TokOrd, Value: "@1"},
		{T: TokIden, Value: "A"},
		{T: TokSemicolon, Value: ";"},
		{T: TokRequired, Value: "required"},
		{T: TokIden, Value: "two"},
		{T: TokOrd, Value: "@2"},
		{T: TokIden, Value: "B"},
		{T: TokSemicolon, Value: ";"},
		{T: TokRBrace, Value: "}"},
		{T: TokEof},
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
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data3"},
		{T: TokEnum, Value: "enum"},
		{T: TokLBrace, Value: "{"},
		{T: TokOrd, Value: "@1"},
		{T: TokIden, Value: "One"},
		{T: TokSemicolon, Value: ";"},
		{T: TokOrd, Value: "@2"},
		{T: TokIden, Value: "Two"},
		{T: TokSemicolon, Value: ";"},
		{T: TokOrd, Value: "@3"},
		{T: TokIden, Value: "Three"},
		{T: TokSemicolon, Value: ";"},
		{T: TokRBrace, Value: "}"},
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data4"},
		{T: TokUnion, Value: "union"},
		{T: TokEqual, Value: "="},
		{T: TokIden, Value: "Data1"},
		{T: TokPipe, Value: "|"},
		{T: TokIden, Value: "Data2"},
		{T: TokSemicolon, Value: ";"},
		{T: TokEof},
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
		{T: TokService, Value: "service"},
		{T: TokIden, Value: "ThingService"},
		{T: TokLBrace, Value: "{"},
		{T: TokRpc, Value: "rpc"},
		{T: TokOrd, Value: "@1"},
		{T: TokIden, Value: "DoThis"},
		{T: TokLParen, Value: "("},
		{T: TokIden, Value: "input"},
		{T: TokRParen, Value: ")"},
		{T: TokReturns, Value: "returns"},
		{T: TokLParen, Value: "("},
		{T: TokIden, Value: "Output"},
		{T: TokRParen, Value: ")"},
		{T: TokRpc, Value: "rpc"},
		{T: TokOrd, Value: "@2"},
		{T: TokIden, Value: "DoThat"},
		{T: TokLParen, Value: "("},
		{T: TokIden, Value: "In"},
		{T: TokRParen, Value: ")"},
		{T: TokReturns, Value: "returns"},
		{T: TokLParen, Value: "("},
		{T: TokIden, Value: "Out"},
		{T: TokRParen, Value: ")"},
		{T: TokRBrace, Value: "}"},
		{T: TokEof},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_Empty(t *testing.T) {
	input := ""
	tokens := runLexer(input)
	assert.Equal(t, []TokVal{{T: TokEof}}, tokens)
}

func TestLexer_BadOrd(t *testing.T) {
	input := `
	message Data3 struct {
		required one @1abc b128;
	}
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data3"},
		{T: TokStruct, Value: "struct"},
		{T: TokLBrace, Value: "{"},
		{T: TokRequired, Value: "required"},
		{T: TokIden, Value: "one"},
		{T: TokErr, Value: "@1abc", Expected: TokOrd},
		{T: TokIden, Value: "b128"},
		{T: TokSemicolon, Value: ";"},
		{T: TokRBrace, Value: "}"},
		{T: TokEof},
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
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data2"},
		{T: TokStruct, Value: "struct"},
		{T: TokLBrace, Value: "{"},
		{T: TokRequired, Value: "required"},
		{T: TokIden, Value: "one"},
		{T: TokOrd, Value: "@1"},
		{T: TokIden, Value: "Data"},
		{T: TokSemicolon, Value: ";"},
		{T: TokErr, Value: "/# bad", Expected: TokComment},
		{T: TokRBrace, Value: "}"},
		{T: TokEof},
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
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data2"},
		{T: TokStruct, Value: "struct"},
		{T: TokLBrace, Value: "{"},
		{T: TokRequired, Value: "required"},
		{T: TokIden, Value: "one"},
		{T: TokOrd, Value: "@1"},
		{T: TokLBrack, Value: "["},
		{T: TokErr, Value: "5a", Expected: TokInteger},
		{T: TokRBrack, Value: "]"},
		{T: TokIden, Value: "Data"},
		{T: TokSemicolon, Value: ";"},
		{T: TokRBrace, Value: "}"},
		{T: TokEof},
	}

	assert.Equal(t, expTokens, tokens)
}
