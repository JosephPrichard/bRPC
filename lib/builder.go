package lib

import (
	"io"
	"math/big"
	"strings"
)

type BitState struct {
	off  int
	curr uint8
}

type BitReader struct {
	BitState
	r io.Reader
}

type BitWriter struct {
	BitState
	w io.Writer
}

func (r *BitReader) ReadInt64(n int) int64 {
	var i int64

	return i
}

func (w *BitWriter) WriteInt64(i int64, n int) {

}

func (r *BitReader) ReadBigInt(n int) big.Int {
	var i big.Int

	return i
}

func (w *BitWriter) WriteBigInt(i big.Int, n int) {

}

func (r *BitReader) ReadFloat32() float32 {
	var f float32

	return f
}

func (w *BitWriter) WriteFloat32(f float32) {

}

func (r *BitReader) ReadFloat64() float64 {
	var f float64

	return f
}

func (w *BitWriter) WriteFloat64(f float64) {

}

func (r *BitReader) ReadString() string {
	var sb strings.Builder

	return sb.String()
}

func (w *BitWriter) WriteString(s string) {

}

func (r *BitReader) ReadBool() bool {
	var b bool

	return b
}

func (w *BitWriter) WriteBool(b bool) {

}
