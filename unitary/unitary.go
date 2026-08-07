// Package unitary provides unitarity checking and projection for the small
// complex operation matrices used by the universal gate.
package unitary

import (
	"math"
	"math/cmplx"
)

// Tolerance is the default max absolute deviation accepted as "unitary".
const Tolerance = 1e-4

// Error returns the largest |(U†U)[i][j] - δ[i][j]| over all entries.
func Error(op [][]complex64) float64 {
	n := len(op)
	if n == 0 {
		return 0
	}
	dagU := conjTranspose(op)
	prod := mul(dagU, op)
	dev := 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			var want complex64
			if i == j {
				want = 1
			}
			if d := complexAbs(prod[i][j] - want); d > dev {
				dev = d
			}
		}
	}
	return dev
}

// IsUnitary reports whether op is unitary within tol.
func IsUnitary(op [][]complex64, tol float64) bool {
	return Error(op) <= tol
}

// NearestUnitary returns the unitary matrix closest to op in Frobenius norm:
// the polar factor computed by Heron iteration U ← (U + (U†)⁻¹)/2, with a
// Gram-Schmidt orthonormalization fallback when U is (near-)singular.
func NearestUnitary(op [][]complex64) [][]complex64 {
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
		next := make([][]complex64, n)
		for i := 0; i < n; i++ {
			next[i] = make([]complex64, n)
			for j := 0; j < n; j++ {
				next[i][j] = (u[i][j] + invDag[i][j]) / 2
			}
		}
		u = next
		if Error(u) < 1e-10 {
			break
		}
	}
	return u
}

func clone(a [][]complex64) [][]complex64 {
	out := make([][]complex64, len(a))
	for i := range a {
		out[i] = make([]complex64, len(a))
		copy(out[i], a[i])
	}
	return out
}

func mul(a, b [][]complex64) [][]complex64 {
	n := len(a)
	out := make([][]complex64, n)
	for i := 0; i < n; i++ {
		out[i] = make([]complex64, n)
		for j := 0; j < n; j++ {
			var s complex64
			for k := 0; k < n; k++ {
				s += a[i][k] * b[k][j]
			}
			out[i][j] = s
		}
	}
	return out
}

func conjTranspose(a [][]complex64) [][]complex64 {
	n := len(a)
	out := make([][]complex64, n)
	for i := 0; i < n; i++ {
		out[i] = make([]complex64, n)
		for j := 0; j < n; j++ {
			out[i][j] = complex64(cmplx.Conj(complex128(a[j][i])))
		}
	}
	return out
}

// inverse returns the inverse of a via Gauss-Jordan elimination with partial
// pivoting, ok=false when a is (numerically) singular.
func inverse(a [][]complex64) ([][]complex64, bool) {
	n := len(a)
	aug := make([][]complex64, n)
	for i := 0; i < n; i++ {
		aug[i] = make([]complex64, 2*n)
		copy(aug[i], a[i])
		aug[i][n+i] = 1
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
			aug[col][j] /= pivot
		}
		for r := 0; r < n; r++ {
			if r == col {
				continue
			}
			f := aug[r][col]
			if f == 0 {
				continue
			}
			for j := 0; j < 2*n; j++ {
				aug[r][j] -= f * aug[col][j]
			}
		}
	}
	out := make([][]complex64, n)
	for i := 0; i < n; i++ {
		out[i] = make([]complex64, n)
		copy(out[i], aug[i][n:])
	}
	return out, true
}

// gramSchmidt orthonormalizes the columns of a left-to-right (modified
// Gram-Schmidt), substituting an orthogonal basis vector for zero columns.
func gramSchmidt(a [][]complex64) [][]complex64 {
	n := len(a)
	out := make([][]complex64, n)
	for i := 0; i < n; i++ {
		out[i] = make([]complex64, n)
	}
	for col := 0; col < n; col++ {
		v := make([]complex64, n)
		copy(v, a[col])
		orthogonalize(v, out, col)
		if norm2(v) < 1e-12 {
			for s := 0; s < n; s++ {
				w := make([]complex64, n)
				w[s] = 1
				orthogonalize(w, out, col)
				if norm2(w) > 1e-12 {
					v = w
					break
				}
			}
		}
		inv := float32(math.Sqrt(norm2(v)))
		if inv == 0 {
			continue
		}
		scale := complex64(complex(inv, 0))
		for r := 0; r < n; r++ {
			out[r][col] = v[r] / scale
		}
	}
	return out
}

// orthogonalize subtracts the projections of v onto the first k columns of m.
func orthogonalize(v []complex64, m [][]complex64, k int) {
	for j := 0; j < k; j++ {
		var dot complex64
		for r := 0; r < len(v); r++ {
			dot += complex64(cmplx.Conj(complex128(m[r][j]))) * v[r]
		}
		for r := 0; r < len(v); r++ {
			v[r] -= dot * m[r][j]
		}
	}
}

func norm2(v []complex64) float64 {
	s := 0.0
	for _, z := range v {
		s += complexAbs(z) * complexAbs(z)
	}
	return s
}

func complexAbs(z complex64) float64 {
	return math.Hypot(float64(real(z)), float64(imag(z)))
}
