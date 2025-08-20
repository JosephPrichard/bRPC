package internal

import (
	"fmt"
	"strings"
)

type ParserError interface {
	error
	addKind(AstKind)
	token() Token
}

type ParsingErr struct {
	actual   Token
	kind     AstKind
	expected []TokKind
	message  string
}

func (err *ParsingErr) token() Token {
	return err.actual
}

func (err *ParsingErr) addKind(kind AstKind) {
	if err.kind == UnknownAstKind {
		err.kind = kind
	}
}

func (err *ParsingErr) Error() string {
	var sb strings.Builder
	sb.WriteString(err.message)
	if len(err.expected) > 0 {
		sb.WriteString("expected ")
		for i, tok := range err.expected {
			var delim string
			if i == len(err.expected)-2 {
				delim = " or "
			} else if i != len(err.expected)-1 {
				delim = ", "
			}
			sb.WriteString(tok.String())
			sb.WriteString(delim)
		}
	}
	return fmt.Sprintf("%s %s, found %s while parsing %s", err.actual.Range.Header(), sb.String(), err.actual.String(), err.kind.String())
}

func makeMessageErr(actual Token, message string) ParserError {
	return &ParsingErr{actual: actual, message: message}
}

func makeExpectErr(actual Token, expected ...TokKind) ParserError {
	return &ParsingErr{actual: actual, expected: expected}
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
