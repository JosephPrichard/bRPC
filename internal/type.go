package internal

import (
	"fmt"
	"strconv"
	"strings"
)

type Type struct {
	TIden     string // transformed iden
	Primitive bool
}

func isPrimitive(iden string) bool {
	switch iden {
	case "string":
		return true
	case "bool":
		return true
	case "float32":
		return true
	case "float64":
		return true
	}
	isInt := len(iden) >= 3 && iden[0:2] == "int"
	return isInt
}

var IntSizes = []int{8, 16, 32, 64}

func makeType(iden string) Type {
	t := Type{TIden: iden, Primitive: isPrimitive(iden)}

	index := strings.Index(iden, "int")
	if index < 0 {
		// does not contain int, so it cannot be a primitive
		return t
	}
	bitsStr := iden[index+3:]
	bits, err := strconv.Atoi(bitsStr)
	if err != nil {
		// contains int, but does not contain a number
		return t
	}

	t = Type{Primitive: true}

	// map to a fix sized primitive, or a big integer if that is not possible
	for _, size := range IntSizes {
		if bits <= size {
			t.TIden = fmt.Sprintf("int%d", size)
			break
		}
	}
	if t.TIden != "" {
		t.TIden = "big.Int"
	}

	return t
}

type TypeTable struct {
	prev *TypeTable
	m    map[string]Node
}

func (t *TypeTable) resolve(iden string) Node {
	table := t
	for table != nil {
		node, ok := table.m[iden]
		if ok {
			return node
		}
		table = table.prev
	}
	return nil
}
