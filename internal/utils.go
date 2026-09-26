package internal

import (
	"errors"
	"fmt"
	"os"
)

func clearNodeList(nodes []DefNode) {
	for i := range nodes {
		node := &nodes[i]
		node.ClearPositions()
		for i := range node.Members {
			node := &node.Members[i]
			node.ClearPositions()
			node.DefaultValue.ClearPositions()
			clearTypeNode(&node.LeftType)
			clearTypeNode(&node.RightType)
		}
		clearNodeList(node.LocalDefs)
	}
}

func clearTypeNode(node *TypeNode) {
	node.ClearPositions()
	for i := range node.TypeArgs {
		clearTypeNode(&node.TypeArgs[i])
	}
}

func linesPerUnit(lineCount int, totalTime float64) float64 {
	linesPerUnits := 0.0
	if totalTime > 0 {
		linesPerUnits = float64(lineCount) / totalTime
	}
	return linesPerUnits
}

func loadSampleFile() (string, error) {
	astGenConfig := &AstGenerationConfig{
		maxDefNodes:    1000,
		maxMemberNodes: 75,
		maxStrLength:   25,
		maxDepth:       3,
		maxArrayDim:    3,
		arrayChance:    10,
	}

	const BenchmarkSampleFile = "./samples/benchmark_samples.brpc"

	var astString string

	fileBytes, err := os.ReadFile(BenchmarkSampleFile)
	if errors.Is(err, os.ErrNotExist) {
		randomAst := generateAstWithConfig(astGenConfig)
		astString = FmtAst(randomAst)

		if err := os.WriteFile(BenchmarkSampleFile, []byte(astString), 0444); err != nil {
			return "", fmt.Errorf("Failed to create benchmark sample state file: %v", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("Failed to read sample file data: %v", err)
	} else {
		astString = string(fileBytes)
	}

	return astString, nil
}