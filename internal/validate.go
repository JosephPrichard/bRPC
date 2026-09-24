package internal

import (
	"slices"
)

func transformDefList(nodes []DefNode, prev *TypeDefStack, errs *[]error) {
	stack := makeTypeDefStack(prev)
	for i := range nodes {
		node := &nodes[i]
		if !node.Kind.isTypeDecl() {
			continue
		}

		if err := stack.insert(node.Iden, node); err != nil {
			*errs = append(*errs, err)
		}
		node.DefStack = stack

		kind := node.Kind.MemberKind()
		for i := range node.Members {
			node := &node.Members[i]
			switch kind {
			case FieldNodeKind, OptionNodeKind:
				transformType(&node.LType)
			case RpcNodeKind:
				transformType(&node.LType)
				transformType(&node.RType)
			}
		}
		slices.SortFunc(node.Members, func(n1, n2 MembNode) int { return int(n1.Ord - n2.Ord) })

		transformDefList(node.LocalDefs, stack, errs)
	}
}

func transformType(node *TypeNode) {
	node.Value = makeType(node.Iden)
	for i := range node.TypeArgs {
		transformType(&node.TypeArgs[i])
	}
}

func validateDefList(defs []DefNode, errs *[]error) {
	for i := range defs {
		def := &defs[i]
		kind := def.Kind.MemberKind()
		if !def.Kind.isTypeDecl() {
			continue
		}

		expOrd := uint64(1)
		for _, mem := range def.Members {
			ord := mem.Ord
			if ord != expOrd {
				*errs = append(*errs, makeOrdErr(kind, mem.Positions, expOrd, ord))
				break
			}
			expOrd++
		}
		for i, mem := range def.Members {
			idenR := mem.Iden
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
				validateType(kind, mem.LType, def, errs)
			case RpcNodeKind:
				validateType(kind, mem.LType, def, errs)
				validateType(kind, mem.RType, def, errs)
			}
		}

		validateDefList(def.LocalDefs, errs)
	}
}

func validateType(kind NodeKind, node TypeNode, parent *DefNode, errs *[]error) {
	if node.Value.Primitive {
		return
	}
	iden := node.Value.Native()
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
