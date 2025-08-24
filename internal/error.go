package internal

import (
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
	sb.WriteString(err.actual.Positions.Header())
	sb.WriteRune(' ')

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

	sb.WriteString(", found ")
	sb.WriteString(err.actual.String())
	if err.kind != UnknownNodeKind {
		sb.WriteString(" while parsing ")
		sb.WriteString(err.kind.String())
	}

	var postfix string
	switch err.actual.Expected {
	case TokOrd:
		postfix = ": an ord must contain an '@' followed by an integer"
	case TokInteger:
		postfix = ": an integer must only contain numeric characters"
	default:
		// no messages for other expected tokens
	}
	sb.WriteString(postfix)

	return sb.String()
}

func makeTextErr(actual Token, text string) ParserError {
	return &ParsingErr{actual: actual, text: text}
}

func makeExpectErr(actual Token, expected ...TokKind) ParserError {
	return &ParsingErr{actual: actual, expected: expected}
}

type CodegenErr struct {
	node Node
	msg  string
}

func (err *CodegenErr) Error() string {
	return err.node.Header() + err.msg + " while inside " + err.node.Kind().String()
}

func printErrors(errs []error, filePath string, printLine func(string)) {
	for _, err := range errs {
		var sb strings.Builder

		sb.WriteString(filePath + ":")
		sb.WriteString(err.Error())

		printLine(sb.String())
	}
}
