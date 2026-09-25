// Command bench measures Qsim engine costs headless for performance
// reporting: gate-application latency by gate width, full-pipeline replay
// time vs qubit count, and state memory. Run with:
//
//	go run ./cmd/bench
//
// Methodology: the GUI recomputes gate outputs every frame with the same
// Merge/SwapColumn/Multiply primitives timed here, so the pipeline column
// is a lower bound on per-frame engine cost (rendering not included).
// Frame-rate ceilings below assume the whole 16.7 ms (60 fps) or 33.3 ms
// (30 fps) budget is available to the engine.
package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"runtime"
	"time"

	"qsim/qubits"
)

func main() {
	fmt.Printf("# Qsim engine benchmark\n")
	fmt.Printf("# go=%s os=%s arch=%s cpus=%d\n", runtime.Version(),
		runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	fmt.Printf("# cpu: 11th Gen Intel Core i7-1165G7 @ 2.80GHz, 16GB RAM\n")
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Printf("# mem: sys=%.1fMB alloc=%.1fMB\n",
		float64(mem.Sys)/1e6, float64(mem.Alloc)/1e6)
	fmt.Println("# times are best-of-N ms/op; complex64 = 8 B/amp")

	gateLatency()
	pipeline()
}

func timeIt(reps int, f func()) time.Duration {
	f() // warmup
	best := time.Duration(math.MaxInt64)
	for i := 0; i < reps; i++ {
		t0 := time.Now()
		f()
		if d := time.Since(t0); d < best {
			best = d
		}
	}
	return best
}

func randState(n int32) *qubits.QubitStateManager {
	size := 1 << n
	amps := make([]complex64, size)
	var norm float64
	for i := range amps {
		re := rand.Float32()*2 - 1
		im := rand.Float32()*2 - 1
		amps[i] = complex(re, im)
		norm += float64(re*re + im*im)
	}
	inv := complex(float32(1/math.Sqrt(norm)), 0)
	for i := range amps {
		amps[i] *= inv
	}
	mods := make([]int32, n)
	for i := range mods {
		mods[i] = int32(1000 + i)
	}
	return qubits.NewQubitStateManagerFrom(amps, mods)
}

func kronIdentity(k int32) [][]complex64 {
	// k-qubit "busy" unitary: dense Hadamard-like matrix so Multiply cannot
	// shortcut on sparsity (worst case, like universal gates).
	sz := 1 << k
	t := complex(float32(1/math.Sqrt(float64(sz))), 0)
	m := make([][]complex64, sz)
	for i := range m {
		m[i] = make([]complex64, sz)
		for j := range m[i] {
			m[i][j] = t
			if (i*j)%2 == 1 {
				m[i][j] = -t
			}
		}
	}
	return m
}

// gateLatency times one Multiply of a k-qubit dense gate on an n-qubit
// state (target qubits pre-swapped to front, as the engine does).
func gateLatency() {
	fmt.Println("\n## gate latency: dense k-qubit Multiply on n-qubit state (ms, best of 20; 5 for n>=12)")
	fmt.Println("n_state | mem | k=1 | k=2 | k=3 | k=4")
	for _, n := range []int32{4, 6, 8, 10, 12, 14, 16} {
		memMB := float64(int64(1)<<n) * 8 / 1e6
		row := fmt.Sprintf("%7d | %5.1fMB", n, memMB)
		for _, k := range []int32{1, 2, 3, 4} {
			if k > n {
				row += " |    -"
				continue
			}
			op := kronIdentity(k)
			reps := 20
			if n >= 12 {
				reps = 5
			}
			st := randState(n)
			d := timeIt(reps, func() {
				cp := qubits.NewQubitStateManagerFrom(
					append([]complex64{}, st.Amptitude...),
					append([]int32{}, st.ModifierID...))
				cp.Multiply(op, k)
			})
			row += fmt.Sprintf(" | %5.2f", float64(d.Microseconds())/1000)
		}
		fmt.Println(row)
	}
}

// pipeline times an engine-like frame of work on n qubits: merge n
// singles, one H layer (swap+multiply+swap per qubit, as
// ControlledGate.updateOutput does), and one CX ladder on pairs.
func pipeline() {
	fmt.Println("\n## pipeline: merge + H layer + CX ladder (ms, best of 5)")
	fmt.Println("n | state_mem | merge | H-layer | CX-ladder | total | ~fps@engine-only")
	sqrt2 := complex(float32(1/math.Sqrt2), 0)
	had := [][]complex64{{sqrt2, sqrt2}, {sqrt2, -sqrt2}}
	cnot := [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 0}}
	for _, n := range []int32{2, 4, 6, 8, 10, 12, 14, 16, 18, 20} {
		memMB := float64(int64(1)<<n) * 8 / 1e6
		mkSingles := func() []*qubits.QubitStateManager {
			ss := make([]*qubits.QubitStateManager, n)
			for i := range ss {
				ss[i] = qubits.NewQubitStateManagerFrom(
					[]complex64{1, 0}, []int32{int32(1000 + i)})
			}
			return ss
		}
		dMerge := timeIt(5, func() {
			res := qubits.NewQubitStateManagerFrom([]complex64{}, []int32{})
			for _, s := range mkSingles() {
				res.Merge(s)
			}
		})
		// H layer on a merged state, engine-style per qubit.
		base := qubits.NewQubitStateManagerFrom([]complex64{}, []int32{})
		for _, s := range mkSingles() {
			base.Merge(s)
		}
		dH := timeIt(5, func() {
			st := qubits.NewQubitStateManagerFrom(
				append([]complex64{}, base.Amptitude...),
				append([]int32{}, base.ModifierID...))
			for q := int32(0); q < n; q++ {
				pos := st.FindID(1000 + q)
				st.SwapColumn(0, pos)
				st.Multiply(had, 1)
				st.SwapColumn(0, pos)
			}
		})
		// CX ladder on adjacent pairs, engine-style (gate qubits to front).
		dCX := timeIt(5, func() {
			st := qubits.NewQubitStateManagerFrom(
				append([]complex64{}, base.Amptitude...),
				append([]int32{}, base.ModifierID...))
			for q := int32(0); q+1 < n; q += 2 {
				for i, mod := range []int32{1000 + q, 1000 + q + 1} {
					st.SwapColumn(int32(i), st.FindID(mod))
				}
				st.Multiply(cnot, 2)
			}
		})
		total := dMerge + dH + dCX
		ms := float64(total.Microseconds()) / 1000
		fps := 1000 / ms
		fpsStr := fmt.Sprintf("%.0f", fps)
		if fps > 10000 {
			fpsStr = ">10000"
		}
		fmt.Printf("%2d | %8.1fMB | %6.2f | %7.2f | %8.2f | %6.2f | %s\n",
			n, memMB, msOf(dMerge), msOf(dH), msOf(dCX), ms, fpsStr)
		if total > 10*time.Second {
			fmt.Println("# stopping: single pipeline over 10 s")
			break
		}
	}
}

func msOf(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }
