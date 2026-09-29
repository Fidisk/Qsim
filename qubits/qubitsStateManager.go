package qubits

import (
	"qsim/qubits/attributes"
	"qsim/symbolic"
	"qsim/utils"
)

type QubitStateManager struct {
	//It's this shit all over again
	Amptitude []symbolic.SymbolicValue

	ModifierID []int32
	Size       int32
	ID         int32
}

func NewQubitStateManager(amplitudes []symbolic.SymbolicValue, size int32) *QubitStateManager {
	id := []int32{}
	for _ = range size {
		id = append(id, attributes.GenerateQubitModifierID())
	}
	tmp := QubitStateManager{
		Amptitude:  amplitudes,
		ModifierID: id,
		Size:       size,
	}
	tmp.ID = utils.GenerateID(tmp)
	return &tmp
}

func NewQubitStateManagerFrom(amplitudes []symbolic.SymbolicValue, modifierID []int32) *QubitStateManager {
	tmp := QubitStateManager{
		Amptitude:  amplitudes,
		ModifierID: modifierID,
		Size:       int32(len(modifierID)),
	}
	tmp.ID = utils.GenerateID(tmp)
	return &tmp
}

func (c *QubitStateManager) SwapColumn(l, r int32) {
	l = c.Size - l - 1
	r = c.Size - r - 1
	replacement := make([]symbolic.SymbolicValue, len(c.Amptitude))
	for i := range c.Amptitude {
		j := i
		a := (j >> l) & 1
		b := (j >> r) & 1
		j -= (a << l) + (b << r)
		j += (a << r) + (b << l)
		replacement[j] = c.Amptitude[i]
	}
	c.Amptitude = replacement
	c.ModifierID[l], c.ModifierID[r] = c.ModifierID[r], c.ModifierID[l]
}

func (c *QubitStateManager) Merge(d *QubitStateManager) {
	if c.Size == 0 {
		c.Amptitude = append(c.Amptitude, d.Amptitude...)
		c.ModifierID = append(c.ModifierID, d.ModifierID...)
		c.Size = d.Size
		return
	}
	n := c.Size
	m := d.Size
	replacement := make([]symbolic.SymbolicValue, len(c.Amptitude)*len(d.Amptitude))
	for i := range c.Amptitude {
		for j := range d.Amptitude {
			idx := (i << m) + j
			replacement[idx] = c.Amptitude[i].Mul(d.Amptitude[j])
		}
	}
	replacementModifier := []int32{}

	for i := 0; i < len(c.ModifierID); i++ {
		replacementModifier = append(replacementModifier, c.ModifierID[i])
	}
	for i := 0; i < len(d.ModifierID); i++ {
		replacementModifier = append(replacementModifier, d.ModifierID[i])
	}
	c.Size = n + m
	c.Amptitude = replacement
	c.ModifierID = replacementModifier
}

func (c *QubitStateManager) FindID(x int32) int32 {
	for i, d := range c.ModifierID {
		if d == x {
			return int32(i)
		}
	}
	//AHHHHHHHHHHHHH
	return -1
}

func (c *QubitStateManager) Multiply(val [][]symbolic.SymbolicValue, n int32) {
	result := make([]symbolic.SymbolicValue, 1<<c.Size)
	for i := 0; i < (1 << n); i++ {
		for j := 0; j < (1 << n); j++ {
			for l := 0; l < (1 << (c.Size - n)); l++ {
				result[(j<<(c.Size-n))+l] = result[(j<<(c.Size-n))+l].Add(c.Amptitude[(i<<(c.Size-n))+l].Mul(val[j][i]))
			}
		}
	}
	c.Amptitude = result
}

func (c *QubitStateManager) SetAmplitudes(amps []symbolic.SymbolicValue, ids []int32) {
	c.Amptitude = amps
	c.ModifierID = ids
	c.Size = int32(len(ids))
}

func (c *QubitStateManager) CopyFrom(other *QubitStateManager) {
	c.Amptitude = make([]symbolic.SymbolicValue, len(other.Amptitude))
	copy(c.Amptitude, other.Amptitude)
	c.ModifierID = make([]int32, len(other.ModifierID))
	copy(c.ModifierID, other.ModifierID)
	c.Size = other.Size
}

// Normalize rescales amplitudes in place so the probabilities (squares of the
// magnitudes) sum to 1. This is the rank-1 case of the SVD "force singular
// values to 1" trick: treated as an n×1 matrix, v = U Σ V† has a single
// singular value ‖v‖; setting it to 1 yields v/‖v‖. A zero vector is left
// untouched.
func Normalize(amplitudes []symbolic.SymbolicValue) {
	symbolic.Normalize(amplitudes)
}
