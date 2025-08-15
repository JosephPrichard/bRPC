package internal

import "strings"

type ParseError interface {
	error
	addKind(AstKind)
}

type ExpectErr struct {
	actual   Token
	kind     AstKind
	expected string
}

func (err *ExpectErr) addKind(kind AstKind) {
	err.kind = kind
}

func (err *ExpectErr) Error() string {
	return err.actual.FormatPosition() + "expected " + err.expected + " but got " + err.actual.String() + " while parsing " + err.kind.String()
}

func makeParseErr(actual Token, expected string) ParseError {
	return &ExpectErr{actual: actual, expected: expected}
}

type TokenErr struct {
	actual   Token
	kind     AstKind
	expected []TokType
}

func (err *TokenErr) addKind(kind AstKind) {
	err.kind = kind
}

func (err *TokenErr) Error() string {
	var sb strings.Builder

	sb.WriteString(err.actual.FormatPosition())
	sb.WriteString("expected ")
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

type TypeErr struct {
	actual Token
	kind   AstKind
	msg    string
}

func (err *TypeErr) Error() string {
	return ""
}

func printErrors(errs []error, filePath string, printLine func(string)) {
	for _, err := range errs {
		var sb strings.Builder

		sb.WriteString(filePath + ":")
		sb.WriteString(err.Error())

		printLine(sb.String())
	}
}
