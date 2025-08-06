package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func runLexer(input string) []TokVal {
	lex := NewLexer(input)
	go lex.Run()

	var tokens []TokVal
	for token := range lex.Tokens {
		tokens = append(tokens, token.TokVal)
	}

	return tokens
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
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data1"},
		{T: TokStruct, Value: "struct"},
		{T: TokLBrace, Value: "{"},
		{T: TokRequired, Value: "required"},
		{T: TokIden, Value: "one"},
		{T: TokOrd, Value: "@1"},
		{T: TokIden, Value: "b128"},
		{T: TokTerm, Value: ";"},
		{T: TokRequired, Value: "required"},
		{T: TokIden, Value: "two"},
		{T: TokOrd, Value: "@2"},
		{T: TokLBrack, Value: "["},
		{T: TokRBrack, Value: "]"},
		{T: TokIden, Value: "b5"},
		{T: TokTerm, Value: ";"},
		{T: TokOptional, Value: "optional"},
		{T: TokIden, Value: "three"},
		{T: TokOrd, Value: "@3"},
		{T: TokLBrack, Value: "["},
		{T: TokInteger, Value: "16"},
		{T: TokRBrack, Value: "]"},
		{T: TokIden, Value: "b4"},
		{T: TokTerm, Value: ";"},
		{T: TokRBrace, Value: "}"},
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data2"},
		{T: TokStruct, Value: "struct"},
		{T: TokLBrace, Value: "{"},
		{T: TokRequired, Value: "required"},
		{T: TokIden, Value: "one"},
		{T: TokOrd, Value: "@1"},
		{T: TokIden, Value: "Data"},
		{T: TokTerm, Value: ";"},
		{T: TokRBrace, Value: "}"},
		{T: TokEof},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_Union(t *testing.T) {
	input := `
	message Data3 enum {
		One = 0b1010;
		Two = 0xabc1;
		Three = 3;
	}
	
	message Data4 union = Data1 | Data2;
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data3"},
		{T: TokEnum, Value: "enum"},
		{T: TokLBrace, Value: "{"},
		{T: TokIden, Value: "One"},
		{T: TokEqual, Value: "="},
		{T: TokBinary, Value: "0b1010"},
		{T: TokTerm, Value: ";"},
		{T: TokIden, Value: "Two"},
		{T: TokEqual, Value: "="},
		{T: TokHex, Value: "0xabc1"},
		{T: TokTerm, Value: ";"},
		{T: TokIden, Value: "Three"},
		{T: TokEqual, Value: "="},
		{T: TokInteger, Value: "3"},
		{T: TokTerm, Value: ";"},
		{T: TokRBrace, Value: "}"},
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data4"},
		{T: TokUnion, Value: "union"},
		{T: TokEqual, Value: "="},
		{T: TokIden, Value: "Data1"},
		{T: TokPipe, Value: "|"},
		{T: TokIden, Value: "Data2"},
		{T: TokTerm, Value: ";"},
		{T: TokEof},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_Service(t *testing.T) {
	input := `
	service ThingService {
    	rpc DoThis (Input) returns (Output)
		rpc DoThat (In) returns (Out)
	}
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{T: TokService, Value: "service"},
		{T: TokIden, Value: "ThingService"},
		{T: TokLBrace, Value: "{"},
		{T: TokRpc, Value: "rpc"},
		{T: TokIden, Value: "DoThis"},
		{T: TokLParen, Value: "("},
		{T: TokIden, Value: "Input"},
		{T: TokRParen, Value: ")"},
		{T: TokReturns, Value: "returns"},
		{T: TokLParen, Value: "("},
		{T: TokIden, Value: "Output"},
		{T: TokRParen, Value: ")"},
		{T: TokRpc, Value: "rpc"},
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
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data3"},
		{T: TokStruct, Value: "struct"},
		{T: TokLBrace, Value: "{"},
		{T: TokRequired, Value: "required"},
		{T: TokIden, Value: "one"},
		{T: TokErr, Value: "@1a"},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_BadComment(t *testing.T) {
	input := `
	message Data3 enum {
		One = 1; /# this is a bad comment
		Two = 2;
		Three = 3;
	}
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data3"},
		{T: TokEnum, Value: "enum"},
		{T: TokLBrace, Value: "{"},
		{T: TokIden, Value: "One"},
		{T: TokEqual, Value: "="},
		{T: TokInteger, Value: "1"},
		{T: TokTerm, Value: ";"},
		{T: TokErr, Value: "/#"},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_BadBinary(t *testing.T) {
	input := `
	message Data3 enum {
		One = 0B102;
		Two = 123;
		Three = 0b321;
	}
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data3"},
		{T: TokEnum, Value: "enum"},
		{T: TokLBrace, Value: "{"},
		{T: TokIden, Value: "One"},
		{T: TokEqual, Value: "="},
		{T: TokErr, Value: "0B102"},
		{T: TokTerm, Value: ";"},
		{T: TokIden, Value: "Two"},
		{T: TokEqual, Value: "="},
		{T: TokInteger, Value: "123"},
		{T: TokTerm, Value: ";"},
		{T: TokIden, Value: "Three"},
		{T: TokEqual, Value: "="},
		{T: TokErr, Value: "0b321"},
		{T: TokTerm, Value: ";"},
		{T: TokRBrace, Value: "}"},
		{T: TokEof},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_BadHex(t *testing.T) {
	input := `
	message Data3 enum {
		One = 0Xabcx;
		Two = 123;
	}
	`
	tokens := runLexer(input)

	expTokens := []TokVal{
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data3"},
		{T: TokEnum, Value: "enum"},
		{T: TokLBrace, Value: "{"},
		{T: TokIden, Value: "One"},
		{T: TokEqual, Value: "="},
		{T: TokErr, Value: "0Xabcx"},
		{T: TokTerm, Value: ";"},
		{T: TokIden, Value: "Two"},
		{T: TokEqual, Value: "="},
		{T: TokInteger, Value: "123"},
		{T: TokTerm, Value: ";"},
		{T: TokRBrace, Value: "}"},
		{T: TokEof},
	}
	assert.Equal(t, expTokens, tokens)
}
