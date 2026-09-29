// Package unitary provides unitarity checking and projection for the small
// complex operation matrices used by the universal gate.
package unitary

import (
	"math"
	"qsim/symbolic"
)

// Tolerance is the default max absolute deviation accepted as "unitary".
const Tolerance = 1e-4

// Error returns the largest |(U†U)[i][j] - δ[i][j]| over all entries.
func Error(op [][]symbolic.SymbolicValue) float64 {
	n := len(op)
	if n == 0 {
		return 0
	}
	dagU := conjTranspose(op)
	prod := mul(dagU, op)
	dev := 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			var want symbolic.SymbolicValue
			if i == j {
				want = symbolic.One()
			}
			if d := complexAbs(prod[i][j].Sub(want)); d > dev {
				dev = d
			}
		}
	}
	return dev
}

// IsUnitary reports whether op is unitary within tol.
func IsUnitary(op [][]symbolic.SymbolicValue, tol float64) bool {
	return Error(op) <= tol
}

// NearestUnitary returns the unitary matrix closest to op in Frobenius norm:
// the polar factor computed by Heron iteration U ← (U + (U†)⁻¹)/2, with a
// Gram-Schmidt orthonormalization fallback when U is (near-)singular.
func NearestUnitary(op [][]symbolic.SymbolicValue) [][]symbolic.SymbolicValue {
	n := len(op)
	if n == 0 {
		return op
	}
	u := clone(op)
	for iter := 0; iter < 64; iter++ {
		inv, ok := inverse(u)
		if !ok {
			return gramSchmidt(u)
		}
		invDag := conjTranspose(inv)
		next := make([][]symbolic.SymbolicValue, n)
		for i := 0; i < n; i++ {
			next[i] = make([]symbolic.SymbolicValue, n)
			for j := 0; j < n; j++ {
				next[i][j] = u[i][j].Add(invDag[i][j]).Scale(0.5)
			}
		}
		u = next
		if Error(u) < 1e-10 {
			break
		}
	}
	return u
}

func clone(a [][]symbolic.SymbolicValue) [][]symbolic.SymbolicValue {
	return symbolic.CloneMatrix(a)
}

func mul(a, b [][]symbolic.SymbolicValue) [][]symbolic.SymbolicValue {
	n := len(a)
	out := make([][]symbolic.SymbolicValue, n)
	for i := 0; i < n; i++ {
		out[i] = make([]symbolic.SymbolicValue, n)
		for j := 0; j < n; j++ {
			var s symbolic.SymbolicValue
			for k := 0; k < n; k++ {
				s = s.Add(a[i][k].Mul(b[k][j]))
			}
			out[i][j] = s
		}
	}
	return out
}

func conjTranspose(a [][]symbolic.SymbolicValue) [][]symbolic.SymbolicValue {
	n := len(a)
	out := make([][]symbolic.SymbolicValue, n)
	for i := 0; i < n; i++ {
		out[i] = make([]symbolic.SymbolicValue, n)
		for j := 0; j < n; j++ {
			out[i][j] = a[j][i].Conj()
		}
	}
	return out
}

// inverse returns the inverse of a via Gauss-Jordan elimination with partial
// pivoting, ok=false when a is (numerically) singular.
func inverse(a [][]symbolic.SymbolicValue) ([][]symbolic.SymbolicValue, bool) {
	n := len(a)
	aug := make([][]symbolic.SymbolicValue, n)
	for i := 0; i < n; i++ {
		aug[i] = make([]symbolic.SymbolicValue, 2*n)
		copy(aug[i], a[i])
		aug[i][n+i] = symbolic.One()
	}
	for col := 0; col < n; col++ {
		best := col
		bestAbs := complexAbs(aug[col][col])
		for r := col + 1; r < n; r++ {
			if ab := complexAbs(aug[r][col]); ab > bestAbs {
				best, bestAbs = r, ab
			}
		}
		if bestAbs < 1e-12 {
			return nil, false
		}
		aug[col], aug[best] = aug[best], aug[col]
		pivot := aug[col][col]
		for j := 0; j < 2*n; j++ {
			aug[col][j] = aug[col][j].Div(pivot)
		}
		for r := 0; r < n; r++ {
			if r == col {
				continue
			}
			f := aug[r][col]
			if f.IsZero() {
				continue
			}
			for j := 0; j < 2*n; j++ {
				aug[r][j] = aug[r][j].Sub(f.Mul(aug[col][j]))
			}
		}
	}
	out := make([][]symbolic.SymbolicValue, n)
	for i := 0; i < n; i++ {
		out[i] = make([]symbolic.SymbolicValue, n)
		copy(out[i], aug[i][n:])
	}
	return out, true
}

// gramSchmidt orthonormalizes the columns of a left-to-right (modified
// Gram-Schmidt), substituting an orthogonal basis vector for zero columns.
func gramSchmidt(a [][]symbolic.SymbolicValue) [][]symbolic.SymbolicValue {
	n := len(a)
	out := make([][]symbolic.SymbolicValue, n)
	for i := 0; i < n; i++ {
		out[i] = make([]symbolic.SymbolicValue, n)
	}
	for col := 0; col < n; col++ {
		v := make([]symbolic.SymbolicValue, n)
		copy(v, a[col])
		orthogonalize(v, out, col)
		if norm2(v) < 1e-12 {
			for s := 0; s < n; s++ {
				w := make([]symbolic.SymbolicValue, n)
				w[s] = symbolic.One()
				orthogonalize(w, out, col)
				if norm2(w) > 1e-12 {
					v = w
					break
				}
			}
		}
		n2 := norm2(v)
		if n2 == 0 {
			continue
		}
		scale := symbolic.New(float32(math.Sqrt(n2)), 0)
		for r := 0; r < n; r++ {
			out[r][col] = v[r].Div(scale)
		}
	}
	return out
}

// orthogonalize subtracts the projections of v onto the first k columns of m.
func orthogonalize(v []symbolic.SymbolicValue, m [][]symbolic.SymbolicValue, k int) {
	for j := 0; j < k; j++ {
		var dot symbolic.SymbolicValue
		for r := 0; r < len(v); r++ {
			dot = dot.Add(m[r][j].Conj().Mul(v[r]))
		}
		for r := 0; r < len(v); r++ {
			v[r] = v[r].Sub(dot.Mul(m[r][j]))
		}
	}
}

func norm2(v []symbolic.SymbolicValue) float64 {
	s := 0.0
	for _, z := range v {
		s += z.AbsSq()
	}
	return s
}

func complexAbs(z symbolic.SymbolicValue) float64 {
	return z.Abs()
}
