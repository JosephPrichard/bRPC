package internal

import (
	"fmt"
	"strconv"
	"strings"
)

// RType is an individual type that contains all information for the codegen stage
type RType struct {
	Iden      string
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

func makeRType(iden string) RType {
	t := RType{Iden: iden, Primitive: isPrimitive(iden)}

	index := strings.Index(iden, "int")
	if index < 0 {
		// does not contain int, so it cannot be a primitive
		return t
	}
	bitsStr := iden[index+2:]
	bits, err := strconv.Atoi(bitsStr)
	if err != nil {
		// contains int, but does not contain a number
		return t
	}

	t = RType{Primitive: true}

	// map to a fix sized primitive, or a big integer if that is not possible
	for _, size := range IntSizes {
		if bits <= size {
			t.Iden = fmt.Sprintf("int%d", size)
			break
		}
	}
	if t.Iden != "" {
		t.Iden = "big.Int"
	}

	return t
}
