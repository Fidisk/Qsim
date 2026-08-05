package components

import (
	"math"
	"math/cmplx"
	"testing"
)

// Brute-force check of the remainder extraction used in MeasureOutput:
// for every size and measured position, each amplitude whose measured bit
// matches the outcome must land in the right remainder slot, normalized.
func TestRemainderExtraction(t *testing.T) {
	for size := int32(2); size <= 4; size++ {
		full := 1 << size
		// distinct amplitudes so a wrong index is always visible
		amps := make([]complex64, full)
		var norm float64
		for i := range amps {
			amps[i] = complex(float32(i+1), float32(i%3))
			norm += float64(real(amps[i])*real(amps[i]) + imag(amps[i])*imag(amps[i]))
		}
		inv0 := complex64(complex(1/math.Sqrt(norm), 0))
		for i := range amps {
			amps[i] *= inv0
		}

		for pos := int32(0); pos < size; pos++ {
			for k := int32(0); k <= 1; k++ {
				n := size - 1
				var l float64
				for i, d := range amps {
					if ((int32(i) >> (n - pos)) & 1) == k {
						l += float64(real(d)*real(d) + imag(d)*imag(d))
					}
				}
				if l == 0 {
					continue
				}

				// --- code under test (mirrors MeasureOutput) ---
				shift := size - 1 - pos
				restSize := int32(1) << (size - 1)
				restAmps := make([]complex64, restSize)
				inv := complex64(complex(1/math.Sqrt(l), 0))
				for j := int32(0); j < restSize; j++ {
					high := (j >> shift) << (shift + 1)
					low := j & ((1 << shift) - 1)
					idx := high | (k << shift) | low
					restAmps[j] = amps[idx] * inv
				}

				// --- reference: iterate full indices, delete the measured bit ---
				ref := make([]float64, restSize)
				for i, d := range amps {
					if ((int32(i) >> (n - pos)) & 1) != k {
						continue
					}
					hi := int32(i) >> (shift + 1)
					lo := int32(i) & ((1 << shift) - 1)
					j := (hi << shift) | lo
					ref[j] = cmplx.Abs(complex128(d)) / l * math.Sqrt(l)
				}

				var sumSq float64
				for j := range ref {
					got := cmplx.Abs(complex128(restAmps[j]))
					sumSq += got * got
					if math.Abs(got-ref[j]) > 1e-5 {
						t.Fatalf("size=%d pos=%d k=%d j=%d: |amp|=%v want %v", size, pos, k, j, got, ref[j])
					}
				}
				if math.Abs(sumSq-1) > 1e-4 {
					t.Fatalf("size=%d pos=%d k=%d: remainder not normalized (sumSq=%v)", size, pos, k, sumSq)
				}
			}
		}
	}
}

// Teleportation sanity check: |psi> = 0.8|0> + 0.6|1>, Bell pair, CNOT, H,
// then measure qubit 0 -> the two-qubit remainder must be
// (a(|00>+|11>) + b(|01>+|10>))/sqrt(2) for outcome 0.
func TestTeleportationFirstMeasurement(t *testing.T) {
	a, b := 0.8, 0.6
	// state after CNOT(q0->q1), H(q0): [a,b,b,a, a,-b,-b,a] / 2
	amps := []complex64{
		complex(float32(a/2), 0), complex(float32(b/2), 0), complex(float32(b/2), 0), complex(float32(a/2), 0),
		complex(float32(a/2), 0), complex(float32(-b/2), 0), complex(float32(-b/2), 0), complex(float32(a/2), 0),
	}
	size, pos, k := int32(3), int32(0), int32(0)

	n := size - 1
	var l float64
	for i, d := range amps {
		if ((int32(i) >> (n - pos)) & 1) == k {
			l += float64(real(d) * real(d))
		}
	}
	shift := size - 1 - pos
	restSize := int32(1) << (size - 1)
	restAmps := make([]complex64, restSize)
	inv := complex64(complex(1 / math.Sqrt(l), 0))
	for j := int32(0); j < restSize; j++ {
		high := (j >> shift) << (shift + 1)
		low := j & ((1 << shift) - 1)
		idx := high | (k << shift) | low
		restAmps[j] = amps[idx] * inv
	}

	s := math.Sqrt2
	want := []float64{a / s, b / s, b / s, a / s}
	for j := range want {
		if math.Abs(float64(real(restAmps[j]))-want[j]) > 1e-5 {
			t.Fatalf("remainder = %v, want %v", restAmps, want)
		}
	}
}
