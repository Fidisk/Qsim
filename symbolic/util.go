package symbolic

import (
	"fmt"
	"math"
	"math/cmplx"
	"strconv"
	"strings"
)

func (s SymbolicValue) ReIm() (float32, float32) {
	return s.Real(), s.Imag()
}

func I() SymbolicValue {
	return New(0, 1)
}

func (s SymbolicValue) ScaleReal(f float32) SymbolicValue {
	return s.Mul(New(f, 0))
}

func (s SymbolicValue) Scale(f float64) SymbolicValue {
	return s.Mul(FromComplex64(complex64(complex(f, 0))))
}

func (s SymbolicValue) Prob() float64 {
	return s.AbsSq()
}

func (s SymbolicValue) IsOne() bool {
	return s == One()
}

func (s SymbolicValue) EqualTol(o SymbolicValue, tol float64) bool {
	return s.Sub(o).Abs() <= tol
}

func InvNormFactor(sumSquares float64) SymbolicValue {
	return New(float32(1/math.Sqrt(sumSquares)), 0)
}

func InvSqrtProb(p float64) SymbolicValue {
	return FromComplex128(cmplx.Sqrt(complex(1/p, 0)))
}

func ProbOf(amps []SymbolicValue) float64 {
	sum := 0.0
	for _, a := range amps {
		sum += a.AbsSq()
	}
	return sum
}

func Normalize(amps []SymbolicValue) {
	sum := ProbOf(amps)
	if sum == 0 {
		return
	}
	inv := InvNormFactor(sum)
	for i := range amps {
		amps[i] = amps[i].Mul(inv)
	}
}

func ZeroSlice(n int) []SymbolicValue {
	return make([]SymbolicValue, n)
}

func CloneSlice(a []SymbolicValue) []SymbolicValue {
	out := make([]SymbolicValue, len(a))
	copy(out, a)
	return out
}

func ZeroMatrix(n int) [][]SymbolicValue {
	out := make([][]SymbolicValue, n)
	for i := range out {
		out[i] = make([]SymbolicValue, n)
	}
	return out
}

func CloneMatrix(a [][]SymbolicValue) [][]SymbolicValue {
	out := make([][]SymbolicValue, len(a))
	for i := range a {
		out[i] = append([]SymbolicValue{}, a[i]...)
	}
	return out
}

func (s SymbolicValue) Format() string {
	re := strconv.FormatFloat(float64(s.Real()), 'g', -1, 32)
	if s.Imag() == 0 {
		return re
	}
	return re + "," + strconv.FormatFloat(float64(s.Imag()), 'g', -1, 32)
}

func (s SymbolicValue) String() string {
	return fmt.Sprintf("%.2f%+.2fi", float64(s.Real()), float64(s.Imag()))
}

func Parse(text string) (SymbolicValue, error) {
	c, err := parseMatrixEntry(text)
	if err != nil {
		return SymbolicValue{}, err
	}
	return FromComplex64(c), nil
}

func (s SymbolicValue) Expr() string {
	return s.Format()
}

func (s SymbolicValue) Simplify() SymbolicValue {
	return s
}

func parseMatrixEntry(s string) (complex64, error) {
	t := strings.TrimSpace(s)
	if t == "" {
		return 0, fmt.Errorf("empty entry")
	}

	if strings.Contains(t, ",") {
		parts := strings.Split(t, ",")
		if len(parts) != 2 {
			return 0, fmt.Errorf("bad entry %q (want re, re,im or a+bi)", s)
		}
		re, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return 0, fmt.Errorf("bad entry %q", s)
		}
		im, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return 0, fmt.Errorf("bad entry %q", s)
		}
		return complex64(complex(re, im)), nil
	}

	hasImag := strings.HasSuffix(t, "i")
	body := t
	if hasImag {
		body = t[:len(t)-1]
		switch body {
		case "":
			return complex64(1i), nil
		case "+":
			return complex64(1i), nil
		case "-":
			return complex64(-1i), nil
		}
		split := -1
		for i := len(body) - 1; i > 0; i-- {
			c := body[i]
			if (c == '+' || c == '-') && body[i-1] != 'e' && body[i-1] != 'E' {
				split = i
				break
			}
		}
		if split == -1 {
			im, err := strconv.ParseFloat(body, 64)
			if err != nil {
				return 0, fmt.Errorf("bad entry %q", s)
			}
			return complex64(complex(0, im)), nil
		}
		re, err := strconv.ParseFloat(body[:split], 64)
		if err != nil {
			return 0, fmt.Errorf("bad entry %q", s)
		}
		if split == len(body)-1 {
			im := float64(1)
			if body[split] == '-' {
				im = -1
			}
			return complex64(complex(re, im)), nil
		}
		im, err := strconv.ParseFloat(body[split:], 64)
		if err != nil {
			return 0, fmt.Errorf("bad entry %q", s)
		}
		return complex64(complex(re, im)), nil
	}

	re, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return 0, fmt.Errorf("bad entry %q", s)
	}
	return complex64(complex(re, 0)), nil
}
