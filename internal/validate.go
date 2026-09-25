package internal

import (
	"slices"
	"unicode"
)

func runValidator(nodes []DefNode, validateErrs *[]ValidateErr) {
	transformDefList(nodes, nil, validateErrs)
	validateDefList(nodes, validateErrs)
}

func transformDefList(nodes []DefNode, prev *TypeDefStack, errs *[]ValidateErr) {
	stack := makeTypeDefStack(prev)
	for i := range nodes {
		node := &nodes[i]
		if !node.Kind.isTypeDef() {
			continue
		}

		if err := stack.insert(node.Iden, node); err.isPresent() {
			*errs = append(*errs, err)
		}
		node.DefStack = stack

		kind := node.Kind.MemberKind()
		for i := range node.Members {
			node := &node.Members[i]
			switch kind {
			case FieldNodeKind, OptionNodeKind:
				transformType(&node.LeftType)
			case RpcNodeKind:
				transformType(&node.LeftType)
				transformType(&node.RightType)
			}
		}
		slices.SortFunc(node.Members, func(n1, n2 MemberNode) int { return int(n1.Tag - n2.Tag) })

		transformDefList(node.LocalDefs, stack, errs)
	}
}

func transformType(node *TypeNode) {
	node.TypeValue = makeTypeValue(node.Iden)
	for i := range node.TypeArgs {
		transformType(&node.TypeArgs[i])
	}
}

func validateDefList(nodes []DefNode, errs *[]ValidateErr) {
	for i := range nodes {
		defNode := &nodes[i]
		memberKind := defNode.Kind.MemberKind()
		if !defNode.Kind.isTypeDef() {
			continue
		}

		nameOk := validateMessageName(defNode.Iden)
		if !nameOk {
			*errs = append(*errs, makeNameErr(defNode.Kind, defNode.Positions, defNode.Iden))
		}

		// invariant: def.Members is sorted by tag
		expTag := uint64(1)
		for _, memberNode := range defNode.Members {
			tag := memberNode.Tag
			if tag != expTag {
				*errs = append(*errs, makeTagErr(memberKind, memberNode.Positions, expTag, tag))
				break
			}
			expTag++
		}
		for i, mem := range defNode.Members {
			rightIden := mem.Iden
			// note(Joseph): def.Members is expected to be small, so brute force search is acceptable
			for j := i - 1; j >= 0; j-- {
				leftIden := defNode.Members[j].Iden
				if rightIden == leftIden {
					*errs = append(*errs, makeRedefErr(memberKind, mem.Positions, rightIden))
				}
			}
		}

		if defNode.Kind == EnumNodeKind {
			continue
		}

		for _, mem := range defNode.Members {
			switch memberKind {
			case FieldNodeKind, OptionNodeKind:
				validateType(memberKind, mem.LeftType, defNode, errs)
			case RpcNodeKind:
				validateType(memberKind, mem.LeftType, defNode, errs)
				validateType(memberKind, mem.RightType, defNode, errs)
			}
		}

		validateDefList(defNode.LocalDefs, errs)
	}
}

func validateType(kind NodeKind, node TypeNode, parent *DefNode, errs *[]ValidateErr) {
	if node.TypeValue.Primitive {
		return
	}
	iden := node.TypeValue.Native()
	def := resolveIden(parent, iden)
	if def == nil {
		*errs = append(*errs, makeUndefErr(kind, node.Positions, iden))
		return
	}
	if len(node.TypeArgs) != len(def.TypeParams) {
		*errs = append(*errs, makeTypeArgErr(kind, node.Positions, def.TypeParams, node.TypeArgs))
		return
	}
	for _, arg := range node.TypeArgs {
		validateType(kind, arg, parent, errs)
	}
}

func validateMessageName(name string) bool {
	for i, c := range name {
		if i == 0 && unicode.IsLower(c) {
			return false
		}
		if !unicode.IsLetter(c) && !unicode.IsNumber(c) {
			return false
		}
	}
	return true
}