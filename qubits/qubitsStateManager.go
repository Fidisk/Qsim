package qubits

type QubitStateManager struct {
	//It's this shit all over again
	Amptitude []float32

	ModifierID []int32
	Size       int32
}

func NewQubitStateManagerFrom(amplitudes []float32, modifierID []int32) *QubitStateManager {
	return &QubitStateManager{
		Amptitude:  amplitudes,
		ModifierID: modifierID,
		Size:       int32(len(modifierID)),
	}
}
