package verify

import (
	"encoding/json"
	"fmt"
	"strings"
)

// QASM renders the circuit as OpenQASM 2.0. Standard gates map to qelib1;
// preparations of non-|0>/|1> states and custom unitaries are comments
// naming the sidecar entry, so the file stays parseable and the sidecar
// stays authoritative for the checker script.
func QASM(c *Circuit) string {
	var b strings.Builder
	b.WriteString("OPENQASM 2.0;\n")
	b.WriteString("include \"qelib1.inc\";\n")
	fmt.Fprintf(&b, "// verify-bundle: %s (%d qubits)\n", c.Name, c.NumQubits)
	fmt.Fprintf(&b, "qreg q[%d];\n", c.NumQubits)
	for _, o := range c.Ops {
		switch o.Gate {
		case "x":
			fmt.Fprintf(&b, "x q[%d]; // %s\n", o.Qubits[0], o.Label)
		case "prep":
			fmt.Fprintf(&b, "// verify:prep q[%d] = [%.8g%+.8gi, %.8g%+.8gi] (see sidecar)\n",
				o.Qubits[0], real(o.State[0]), imag(o.State[0]), real(o.State[1]), imag(o.State[1]))
		case "h", "y", "z":
			fmt.Fprintf(&b, "%s q[%d]; // %s\n", o.Gate, o.Qubits[0], o.Label)
		case "cx", "cy", "cz":
			fmt.Fprintf(&b, "%s q[%d],q[%d]; // %s\n", o.Gate, o.Qubits[0], o.Qubits[1], o.Label)
		case "unitary":
			qs := make([]string, len(o.Qubits))
			for i, q := range o.Qubits {
				qs[i] = fmt.Sprintf("q[%d]", q)
			}
			fmt.Fprintf(&b, "// verify:custom U%d on %s (see sidecar)\n", o.Custom, strings.Join(qs, ","))
		}
	}
	return b.String()
}

type sidecarQubit struct {
	QASM     int         `json:"qasm"`
	Modifier int32       `json:"modifier"`
	Init     []cmplxJSON `json:"init"`
}

type cmplxJSON struct {
	Real float64 `json:"real"`
	Imag float64 `json:"imag"`
}

func toJSON(z complex128) cmplxJSON { return cmplxJSON{Real: real(z), Imag: imag(z)} }

func fromJSON(z cmplxJSON) complex128 { return complex(z.Real, z.Imag) }

// opJSON is the wire form of Op; encoding/json has no native complex
// support, so State/Matrix travel as {real,imag} objects.
type opJSON struct {
	Gate   string        `json:"gate"`
	Qubits []int         `json:"qubits"`
	State  []cmplxJSON   `json:"state,omitempty"`
	Matrix [][]cmplxJSON `json:"matrix,omitempty"`
	Label  string        `json:"label"`
	Custom int           `json:"custom,omitempty"`
}

// MarshalJSON encodes State/Matrix as {real,imag} objects.
func (o Op) MarshalJSON() ([]byte, error) {
	aux := opJSON{Gate: o.Gate, Qubits: o.Qubits, Label: o.Label, Custom: o.Custom}
	for _, z := range o.State {
		aux.State = append(aux.State, toJSON(z))
	}
	for _, row := range o.Matrix {
		var r []cmplxJSON
		for _, z := range row {
			r = append(r, toJSON(z))
		}
		aux.Matrix = append(aux.Matrix, r)
	}
	return json.Marshal(aux)
}

// UnmarshalJSON decodes the {real,imag} form back into complex values.
func (o *Op) UnmarshalJSON(data []byte) error {
	var aux opJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	o.Gate, o.Qubits, o.Label, o.Custom = aux.Gate, aux.Qubits, aux.Label, aux.Custom
	o.State = nil
	o.Matrix = nil
	for _, z := range aux.State {
		o.State = append(o.State, fromJSON(z))
	}
	for _, row := range aux.Matrix {
		var r []complex128
		for _, z := range row {
			r = append(r, fromJSON(z))
		}
		o.Matrix = append(o.Matrix, r)
	}
	return nil
}

type sidecar struct {
	Name     string         `json:"name"`
	NQubits  int            `json:"n_qubits"`
	QASM     string         `json:"qasm"`
	Qubits   []sidecarQubit `json:"qubits"`
	Ops      []Op           `json:"ops"`
	Expected []cmplxJSON    `json:"expected"`
	Notes    []string       `json:"notes"`
}

// Sidecar serializes the authoritative checker input: qubit mapping,
// initial states, the full ordered op list with matrices, and Expected.
func Sidecar(c *Circuit) (string, error) {
	s := sidecar{
		Name:    c.Name,
		NQubits: c.NumQubits,
		QASM:    BundleName(c.Name) + ".qasm",
		Notes:   c.Notes,
	}
	if s.Notes == nil {
		s.Notes = []string{}
	}
	for qi, mod := range c.Modifiers {
		s.Qubits = append(s.Qubits, sidecarQubit{
			QASM:     qi,
			Modifier: mod,
			Init:     []cmplxJSON{{Real: real(c.Init[qi][0]), Imag: imag(c.Init[qi][0])}, {Real: real(c.Init[qi][1]), Imag: imag(c.Init[qi][1])}},
		})
	}
	s.Ops = c.Ops
	if s.Ops == nil {
		s.Ops = []Op{}
	}
	for _, a := range c.Expected {
		s.Expected = append(s.Expected, cmplxJSON{Real: float64(real(a)), Imag: float64(imag(a))})
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw) + "\n", nil
}

// BundleName sanitizes a circuit name into a filename stem.
func BundleName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "circuit"
	}
	return b.String()
}

// WriteBundle returns the two verification artifacts (QASM text and checker
// sidecar) keyed by filename. It only formats data; callers decide where
// the files live.
func WriteBundle(c *Circuit) (map[string]string, error) {
	stem := BundleName(c.Name)
	sc, err := Sidecar(c)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		stem + ".qasm":        QASM(c),
		stem + ".verify.json": sc,
	}, nil
}
