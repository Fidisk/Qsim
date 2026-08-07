package unitary

import (
	"math"
	"math/rand"
	"testing"
)

func approxUnitary(t *testing.T, op [][]complex64) {
	t.Helper()
	if e := Error(op); e > Tolerance {
		t.Fatalf("deviation %g > tolerance %g", e, Tolerance)
	}
}

func diag(n int, v complex64, off complex64) [][]complex64 {
	op := make([][]complex64, n)
	for i := 0; i < n; i++ {
		op[i] = make([]complex64, n)
		for j := 0; j < n; j++ {
			if i == j {
				op[i][j] = v
			} else {
				op[i][j] = off
			}
		}
	}
	return op
}

func TestIdentityError(t *testing.T) {
	if e := Error(diag(4, 1, 0)); e > 1e-6 {
		t.Fatalf("identity deviation %g", e)
	}
}

func TestCloneUnchanged(t *testing.T) {
	// A known unitary must be returned essentially unchanged.
	// (0 1; -1 0) with i: [[i,0],[0,-i]] is unitary (diagonal phase).
	op := [][]complex64{{1, 0}, {0, -1}}
	u := NearestUnitary(op)
	if complexAbs(u[0][0]-1) > 1e-6 || complexAbs(u[1][1]+1) > 1e-6 {
		t.Fatalf("diagonal phase gate altered: %v", u)
	}
}

func TestScaledIdentity(t *testing.T) {
	u := NearestUnitary(diag(2, 2, 0))
	approxUnitary(t, u)
	if complexAbs(u[0][0]-1) > 1e-6 {
		t.Fatalf("nearest unitary to 2I should be I, got %v", u[0])
	}
}

func TestHadamard(t *testing.T) {
	op := [][]complex64{{1, 1}, {1, -1}} // scaled Hadamard, not unitary
	if e := Error(op); e < 0.1 {
		t.Fatalf("scaled Hadamard should be far from unitary, got %g", e)
	}
	u := NearestUnitary(op)
	approxUnitary(t, u)
	// Should look like H/√2: entries +1/√2 except the bottom-right -1/√2.
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			want := 0.70710678
			if i == 1 && j == 1 {
				want = -want
			}
			if d := math.Abs(float64(real(u[i][j])) - want); d > 1e-4 {
				t.Fatalf("entry [%d][%d] = %v, want %g", i, j, u[i][j], want)
			}
			if imag(u[i][j]) != 0 {
				t.Fatalf("Hadamard should be real, got %v", u[i][j])
			}
		}
	}
}

func TestSingular(t *testing.T) {
	op := [][]complex64{
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}
	approxUnitary(t, NearestUnitary(op))
}

func TestRandom(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	for dim := 2; dim <= 16; dim <<= 1 {
		op := make([][]complex64, dim)
		for i := 0; i < dim; i++ {
			op[i] = make([]complex64, dim)
			for j := 0; j < dim; j++ {
				op[i][j] = complex64(complex(r.Float64()*2-1, r.Float64()*2-1))
			}
		}
		approxUnitary(t, NearestUnitary(op))
	}
}

func TestNonUnitaryDetected(t *testing.T) {
	op := [][]complex64{{1.1, 0}, {0, 0.9}}
	if IsUnitary(op, Tolerance) {
		t.Fatal("non-unitary matrix reported as unitary")
	}
	if !IsUnitary(NearestUnitary(op), Tolerance) {
		t.Fatal("projected matrix not unitary")
	}
}
