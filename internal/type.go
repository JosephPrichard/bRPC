package internal

import (
	"fmt"
	"strconv"
	"strings"
)

type Type struct {
	Bits      int    // populated for bit-width integers intead of iden
	Iden      string // populated for non-integer identifiers
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
	t := Type{Iden: iden, Primitive: isPrimitive(iden)}

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

	t = Type{Primitive: true, Bits: bits}
	return t
}

func (t Type) Native() string {
	if t.Iden != "" {
		return t.Iden
	}

	// map to a fix sized primitive, or a big integer if that is not possible
	for _, size := range IntSizes {
		if t.Bits <= size {
			return fmt.Sprintf("int%d", size)
		}
	}
	return "big.Int"
}
