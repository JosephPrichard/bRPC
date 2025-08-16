package internal

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"testing"
)

func TestWalk(t *testing.T) {
	asts := []Ast{
		&PropertyAst{Name: "key", Value: "value"},
		&ImportAst{Path: "test"},
		&StructAst{
			Name: "Data1",
			Fields: []FieldAst{
				{Modifier: Required, Name: "one", Ord: 1, Type: &TypeRefAst{Iden: "b1"}},
			},
			LocalDefs: []Ast{
				&UnionAst{
					Name: "Data2",
					Options: []OptionAst{
						{Ord: 1, Type: &TypeArrAst{Type: &TypeRefAst{Iden: "b2"}, Size: []uint64{0, 5}}},
					},
				},
			},
		},
		&ServiceAst{
			Name: "ServiceA",
			Procedures: []RpcAst{
				{Ord: 1, Name: "Hello", Arg: &TypeRefAst{Iden: "Input"}, Ret: &TypeRefAst{Iden: "Output"}},
			},
			LocalDefs: []Ast{&EnumAst{Name: "One"}},
		},
	}

	var keys []string

	visit := func(ast Ast) {
		var sb strings.Builder
		switch ast := ast.(type) {
		case *ImportAst:
			sb.WriteString(ast.Path)
		case *PropertyAst:
			sb.WriteString(ast.Name + "+" + ast.Value)
		case *StructAst:
			sb.WriteString(ast.Name)
		case *UnionAst:
			sb.WriteString(ast.Name)
		case *EnumAst:
			sb.WriteString(ast.Name)
		case *ServiceAst:
			sb.WriteString(ast.Name)
		case *OptionAst:
			sb.WriteString(strconv.Itoa(int(ast.Ord)))
		case *FieldAst:
			sb.WriteString(ast.Name + "+" + strconv.Itoa(int(ast.Ord)))
		case *RpcAst:
			sb.WriteString(ast.Name + "+" + strconv.Itoa(int(ast.Ord)))
		case *TypeRefAst:
			sb.WriteString(ast.Alias + ast.Iden)
		case *TypeArrAst:
			for _, s := range ast.Size {
				sb.WriteString(strconv.Itoa(int(s)))
				sb.WriteByte(',')
			}
		}
		key := sb.String()
		if key != "" {
			keys = append(keys, key)
		}
	}
	WalkList(visit, asts)

	expectedKeys := []string{
		"key+value",
		"test",
		"Data1",
		"Data2",
		"1",
		"0,5,",
		"b2",
		"one+1",
		"b1",
		"ServiceA",
		"One",
		"Hello+1",
		"Input",
		"Output",
	}

	assert.Equal(t, expectedKeys, keys)
}
