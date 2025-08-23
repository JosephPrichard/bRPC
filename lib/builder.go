package lib

import (
	"math/big"
	"strings"
)

type Builder struct {
	buf []byte
}

func (b *Builder) ReadInt64(off int, n int) int64 {
	var i int64

	return i
}

func (b *Builder) WriteInt64(off int, i int64, n int) {

}

func (b *Builder) ReadBigInt(off int, n int) big.Int {
	var i big.Int

	return i
}

func (b *Builder) WriteBigInt(off int, i big.Int, n int) {

}

func (b *Builder) ReadFloat32(off int) float32 {
	var f float32

	return f
}

func (b *Builder) WriteFloat32(off int, f float32) {

}

func (b *Builder) ReadFloat64(off int) float64 {
	var f float64

	return f
}

func (b *Builder) WriteFloat64(off int, f float64) {

}

func (b *Builder) ReadString(off int) string {
	var sb strings.Builder

	return sb.String()
}

func (b *Builder) WriteString(off int, s string) {

}

func (b *Builder) ReadBool(off int) bool {
	var t bool

	return t
}

func (b *Builder) WriteBool(off int, t bool) {

}
