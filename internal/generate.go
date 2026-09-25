package internal

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
)

type AstGenerationConfig struct {
	maxDefNodes    uint64
	maxMemberNodes uint64
	maxStrLength   uint64
	maxDepth       uint64
	maxArrayDim    uint64
	arrayChance    uint64 // denominator, so arrayChance = 10 is 1/10 chance of being an array
}

var DefaultGenerationConfig = AstGenerationConfig{
	maxDefNodes:    50,
	maxMemberNodes: 8,
	maxStrLength:   25,
	maxDepth:       5,
	maxArrayDim:    2,
	arrayChance:    10,
}

type TypeStack struct {
	stack []TypeStackFrame
}

func (t *TypeStack) push() *TypeStackFrame {
	t.stack = append(t.stack, TypeStackFrame{})
	return &t.stack[len(t.stack)-1]
}

func (t *TypeStack) pop() {
	t.stack = t.stack[0 : len(t.stack)-1]
}

func (t *TypeStack) getRandomIden() string {
	if len(t.stack) == 0 {
		return ""
	}
	stackIndex := rand.IntN(len(t.stack))

	stackFrame := t.stack[stackIndex]
	if len(stackFrame.idens) == 0 {
		return ""
	}
	stackFrameIndex := rand.IntN(len(stackFrame.idens))

	return stackFrame.idens[stackFrameIndex]
}

type TypeStackFrame struct {
	idens []string
}

func generateAst() []DefNode {
	return generateAstWithConfig(&DefaultGenerationConfig)
}

func generateAstWithConfig(config *AstGenerationConfig) []DefNode {
	// rootNodes := []DefNode{
	// 	{
	// 		Kind:     PropertyNodeKind,
	// 		StrValue: "test",
	// 	},
	// }
	var rootNodes []DefNode

	defNodes := generateDefList(config, 0, &TypeStack{})

	rootNodes = append(rootNodes, defNodes...)
	return rootNodes
}

var DefKindOptions = [][]NodeKind{
	{StructNodeKind, EnumNodeKind, UnionNodeKind, ServiceNodeKind},
	{StructNodeKind, EnumNodeKind, UnionNodeKind},
}

func getDefKindOptions(depth uint64) []NodeKind {
	if int(depth) >= len(DefKindOptions) {
		depth = uint64(len(DefKindOptions) - 1)
	}
	return DefKindOptions[depth]
}

const idenCharacters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func generateString(config *AstGenerationConfig) string {
	idenLength := rand.N(config.maxStrLength-4) + 4

	var sb strings.Builder

	for range idenLength {
		chIndex := rand.N(len(idenCharacters))
		ch := idenCharacters[chIndex]
		sb.WriteRune(rune(ch))
	}

	return sb.String()
}

func generateSize(kind NodeKind) uint64 {
	switch kind {
	case UnionNodeKind, EnumNodeKind:
		return 8
	default:
		return 0
	}
}

func generateDefList(config *AstGenerationConfig, depth uint64, typeStack *TypeStack) []DefNode {
	if depth >= config.maxDepth {
		return nil
	}

	maxDefNodes := int(math.Pow(float64(config.maxDefNodes), 1/math.Pow(2, float64(depth))))
	nodeCount := rand.N(maxDefNodes)
	defNodes := make([]DefNode, 0)

	typeStackFrame := typeStack.push()

	for range nodeCount {
		kindOptions := getDefKindOptions(depth)
		kind := kindOptions[rand.N(len(kindOptions))]

		localDefs := generateDefList(config, depth+1, typeStack)
		memberNodes := generateMemberList(config, kind, typeStack)

		defNode := DefNode{
			Kind:      kind,
			Iden:      generateString(config),
			Members:   memberNodes,
			LocalDefs: localDefs,
			Size:      generateSize(kind),
		}
		defNodes = append(defNodes, defNode)

		typeStackFrame.idens = append(typeStackFrame.idens, defNode.Iden)
	}

	typeStack.pop()

	return defNodes
}

func generateType(config *AstGenerationConfig, typeStack *TypeStack) TypeNode {
	// Decide to use an array or not, then if so, the number of dimensions of the array, and the
	var array []uint64
	if rand.IntN(int(config.arrayChance)) == 0 {
		arrayDimensionCount := rand.IntN(int(config.maxArrayDim))
		for range arrayDimensionCount {
			array = append(array, 0)
		}
	}
	getPrimitiveTypeNode := func() TypeNode {
		bitSize := rand.N(60) + 4
		return TypeNode{Iden: fmt.Sprintf("int%d", bitSize), Array: array}
	}
	// Decide to use a primitive iden or reference a previously defined iden
	if rand.IntN(2) == 0 {
		return getPrimitiveTypeNode()
	} else {
		iden := typeStack.getRandomIden()
		if iden != "" {
			return TypeNode{Iden: iden, Array: array}
		} else {
			return getPrimitiveTypeNode()
		}
	}
}

func generateDefaultValue(_ *AstGenerationConfig, _ *TypeNode) ValueNode {
	return ValueNode{Kind: NoInstanceKind}
}

func generateMemberList(config *AstGenerationConfig, kind NodeKind, typeStack *TypeStack) []MemberNode {
	nodeCount := rand.N(config.maxMemberNodes)
	memberNodes := make([]MemberNode, 0, nodeCount)
	memberKind := kind.MemberKind()

	for i := range nodeCount {
		node := MemberNode{Tag: i + 1, Iden: generateString(config)}

		switch memberKind {
		case FieldNodeKind:
			node.Modifier = Optional
			node.LeftType = generateType(config, typeStack)
			node.DefaultValue = generateDefaultValue(config, &node.LeftType)
		case OptionNodeKind:
			node.LeftType = generateType(config, typeStack)
		case CaseNodeKind: // do-nothing
		case RpcNodeKind:
			node.LeftType = generateType(config, typeStack)
			node.RightType = generateType(config, typeStack)
		}

		memberNodes = append(memberNodes, node)
	}

	return memberNodes
}
