package internal

import (
	"fmt"
	"strings"
)

type ParserError interface {
	error
	token() Token
	addKind(kind NodeKind)
	withKind(kind NodeKind) ParserError
}

type ParseErrKind int

const (
	ExpectErrKind ParseErrKind = iota
	EscSeqErrKind
	SizeErrKind
	IdenErrKind
	NumErrKind
)

type ParseErr struct {
	actual   Token
	nodeKind NodeKind
	expected []TokKind
	errKind  ParseErrKind
	escSeq   rune
}

func makeKindErr(actual Token, kind ParseErrKind) ParserError {
	return &ParseErr{actual: actual, errKind: kind}
}

func makeExpectErr(actual Token, expected ...TokKind) ParserError {
	return &ParseErr{actual: actual, expected: expected, errKind: ExpectErrKind}
}

func makeEscSeqErr(actual Token, escSeq rune) ParserError {
	return &ParseErr{actual: actual, escSeq: escSeq, errKind: EscSeqErrKind}
}

func (err *ParseErr) token() Token {
	return err.actual
}

func (err *ParseErr) addKind(kind NodeKind) {
	if err.nodeKind == UnknownNodeKind {
		err.nodeKind = kind
	}
}

func (err *ParseErr) withKind(kind NodeKind) ParserError {
	err.addKind(kind)
	return err
}

func (err *ParseErr) Error() string {
	var sb strings.Builder

	// header
	sb.WriteString(err.actual.Positions.Header())
	sb.WriteRune(' ')

	// text
	switch err.errKind {
	case ExpectErrKind:
		sb.WriteString("expected ")
		for i, tok := range err.expected {
			dlm := ""
			if i == len(err.expected)-2 {
				dlm = " or "
			} else if i != len(err.expected)-1 {
				dlm = ", "
			}
			sb.WriteString(tok.String())
			sb.WriteString(dlm)
		}
	case EscSeqErrKind:
		sb.WriteString(fmt.Sprintf("invalid escape sequence: '/%c'", err.escSeq))
	case NumErrKind:
		sb.WriteString(fmt.Sprintf("%s is an invalid integer", err.actual.String()))
	case SizeErrKind:
		sb.WriteString("struct does not allow a size argument")
	case IdenErrKind:
		sb.WriteString("iden must begin with an uppercase and only contain alphanumerics")
	default:
		panic(fmt.Sprintf("assertion errror: unknown parse errKind: %d", err.errKind))
	}

	// actual
	sb.WriteString(", found ")
	sb.WriteString(err.actual.String())

	// node
	if err.nodeKind != UnknownNodeKind {
		sb.WriteString(" while parsing ")
		sb.WriteString(err.nodeKind.String())
	}

	// expected
	switch err.actual.Expected {
	case TokOrd:
		sb.WriteString(": an ord must contain an '@' followed by an integer")
	case TokInteger:
		sb.WriteString(": an integer must only contain numeric characters")
	default:
	}

	return sb.String()
}

type TransformErrKind int

const (
	RedefErrKind TransformErrKind = iota
	UndefErrKind
	FirstOrdErrKind
	OrdErrKind
)

type TransformErr struct {
	kind   TransformErrKind
	node   Node
	iden   string
	expOrd uint64
	gotOrd uint64
}

func makeRedefErr(node Node, iden string) error {
	return &TransformErr{node: node, kind: RedefErrKind, iden: iden}
}

func makeUndefErr(node Node, iden string) error {
	return &TransformErr{node: node, kind: UndefErrKind, iden: iden}
}

func makeOrdErr(node Node, expOrd uint64, gotOrd uint64) error {
	return &TransformErr{node: node, kind: OrdErrKind, expOrd: expOrd, gotOrd: gotOrd}
}

func (err *TransformErr) Error() string {
	var sb strings.Builder
	sb.WriteString(err.node.Header())
	sb.WriteRune(' ')
	sb.WriteString(err.node.Kind().String())
	sb.WriteString(": ")

	switch err.kind {
	case RedefErrKind:
		sb.WriteString(fmt.Sprintf("\"%s\" is redefined", err.iden))
	case UndefErrKind:
		sb.WriteString(fmt.Sprintf("\"%s\" is undefined", err.iden))
	case OrdErrKind:
		sb.WriteString(fmt.Sprintf("order tag '@%d' should be '@%d'", err.gotOrd, err.expOrd))

	}

	return sb.String()
}

func printErrors(errs []error, filePath string, printLine func(string)) {
	for _, err := range errs {
		printLine(fmt.Sprintf("%s:%s", filePath, err.Error()))
	}
}

func clearErrors(errs []error) {
	for _, err := range errs {
		if pErr, ok := err.(*ParseErr); ok {
			pErr.actual.Positions = Positions{}
		}
	}
}
