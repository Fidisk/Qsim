package animation

import (
	"testing"

	"qsim/components"
	"qsim/symbolic"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestCircuitHashSensitivity(t *testing.T) {
	x := [][]symbolic.SymbolicValue{{symbolic.Zero(), symbolic.One()}, {symbolic.One(), symbolic.Zero()}}
	z := [][]symbolic.SymbolicValue{{symbolic.One(), symbolic.Zero()}, {symbolic.Zero(), symbolic.New(-1, 0)}}
	g := components.NewGate(0, 0, 30, rl.Lime, "X", x, 1)
	g.Editable = true

	base := circuitHash([]components.Component{g})

	// Matrix edit must change the hash.
	g.Operation = z
	edited := circuitHash([]components.Component{g})
	if edited == base {
		t.Fatal("matrix edit not detected")
	}

	// Position changes must NOT change the hash (layout moves are cosmetic).
	g.Center = rl.Vector2{X: 300, Y: -200}
	if h := circuitHash([]components.Component{g}); h != edited {
		t.Fatal("position change should not reset the animation")
	}

	// Wiring changes must change the hash.
	g.HookList[0].IsHooked = true
	g.HookList[0].TargetID = 42
	wired := circuitHash([]components.Component{g})
	if wired == edited {
		t.Fatal("wiring change not detected")
	}

	// Component removal must change the hash.
	if circuitHash(nil) == wired {
		t.Fatal("component removal not detected")
	}

	// Same circuit twice: stable.
	if circuitHash([]components.Component{g}) != wired {
		t.Fatal("hash not stable for identical circuit")
	}
}
