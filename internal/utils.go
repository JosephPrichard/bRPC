package internal

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