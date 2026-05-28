package qubits

type QubitStateManager struct {
	//It's this shit all over again

	Amptitude      []float32
	Representation []int32 //Oh no, but anyways
}

func NewQubitStateManagerFrom(amplitudes []float32, representation []int32) *QubitStateManager {
	return &QubitStateManager{
		Amptitude:      amplitudes,
		Representation: representation,
	}
}
