package internal

import "strings"

type ParseError interface {
	error
	addKind(AstKind)
	token() Token
}

type ExpectErr struct {
	actual  Token
	kind    AstKind
	message string
}

func (err *ExpectErr) token() Token {
	return err.actual
}

func (err *ExpectErr) addKind(kind AstKind) {
	err.kind = kind
}

func (err *ExpectErr) Error() string {
	return err.actual.Range.Header() + err.message + " at " + err.actual.String() + " while parsing " + err.kind.String()
}

func makeParseErr(actual Token, expected string) ParseError {
	return &ExpectErr{actual: actual, message: expected}
}

type TokenErr struct {
	actual   Token
	kind     AstKind
	expected []TokType
}

func (err *TokenErr) token() Token {
	return err.actual
}

func (err *TokenErr) addKind(kind AstKind) {
	err.kind = kind
}

func (err *TokenErr) Error() string {
	var sb strings.Builder

	sb.WriteString(err.actual.Range.Header())
	sb.WriteString("message ")
	for i, tok := range err.expected {
		sb.WriteString(tok.String())
		if i == len(err.expected)-2 {
			sb.WriteString(" or ")
		} else if i != len(err.expected)-1 {
			sb.WriteString(", ")
		}
	}

	sb.WriteString(" but got ")
	sb.WriteString(err.actual.String())
	sb.WriteString(" while parsing ")
	sb.WriteString(err.kind.String())

	return sb.String()
}

func makeTokenErr(actual Token, expected ...TokType) ParseError {
	return &TokenErr{actual: actual, expected: expected}
}

type AstErr struct {
	ast Ast
	msg string
}

func (err *AstErr) Error() string {
	return err.ast.Header() + err.msg + " while inside " + err.ast.Kind().String()
}

func printErrors(errs []error, filePath string, printLine func(string)) {
	for _, err := range errs {
		var sb strings.Builder

		sb.WriteString(filePath + ":")
		sb.WriteString(err.Error())

		printLine(sb.String())
	}
}
