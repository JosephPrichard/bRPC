package internal

import (
	"fmt"
	"strings"
)

type ParserError interface {
	error
	addKind(NodeKind)
	token() Token
}

type ParsingErr struct {
	actual   Token
	kind     NodeKind
	expected []TokKind
	text     string
}

func (err *ParsingErr) token() Token {
	return err.actual
}

func (err *ParsingErr) addKind(kind NodeKind) {
	if err.kind == UnknownNodeKind {
		err.kind = kind
	}
}

func addKind(err ParserError, kind NodeKind) ParserError {
	err.addKind(kind)
	return err
}

func (err *ParsingErr) Error() string {
	var sb strings.Builder
	sb.WriteString(err.text)
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
	return fmt.Sprintf("%s %s, found %s while parsing %s", err.actual.Positions.Header(), sb.String(), err.actual.String(), err.kind.String())
}

func makeTextErr(actual Token, text string) ParserError {
	return &ParsingErr{actual: actual, text: text}
}

func makeExpectErr(actual Token, expected ...TokKind) ParserError {
	return &ParsingErr{actual: actual, expected: expected}
}

type CodegenErr struct {
	ast Node
	msg string
}

func (err *CodegenErr) Error() string {
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
