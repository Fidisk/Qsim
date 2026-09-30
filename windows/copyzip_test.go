package windows

import (
	"os"
	"testing"

	"qsim/components"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// A copy input left unhooked in the save must attach to the nearby qubit
// system on the first frame (justSpawned hook zip) and stay attached while
// frames run. Guards the source-removal migration: migrated standalone
// systems must remain valid copy targets.
func TestCopyInputFirstFrameZip(t *testing.T) {
	far := rl.Vector2{X: -99999, Y: -99999}
	for _, name := range []string{"../saves/Telepor2.qsim", "../saves/Bug.qsim"} {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		var rw *RenderWindow
		for _, o := range LoadState(string(raw)) {
			if w, ok := o.(*RenderWindow); ok {
				rw = w
			}
		}
		if rw == nil {
			t.Fatalf("%s: no RenderWindow loaded", name)
		}
		found := false
		for _, c := range rw.WComp {
			cg, ok := c.(*components.CopyGate)
			if !ok || cg.InHook.IsHooked {
				continue
			}
			found = true
			avail := true
			cg.InHook.Update(far, false, &avail)
			if !cg.InHook.IsHooked {
				t.Fatalf("%s: copy %d input did not zip on first frame", name, cg.ID)
			}
			qs, ok := utils.GetObjectFromID(cg.InHook.TargetID).(*components.QubitsSystem)
			if !ok || qs.Origin == nil {
				t.Fatalf("%s: copy %d input zipped to non-system %T", name, cg.ID, utils.GetObjectFromID(cg.InHook.TargetID))
			}
		}
		if !found {
			t.Fatalf("%s: no unhooked copy input to test", name)
		}
		for frame := 0; frame < 300; frame++ {
			isCursorAvailable := false
			for i := len(rw.WComp) - 1; i >= 0; i-- {
				if i >= len(rw.WComp) {
					continue
				}
				if q, ok := rw.WComp[i].(*components.QubitsSystem); ok {
					q.UpdateDeterminators(far, false, &isCursorAvailable)
				}
			}
			for i := len(rw.WComp) - 1; i >= 0; i-- {
				if i >= len(rw.WComp) {
					continue
				}
				rw.WComp[i].Update(far, false, &isCursorAvailable)
			}
		}
		for _, c := range rw.WComp {
			if cg, ok := c.(*components.CopyGate); ok && !cg.InHook.IsHooked {
				t.Fatalf("%s: copy %d input disconnected during frames", name, cg.ID)
			}
		}
	}
}
