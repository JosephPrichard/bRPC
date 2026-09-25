package internal

import (
	"fmt"
	"strings"
)

type ParseErrKind int

const (
	NoParseErrKind ParseErrKind = iota
	EofErrKind
	ExpectErrKind
	EscSeqErrKind
	SizeErrKind
	IdenErrKind
	NumErrKind
)

type ParseError struct {
	errKind     ParseErrKind
	actualToken Token
	nodeKind    NodeKind
	expected    []TokKind
	escSeq      rune
}

func makeEofErr() ParseError {
	return ParseError{errKind: EofErrKind}
}

func makeKindErr(actual Token, kind ParseErrKind) ParseError {
	return ParseError{actualToken: actual, errKind: kind}
}

func makeExpectErr(actual Token, expected ...TokKind) ParseError {
	return ParseError{actualToken: actual, expected: expected, errKind: ExpectErrKind}
}

func makeEscSeqErr(actual Token, escSeq rune) ParseError {
	return ParseError{actualToken: actual, escSeq: escSeq, errKind: EscSeqErrKind}
}

func (err *ParseError) isPresent() bool {
	return err.errKind != NoParseErrKind
}

func (err *ParseError) addKind(kind NodeKind) {
	if err.nodeKind == NoNodeKind {
		err.nodeKind = kind
	}
}

func (err ParseError) withKind(kind NodeKind) ParseError {
	err.addKind(kind)
	return err
}

func (err ParseError) Error() string {
	return err.String()
}

func (err ParseError) String() string {
	var sb strings.Builder

	// header
	sb.WriteString(err.actualToken.Positions.Offset())
	sb.WriteRune(' ')

	// text
	switch err.errKind {
	case EofErrKind:
		sb.WriteString("reached end of stream while parsing")
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
		fmt.Fprintf(&sb, "%s is an invalid integer", err.actualToken.String())
	case SizeErrKind:
		sb.WriteString("struct does not allow a size argument")
	case IdenErrKind:
		sb.WriteString("Iden must begin with an uppercase and only contain alphanumerics")
	default:
		panic(fmt.Sprintf("assertion errror: unknown parse errKind: %d", err.errKind))
	}

	// actual
	if err.errKind != EofErrKind {
		sb.WriteString(", found ")
		sb.WriteString(err.actualToken.String())
	}

	// node
	if err.nodeKind != NoNodeKind {
		sb.WriteString(" while parsing ")
		sb.WriteString(err.nodeKind.String())
	}

	// expected
	switch err.actualToken.Expected {
	case TokTag:
		sb.WriteString(": an ord must contain an '@' followed by an integer")
	case TokInteger:
		sb.WriteString(": an integer must only contain numeric characters")
	default:
	}

	return sb.String()
}

type ValidErrKind int

const (
	NoValidateErrKind ValidErrKind = iota
	RedefErrKind
	UndefErrKind
	FirstOrdErrKind
	TagErrKind
	TypeArgErrKind
)

type ValidateErr struct {
	errKind     ValidErrKind
	positions   Positions
	nodeKind    NodeKind
	iden        string
	expTag      uint64
	gotTag      uint64
	expTypeArgs []string
	gotTypeArgs []TypeNode
}

func makeRedefErr(nKind NodeKind, positions Positions, Iden string) ValidateErr {
	return ValidateErr{errKind: RedefErrKind, positions: positions, nodeKind: nKind, iden: Iden}
}

func makeUndefErr(nKind NodeKind, positions Positions, Iden string) ValidateErr {
	return ValidateErr{errKind: UndefErrKind, positions: positions, nodeKind: nKind, iden: Iden}
}

func makeTagErr(nKind NodeKind, positions Positions, expOrd uint64, gotOrd uint64) ValidateErr {
	return ValidateErr{errKind: TagErrKind, positions: positions, nodeKind: nKind, expTag: expOrd, gotTag: gotOrd}
}

func makeTypeArgErr(nKind NodeKind, positions Positions, expTypeArgs []string, gotTypeArgs []TypeNode) ValidateErr {
	return ValidateErr{errKind: TypeArgErrKind, positions: positions, nodeKind: nKind, expTypeArgs: expTypeArgs, gotTypeArgs: gotTypeArgs}
}

func (err *ValidateErr) isPresent() bool {
	return err.errKind != NoValidateErrKind
}

func (err ValidateErr) Error() string {
	return err.String()
}

func (err ValidateErr) String() string {
	var sb strings.Builder
	sb.WriteString(err.positions.Offset())
	sb.WriteRune(' ')
	sb.WriteString(err.nodeKind.String())
	sb.WriteString(": ")

	switch err.errKind {
	case RedefErrKind:
		fmt.Fprintf(&sb, "\"%s\" is redefined", err.iden)
	case UndefErrKind:
		fmt.Fprintf(&sb, "\"%s\" is undefined", err.iden)
	case TagErrKind:
		fmt.Fprintf(&sb, "order tag '@%d' should be '@%d'", err.gotTag, err.expTag)
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

func printErrors[T error](errs []T, filePath string, printLine func(string)) {
	for _, err := range errs {
		printLine(fmt.Sprintf("%s:%s", filePath, err.Error()))
	}
}

func clearParseErrors(errs []ParseError) {
	for i := range errs {
		errs[i].actualToken.Positions = Positions{}
	}
}

func clearValidateErrors(errs []ValidateErr) {
	for i := range errs {
		errs[i].positions = Positions{}
	}
}
