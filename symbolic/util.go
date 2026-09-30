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

func (s SymbolicValue) Abs() float64 {
	return math.Hypot(float64(s.Real()), float64(s.Imag()))
}

func (s SymbolicValue) AbsSq() float64 {
	re := float64(s.Real())
	im := float64(s.Imag())
	return re*re + im*im
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
	if sum == 0 || math.IsNaN(sum) {
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
	if c, ok := s.closed(); ok {
		v := complex128(c)
		re := strconv.FormatFloat(real(v), 'g', -1, 32)
		if imag(v) == 0 {
			return re
		}
		return re + "," + strconv.FormatFloat(imag(v), 'g', -1, 32)
	}
	return s.String()
}

func Parse(text string) (SymbolicValue, error) {
	if c, err := parseNumericEntry(text); err == nil {
		return FromComplex64(c), nil
	}
	return parseSymbolicEntry(text)
}

func (s SymbolicValue) Simplify() SymbolicValue {
	return s
}

func parseNumericEntry(s string) (complex64, error) {
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

func startsFactor(c byte) bool {
	return c == '(' || c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.'
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentChar(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func parseSymbolicEntry(s string) (SymbolicValue, error) {
	if strings.TrimSpace(s) == "" {
		return SymbolicValue{}, fmt.Errorf("empty entry")
	}
	e := &entryParser{s: s}
	v, err := e.parseSum()
	if err != nil {
		return SymbolicValue{}, err
	}
	e.skipSpaces()
	if e.peek() != 0 {
		return SymbolicValue{}, fmt.Errorf("bad entry %q", s)
	}
	return v, nil
}

type entryParser struct {
	s string
	p int
}

func (e *entryParser) skipSpaces() {
	for e.p < len(e.s) && (e.s[e.p] == ' ' || e.s[e.p] == '\t') {
		e.p++
	}
}

func (e *entryParser) peek() byte {
	if e.p >= len(e.s) {
		return 0
	}
	return e.s[e.p]
}

func (e *entryParser) parseSum() (SymbolicValue, error) {
	e.skipSpaces()
	neg := false
	if c := e.peek(); c == '+' || c == '-' {
		neg = c == '-'
		e.p++
	}
	v, err := e.parseTerm()
	if err != nil {
		return SymbolicValue{}, err
	}
	if neg {
		v = v.Neg()
	}
	for {
		e.skipSpaces()
		c := e.peek()
		if c != '+' && c != '-' {
			return v, nil
		}
		e.p++
		t, err := e.parseTerm()
		if err != nil {
			return SymbolicValue{}, err
		}
		if c == '-' {
			v = v.Sub(t)
		} else {
			v = v.Add(t)
		}
	}
}

func (e *entryParser) parseTerm() (SymbolicValue, error) {
	f, err := e.parseFactor()
	if err != nil {
		return SymbolicValue{}, err
	}
	for {
		e.skipSpaces()
		if e.peek() == '*' {
			e.p++
			g, err := e.parseFactor()
			if err != nil {
				return SymbolicValue{}, err
			}
			f = f.Mul(g)
			continue
		}
		if e.peek() == '/' {
			e.p++
			g, err := e.parseFactor()
			if err != nil {
				return SymbolicValue{}, err
			}
			f = f.Div(g)
			continue
		}
		if startsFactor(e.peek()) {
			g, err := e.parseFactor()
			if err != nil {
				return SymbolicValue{}, err
			}
			f = f.Mul(g)
			continue
		}
		return f, nil
	}
}

func (e *entryParser) parseFactor() (SymbolicValue, error) {
	e.skipSpaces()
	c := e.peek()
	if c == '+' || c == '-' {
		e.p++
		v, err := e.parseFactor()
		if err != nil {
			return SymbolicValue{}, err
		}
		if c == '-' {
			return v.Neg(), nil
		}
		return v, nil
	}
	if c == '(' {
		e.p++
		v, err := e.parseSum()
		if err != nil {
			return SymbolicValue{}, err
		}
		e.skipSpaces()
		if e.peek() != ')' {
			return SymbolicValue{}, fmt.Errorf("bad entry %q", e.s)
		}
		e.p++
		return v, nil
	}
	if isIdentStart(c) {
		start := e.p
		for e.p < len(e.s) && isIdentChar(e.s[e.p]) {
			e.p++
		}
		name := e.s[start:e.p]
		if name == "i" {
			return New(0, 1), nil
		}
		if name == "conj" {
			e.skipSpaces()
			if e.peek() != '(' {
				return SymbolicValue{}, fmt.Errorf("conj requires (...) in %q", e.s)
			}
			e.p++
			v, err := e.parseSum()
			if err != nil {
				return SymbolicValue{}, err
			}
			e.skipSpaces()
			if e.peek() != ')' {
				return SymbolicValue{}, fmt.Errorf("bad entry %q", e.s)
			}
			e.p++
			return v.Conj(), nil
		}
		if name == "sqrt2" {
			return FromComplex64(complex64(complex(math.Sqrt2, 0))), nil
		}
		if name == "pi" {
			return FromComplex64(complex64(complex(math.Pi, 0))), nil
		}
		return Symbol(name), nil
	}
	if (c >= '0' && c <= '9') || c == '.' {
		start := e.p
		for e.p < len(e.s) && e.s[e.p] >= '0' && e.s[e.p] <= '9' {
			e.p++
		}
		if e.p < len(e.s) && e.s[e.p] == '.' {
			e.p++
			for e.p < len(e.s) && e.s[e.p] >= '0' && e.s[e.p] <= '9' {
				e.p++
			}
		}
		if e.p < len(e.s) && (e.s[e.p] == 'e' || e.s[e.p] == 'E') {
			q := e.p + 1
			if q < len(e.s) && (e.s[q] == '+' || e.s[q] == '-') {
				q++
			}
			r := q
			for r < len(e.s) && e.s[r] >= '0' && e.s[r] <= '9' {
				r++
			}
			if r > q {
				e.p = r
			}
		}
		v, err := strconv.ParseFloat(e.s[start:e.p], 64)
		if err != nil {
			return SymbolicValue{}, fmt.Errorf("bad entry %q", e.s)
		}
		return FromComplex128(complex(v, 0)), nil
	}
	return SymbolicValue{}, fmt.Errorf("bad entry %q", e.s)
}
