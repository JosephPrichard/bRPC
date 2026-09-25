package internal

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidation_Errors(t *testing.T) {
	type Test struct {
		name  string
		input string
		errs  []ValidateErr
	}

	tests := []Test{
		{
			name: "DuplicateTypeIden",
			input: `
			message Data struct {}
			message Data struct {
				message Data1 struct {
					message Data1 struct {}
				}
				message Data1 struct {}
			}
			`,
			errs: []ValidateErr{
				{
					errKind: RedefErrKind,
					nodeKind: StructNodeKind,
					iden:     "Data",
				},
				{
					errKind: RedefErrKind,
					nodeKind: StructNodeKind,
					iden:     "Data1",
				},
			},
		},
		{
			name: "DuplicateFieldIdens",
			input: `
			message Data1 struct {
				required one @1 int16;
				deprecated one @2 int16;
			}
			message Data2 enum {
				@1 One;
				@2 One;
			}
			message Data3 union {
				one @1 Data1;
				two @2 Data2;
			}
			`,
			errs: []ValidateErr{
				{
					errKind:  RedefErrKind,
					nodeKind: FieldNodeKind,
					iden:     "one",
				},
				{
					errKind:  RedefErrKind,
					nodeKind: CaseNodeKind,
					iden:     "One",
				},
			},
		},
		{
			name: "InvalidTags",
			input: `
			message Data1 struct {
				required one @1 int16;
				deprecated two @2 int16;
				deprecated one @3 int16;
			}
			message Data2 enum {
				@1 One;
				@1 One;
			}
			message Data3 union {
				one @1 Data1;
				two @1 Data2;
			}
			message Data4 union {
				one @0 Data1;
				two @4 Data2;
			}
			`,
			errs: []ValidateErr{
				{
					errKind: RedefErrKind,
					iden:    "one",
					nodeKind: FieldNodeKind,
				},
				{
					errKind: TagErrKind,
					iden:    "",
					nodeKind: CaseNodeKind,
					gotTag:   1,
					expTag:   2,
				},
				{
					errKind: RedefErrKind,
					iden:    "One",
					nodeKind: CaseNodeKind,
				},
				{
					errKind: TagErrKind,
					iden:    "",
					nodeKind: OptionNodeKind,
					gotTag:   1,
					expTag:   2,
				},
				{
					errKind: TagErrKind,
					iden:    "",
					nodeKind: OptionNodeKind,
					gotTag:   0,
					expTag:   1,
				},
			},
		},
		{
			name: "UnresolvedIden",
			input: `
			message Data1 struct {
				required one @1 int16;
				deprecated two @2 int16;
				deprecated one @3 int16;

				message Data3 union {
					one @1 Data2;
					two @2 Invalid;
				}
			}
			message Data2 struct {
				required one @1 Data1;
				required two @2 Invalid;
			}
			`,
			errs: []ValidateErr{
				{
					errKind:  RedefErrKind,
					iden:     "one",
					nodeKind: FieldNodeKind,
				},
				{
					errKind:  UndefErrKind,
					iden:     "Invalid",
					nodeKind: OptionNodeKind,
				},
				{
					errKind:  UndefErrKind,
					iden:     "Invalid",
					nodeKind: FieldNodeKind,
				},
			},
		},
		// {
		// 	name: "RecursiveAst",
		// 	input: `
		// 	message Data1 struct {
		// 		required one @1 int16;
		// 		deprecated two @2 Data1;
		// 		deprecated one @3 int16;

		// 		message Data4 union {
		// 			one @1 Data1;
		// 			two @2 Data4;
		// 		}
		// 	}

		// 	message Data2 struct {
		// 		required one @1 Data3;

		// 		message Data3 struct {
		// 			required one @1 Data2;
		// 		}
		// 	}
		// 	`,
		// 	errs: []ValidateErr{},
		// },
		// {
		// 	name: "InvalidTypeArgs",
		// 	input: `
		// 	message Data1 struct {
		// 		required one @1 Data2;
		// 		required two @2 Data2(int8);
		// 		required three @3 Data2(int16, int18);
		// 	}

		// 	message Data2 union(A) {
		// 		one @1 A;
		// 		two @2 B;
		// 	}
		// 	`,
		// 	errs: []error{},
		// },
		// {
		// 	name: "RecursiveTypeArgs",
		// 	input: `
		// 	message Data3 struct(A) {
		// 		deprecated one @1 A;
		// 	}

		// 	message Data2 union {
		// 		one @1 Data3(Data2);
		// 		two @2 int16;
		// 	}
		// 	`,
		// 	errs: []error{},
		// },
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("test/%s", test.name), func(t *testing.T) {
			var errs []ValidateErr
			runValidator(parseOrElse(test.input), &errs)

			printLine := func(err string) { t.Log(err) }
			printErrors(errs, "test", printLine)
			clearValidateErrors(errs)

			assert.Equal(t, test.errs, errs)
		})
	}
}
