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

type ParsingError struct {
	actual   Token
	nodeKind NodeKind
	expected []TokKind
	errKind  ParseErrKind
	escSeq   rune
}

func makeKindErr(actual Token, kind ParseErrKind) ParserError {
	return &ParsingError{actual: actual, errKind: kind}
}

func makeExpectErr(actual Token, expected ...TokKind) ParserError {
	return &ParsingError{actual: actual, expected: expected, errKind: ExpectErrKind}
}

func makeEscSeqErr(actual Token, escSeq rune) ParserError {
	return &ParsingError{actual: actual, escSeq: escSeq, errKind: EscSeqErrKind}
}

func (err *ParsingError) token() Token {
	return err.actual
}

func (err *ParsingError) addKind(kind NodeKind) {
	if err.nodeKind == NoNodeKind {
		err.nodeKind = kind
	}
}

func (err *ParsingError) withKind(kind NodeKind) ParserError {
	err.addKind(kind)
	return err
}

func (err *ParsingError) Error() string {
	var sb strings.Builder

	// header
	sb.WriteString(err.actual.Positions.Offset())
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
		fmt.Fprintf(&sb, "invalid escape sequence: '/%c'", err.escSeq)
	case NumErrKind:
		fmt.Fprintf(&sb, "%s is an invalid integer", err.actual.String())
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
	if err.nodeKind != NoNodeKind {
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

type ValidErrKind int

const (
	RedefErrKind ValidErrKind = iota
	UndefErrKind
	FirstOrdErrKind
	OrdErrKind
	TypeArgErrKind
)

type ValidateErr struct {
	eKind       ValidErrKind
	p           Positions
	nKind       NodeKind
	iden        string
	expOrd      uint64
	gotOrd      uint64
	expTypeArgs []string
	gotTypeArgs []TypeNode
}

func makeRedefErr(nKind NodeKind, p Positions, iden string) error {
	return &ValidateErr{eKind: RedefErrKind, p: p, nKind: nKind, iden: iden}
}

func makeUndefErr(nKind NodeKind, p Positions, iden string) error {
	return &ValidateErr{eKind: UndefErrKind, p: p, nKind: nKind, iden: iden}
}

func makeOrdErr(nKind NodeKind, p Positions, expOrd uint64, gotOrd uint64) error {
	return &ValidateErr{eKind: OrdErrKind, p: p, nKind: nKind, expOrd: expOrd, gotOrd: gotOrd}
}

func makeTypeArgErr(nKind NodeKind, p Positions, expTypeArgs []string, gotTypeArgs []TypeNode) error {
	return &ValidateErr{eKind: TypeArgErrKind, p: p, nKind: nKind, expTypeArgs: expTypeArgs, gotTypeArgs: gotTypeArgs}
}

func (err *ValidateErr) Error() string {
	var sb strings.Builder
	sb.WriteString(err.p.Offset())
	sb.WriteRune(' ')
	sb.WriteString(err.nKind.String())
	sb.WriteString(": ")

	switch err.eKind {
	case RedefErrKind:
		fmt.Fprintf(&sb, "\"%s\" is redefined", err.iden)
	case UndefErrKind:
		fmt.Fprintf(&sb, "\"%s\" is undefined", err.iden)
	case OrdErrKind:
		fmt.Fprintf(&sb, "order tag '@%d' should be '@%d'", err.gotOrd, err.expOrd)
	case TypeArgErrKind:
		fmt.Fprintf(&sb, "expected %d type arguments", len(err.expTypeArgs))
		if len(err.expTypeArgs) > 0 {
			sb.WriteString(": ")
		}
		FmtTypeParams(&sb, err.expTypeArgs)
		fmt.Fprintf(&sb, ", got %d type arguments", len(err.gotTypeArgs))
		if len(err.gotTypeArgs) > 0 {
			sb.WriteString(": ")
		}
		FmtTypeArgs(&sb, err.gotTypeArgs)
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
		switch err := err.(type) {
		case *ParsingError:
			err.actual.Positions = Positions{}
		}
	}
}
