package internal

import (
	"slices"
)

func runValidator(nodes []DefNode, validateErrs *[]ValidateErr) {
	transformDefList(nodes, nil, validateErrs)
	validateDefList(nodes, validateErrs)
}

func transformDefList(nodes []DefNode, prev *TypeDefStack, errs *[]ValidateErr) {
	stack := makeTypeDefStack(prev)
	for i := range nodes {
		node := &nodes[i]
		if !node.Kind.isTypeDecl() {
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
		def := &nodes[i]
		kind := def.Kind.MemberKind()
		if !def.Kind.isTypeDecl() {
			continue
		}

		// invariant: def.Members is sorted by tag
		expTag := uint64(1)
		for _, mem := range def.Members {
			tag := mem.Tag
			if tag != expTag {
				*errs = append(*errs, makeTagErr(kind, mem.Positions, expTag, tag))
				break
			}
			expTag++
		}
		for i, mem := range def.Members {
			idenR := mem.Iden
			// note(Joseph): def.Members is expected to be small, so brute force search is acceptable
			for j := i - 1; j >= 0; j-- {
				idenL := def.Members[j].Iden
				if idenR == idenL {
					*errs = append(*errs, makeRedefErr(kind, mem.Positions, idenR))
				}
			}
		}

		if def.Kind == EnumNodeKind {
			continue
		}

		for _, mem := range def.Members {
			switch kind {
			case FieldNodeKind, OptionNodeKind:
				validateType(kind, mem.LeftType, def, errs)
			case RpcNodeKind:
				validateType(kind, mem.LeftType, def, errs)
				validateType(kind, mem.RightType, def, errs)
			}
		}

		validateDefList(def.LocalDefs, errs)
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

