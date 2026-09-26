package internal

import (
	"fmt"
	"strings"
	"testing"
	"time"

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
					errKind:  RedefErrKind,
					nodeKind: StructNodeKind,
					iden:     "Data",
				},
				{
					errKind:  RedefErrKind,
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
					errKind:  RedefErrKind,
					iden:     "one",
					nodeKind: FieldNodeKind,
				},
				{
					errKind:  TagErrKind,
					iden:     "",
					nodeKind: CaseNodeKind,
					gotTag:   1,
					expTag:   2,
				},
				{
					errKind:  RedefErrKind,
					iden:     "One",
					nodeKind: CaseNodeKind,
				},
				{
					errKind:  TagErrKind,
					iden:     "",
					nodeKind: OptionNodeKind,
					gotTag:   1,
					expTag:   2,
				},
				{
					errKind:  TagErrKind,
					iden:     "",
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
		{
			name:  "InvalidMessageNames",
			input: `message Data_1 struct { required one @1 int128; } message Data_2 enum { } message Data_3 union { }`,
			errs: []ValidateErr{
				{
					iden:     "Data_1",
					nodeKind: StructNodeKind,
					errKind:  IdenNameErr,
				},
				{
					iden:     "Data_2",
					nodeKind: EnumNodeKind,
					errKind:  IdenNameErr,
				},
				{
					iden:     "Data_3",
					nodeKind: UnionNodeKind,
					errKind:  IdenNameErr,
				},
			},
		},
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
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("test/%s", test.name), func(t *testing.T) {
			errs := Validate(MustParse(test.input))

			printLine := func(err string) { t.Log(err) }
			PrintErrors(errs, "test", printLine)
			clearValidateErrors(errs)

			assert.Equal(t, test.errs, errs)
		})
	}
}

func Benchmark_Validator(b *testing.B) {
	astGenConfig := &AstGenerationConfig{
		maxDefNodes:    1000,
		maxMemberNodes: 75,
		maxStrLength:   25,
		maxDepth:       3,
		maxArrayDim:    3,
		arrayChance:    10,
	}
	for b.Loop() {
		b.StopTimer()
		randomAst := generateAstWithConfig(astGenConfig)
		astString := FmtAst(randomAst)

		// fmt.Printf("%v\n\n", FmtAstWithConfig(randomNodes, &FmtAstConfig{ShouldPrintLines: true}))

		startTime := time.Now()
		b.StartTimer()
		_ = Validate(randomAst)

		b.StopTimer()
		endTime := time.Now()

		lineCount := strings.Count(astString, "\n") + 1
		totalTime := endTime.Sub(startTime)

		fmt.Printf("AstLineCount: %d\nDuration: %v\nLinesPerSecond: %f\n\n",
			lineCount,
			totalTime,
			linesPerUnit(lineCount, totalTime.Seconds()),
		)

		b.StartTimer()
	}
}