package animation

import (
	"testing"

	"qsim/qubits"
	"qsim/symbolic"
)

// Verify the reorder tracking (PostMods / DigitDest / Perm) against the real
// QubitStateManager swap behavior, using FindID for swap targets exactly like
// Gate.CalculateOutPut. Content of old row r must end at Perm[r] with
// InAmps[Perm[r]] == MergedAmps[r], and PostMods must reflect the physical
// display order (SwapColumn(i, m) swaps display positions i and m).
func TestReorderTracking(t *testing.T) {
	// 3-qubit system, modifier IDs as stored (display order top -> bottom)
	amps := []symbolic.SymbolicValue{symbolic.New(0, 0), symbolic.New(1, 0), symbolic.New(2, 0), symbolic.New(3, 0), symbolic.New(4, 0), symbolic.New(5, 0), symbolic.New(6, 0), symbolic.New(7, 0)}
	mods := []int32{10, 11, 12}
	merged := qubits.NewQubitStateManagerFrom(symbolic.CloneSlice(amps), append([]int32(nil), mods...))

	size := int32(3)
	idx := []int32{12, 10} // gate acts on Q12 then Q10 (n=2)

	post := append([]int32(nil), mods...)
	dest := make([]int32, size)
	at := make([]int32, size)
	for p := range post {
		dest[p] = int32(p)
		at[p] = int32(p)
	}
	perm := make([]int32, 1<<uint(size))
	for s := range perm {
		perm[s] = int32(s)
	}
	swapPos := func(a, b int32) {
		ba := size - 1 - a
		bb := size - 1 - b
		for s := range perm {
			cur := perm[s]
			bita := (cur >> uint(ba)) & 1
			bitb := (cur >> uint(bb)) & 1
			if bita != bitb {
				perm[s] = cur ^ (1 << uint(ba)) ^ (1 << uint(bb))
			}
		}
		post[a], post[b] = post[b], post[a]
		oa, ob := at[a], at[b]
		at[a], at[b] = ob, oa
		dest[oa], dest[ob] = b, a
	}
	for i := int32(0); i < 2; i++ {
		m := merged.FindID(idx[i])
		if m < 0 {
			t.Fatalf("modifier %d not found", idx[i])
		}
		if m != i {
			swapPos(i, m)
		}
		merged.SwapColumn(i, m)
	}

	// gate qubits must be physically on top: pos0 = Q12, pos1 = Q10
	if post[0] != 12 || post[1] != 10 {
		t.Fatalf("gate qubits not on top: %v", post)
	}
	// every amplitude must have traveled to its permuted row
	for r := range perm {
		if merged.Amptitude[perm[r]] != amps[r] {
			t.Fatalf("Perm mismatch at row %d: InAmps[%d]=%v want %v", r, perm[r], merged.Amptitude[perm[r]], amps[r])
		}
	}
	// dest must be consistent with post
	for p := range dest {
		if post[dest[p]] != mods[p] {
			t.Fatalf("DigitDest mismatch for pos %d", p)
		}
	}
}
