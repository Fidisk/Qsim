package symbolic

import "math"

type SymbolicValue struct {
	v complex64
}

func New(re, im float32) SymbolicValue {
	return SymbolicValue{v: complex(re, im)}
}

func FromComplex64(c complex64) SymbolicValue {
	return SymbolicValue{v: c}
}

func FromComplex128(c complex128) SymbolicValue {
	return SymbolicValue{v: complex64(c)}
}

func (s SymbolicValue) ToComplex64() complex64 {
	return s.v
}

func (s SymbolicValue) ToComplex128() complex128 {
	return complex128(s.v)
}

func (s SymbolicValue) Real() float32 {
	return real(s.v)
}

func (s SymbolicValue) Imag() float32 {
	return imag(s.v)
}

func Zero() SymbolicValue {
	return SymbolicValue{v: 0}
}

func One() SymbolicValue {
	return SymbolicValue{v: 1}
}

func (s SymbolicValue) Add(o SymbolicValue) SymbolicValue {
	return SymbolicValue{v: s.v + o.v}
}

func (s SymbolicValue) Sub(o SymbolicValue) SymbolicValue {
	return SymbolicValue{v: s.v - o.v}
}

func (s SymbolicValue) Mul(o SymbolicValue) SymbolicValue {
	return SymbolicValue{v: s.v * o.v}
}

func (s SymbolicValue) Div(o SymbolicValue) SymbolicValue {
	return SymbolicValue{v: s.v / o.v}
}

func (s SymbolicValue) Neg() SymbolicValue {
	return SymbolicValue{v: -s.v}
}

func (s SymbolicValue) Conj() SymbolicValue {
	return SymbolicValue{v: complex(real(s.v), -imag(s.v))}
}

func (s SymbolicValue) Abs() float64 {
	return math.Hypot(float64(real(s.v)), float64(imag(s.v)))
}

func (s SymbolicValue) AbsSq() float64 {
	re := float64(real(s.v))
	im := float64(imag(s.v))
	return re*re + im*im
}

func (s SymbolicValue) IsZero() bool {
	return s.v == 0
}
