package qubits

import (
	"fmt"
	"qsim/utils"
)

type QubitStateManager struct {
	//It's this shit all over again
	Amptitude []complex64

	ModifierID []int32
	Size       int32
	ID         int32
}

func NewQubitStateManagerFrom(amplitudes []complex64, modifierID []int32) *QubitStateManager {
	tmp := QubitStateManager{
		Amptitude:  amplitudes,
		ModifierID: modifierID,
		Size:       int32(len(modifierID)),
	}
	tmp.ID = utils.GenerateID(tmp)
	return &tmp
}

func (c *QubitStateManager) SwapColumn(l, r int32) {
	fmt.Println(c.Amptitude)
	replacement := make([]complex64, len(c.Amptitude))
	for i := range c.Amptitude {
		j := i
		a := (j >> l) & 1
		b := (j >> r) & 1
		j -= (a << l) + (b << r)
		j += (a << r) + (b << l)
		replacement[j] = c.Amptitude[i]
	}
	c.Amptitude = replacement
	fmt.Println(c.Amptitude)
}

func (c *QubitStateManager) MergeWithPrefix(d *QubitStateManager, l, r int32) {
	if c.Size == 0 {
		c.Amptitude = append(c.Amptitude, d.Amptitude...)
		c.ModifierID = append(c.ModifierID, d.ModifierID...)
		c.Size = d.Size
		return
	}
	n := c.Size
	m := d.Size
	replacement := make([]complex64, len(c.Amptitude)*len(d.Amptitude))
	for i := range c.Amptitude {
		for j := range d.Amptitude {
			//l | r | n - l | m - r
			//Artificial Intelligent cant defeat natural stupidity
			idx := ((i >> (n - l)) << (n + m - l)) + ((j >> (m - r)) << (n + m - l - r)) + ((i & ((1 << (n - l)) - 1)) << (m - r)) + (j & ((1 << (m - r)) - 1))
			replacement[idx] = c.Amptitude[i] * d.Amptitude[j]
		}
	}
	replacementModifier := []int32{}
	for i := 0; i < int(l); i++ {
		replacementModifier = append(replacementModifier, c.ModifierID[i])
	}
	for i := 0; i < int(r); i++ {
		replacementModifier = append(replacementModifier, d.ModifierID[i])
	}
	for i := l; i < int32(len(c.ModifierID)); i++ {
		replacementModifier = append(replacementModifier, c.ModifierID[i])
	}
	for i := r; i < int32(len(d.ModifierID)); i++ {
		replacementModifier = append(replacementModifier, d.ModifierID[i])
	}
	c.Size = n + m
	c.Amptitude = replacement
	c.ModifierID = replacementModifier
}

func (c *QubitStateManager) Multiply(val [][]complex64, n int32) {
	result := make([]complex64, 1<<c.Size)
	for i := 0; i < (1 << n); i++ {
		for j := 0; j < (1 << n); j++ {
			for l := 0; l < (1 << (c.Size - n)); l++ {
				fmt.Println(i, j, l, c.Size, n, (j<<(c.Size-n))+l)
				result[(j<<(c.Size-n))+l] += c.Amptitude[i] * val[i][j]
			}
		}
	}
	c.Amptitude = result
}
