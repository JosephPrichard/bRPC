package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLexer_Struct(t *testing.T) {
	input := `
	message Data1 struct {
		required one b128 = 1; // this is the first comment
		required two []b5 = 2; // this is the second comment
		optional three [16]b4 = 3;
	}

	message Data2 struct {
		required one Data = 1;
	}
	`

	l := newLexer(input)
	go l.Run()

	var tokens []TokVal
	for token := range l.tokens {
		tokens = append(tokens, token.TokVal)
	}

	expTokens := []TokVal{
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data1"},
		{T: TokStruct, Value: "struct"},
		{T: TokLBrace, Value: "{"},
		{T: TokRequired, Value: "required"},
		{T: TokIden, Value: "one"},
		{T: TokIden, Value: "b128"},
		{T: TokEqual, Value: "="},
		{T: TokOrder, Value: "1"},
		{T: TokTerm, Value: ";"},
		{T: TokRequired, Value: "required"},
		{T: TokIden, Value: "two"},
		{T: TokLBrack, Value: "["},
		{T: TokRBrack, Value: "]"},
		{T: TokIden, Value: "b5"},
		{T: TokEqual, Value: "="},
		{T: TokOrder, Value: "2"},
		{T: TokTerm, Value: ";"},
		{T: TokOptional, Value: "optional"},
		{T: TokIden, Value: "three"},
		{T: TokLBrack, Value: "["},
		{T: TokOrder, Value: "16"},
		{T: TokRBrack, Value: "]"},
		{T: TokIden, Value: "b4"},
		{T: TokEqual, Value: "="},
		{T: TokOrder, Value: "3"},
		{T: TokTerm, Value: ";"},
		{T: TokRBrace, Value: "}"},
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data2"},
		{T: TokStruct, Value: "struct"},
		{T: TokLBrace, Value: "{"},
		{T: TokRequired, Value: "required"},
		{T: TokIden, Value: "one"},
		{T: TokIden, Value: "Data"},
		{T: TokEqual, Value: "="},
		{T: TokOrder, Value: "1"},
		{T: TokTerm, Value: ";"},
		{T: TokRBrace, Value: "}"},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_Union(t *testing.T) {
	input := `
	message Data3 enum {
		One = 1;
		Two = 2;
		Three = 3;
	}
	
	message Data4 union {
		Data1, 
		Data2
	}
	`

	l := newLexer(input)
	go l.Run()

	var tokens []TokVal
	for token := range l.tokens {
		tokens = append(tokens, token.TokVal)
	}

	expTokens := []TokVal{
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data3"},
		{T: TokEnum, Value: "enum"},
		{T: TokLBrace, Value: "{"},
		{T: TokIden, Value: "One"},
		{T: TokEqual, Value: "="},
		{T: TokOrder, Value: "1"},
		{T: TokTerm, Value: ";"},
		{T: TokIden, Value: "Two"},
		{T: TokEqual, Value: "="},
		{T: TokOrder, Value: "2"},
		{T: TokTerm, Value: ";"},
		{T: TokIden, Value: "Three"},
		{T: TokEqual, Value: "="},
		{T: TokOrder, Value: "3"},
		{T: TokTerm, Value: ";"},
		{T: TokRBrace, Value: "}"},
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data4"},
		{T: TokUnion, Value: "union"},
		{T: TokLBrace, Value: "{"},
		{T: TokIden, Value: "Data1"},
		{T: TokSep, Value: ","},
		{T: TokIden, Value: "Data2"},
		{T: TokRBrace, Value: "}"},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_Empty(t *testing.T) {
	input := ""

	l := newLexer(input)
	go l.Run()

	var tokens []TokVal
	for token := range l.tokens {
		tokens = append(tokens, token.TokVal)
	}

	assert.Nil(t, tokens)
}

func TestLexer_BadNumber(t *testing.T) {
	input := `
	message Data3 struct {
		required one b128 = 1abc;
	}
	`

	l := newLexer(input)
	go l.Run()

	var tokens []TokVal
	for token := range l.tokens {
		tokens = append(tokens, token.TokVal)
	}

	expTokens := []TokVal{
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data3"},
		{T: TokStruct, Value: "struct"},
		{T: TokLBrace, Value: "{"},
		{T: TokRequired, Value: "required"},
		{T: TokIden, Value: "one"},
		{T: TokIden, Value: "b128"},
		{T: TokEqual, Value: "="},
		{T: TokErr, Value: "1a"},
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

	l := newLexer(input)
	go l.Run()

	var tokens []TokVal
	for token := range l.tokens {
		tokens = append(tokens, token.TokVal)
	}

	expTokens := []TokVal{
		{T: TokMessage, Value: "message"},
		{T: TokIden, Value: "Data3"},
		{T: TokEnum, Value: "enum"},
		{T: TokLBrace, Value: "{"},
		{T: TokIden, Value: "One"},
		{T: TokEqual, Value: "="},
		{T: TokOrder, Value: "1"},
		{T: TokTerm, Value: ";"},
		{T: TokErr, Value: "/#"},
	}
	assert.Equal(t, expTokens, tokens)
}

func TestLexer_BadArrow(t *testing.T) {
	input := `
	service ThingService {
    	DoThing (input Input) -^ (output Output); 
	}
	`

	l := newLexer(input)
	go l.Run()

	var tokens []TokVal
	for token := range l.tokens {
		tokens = append(tokens, token.TokVal)
	}

	expTokens := []TokVal{
		{T: TokService, Value: "service"},
		{T: TokIden, Value: "ThingService"},
		{T: TokLBrace, Value: "{"},
		{T: TokIden, Value: "DoThing"},
		{T: TokLParen, Value: "("},
		{T: TokIden, Value: "input"},
		{T: TokIden, Value: "Input"},
		{T: TokRParen, Value: ")"},
		{T: TokErr, Value: "-^"},
	}
	assert.Equal(t, expTokens, tokens)
}
