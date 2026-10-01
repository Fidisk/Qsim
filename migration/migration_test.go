package migration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateUnnumberedSave(t *testing.T) {
	old := `{"type":"RenderWindow","id":1,"window":{"name":"c"},"components":[{"type":"QubitsSystem","id":2,"origin":{"size":1,"amplitudes":[{"real":1,"imag":0}],"modifierIDs":[7]}},{"type":"M4Gate","id":3,"hooks":[]}]}`
	out := Migrate(old)

	var w map[string]interface{}
	if err := json.Unmarshal([]byte(out), &w); err != nil {
		t.Fatalf("migrated output is not valid JSON: %v", err)
	}
	if w["saveVersion"] != "0.9.0" {
		t.Fatalf("saveVersion = %v, want 0.9.0", w["saveVersion"])
	}
	if w["showGrid"] != true {
		t.Fatalf("showGrid = %v, want true", w["showGrid"])
	}
	comps := w["components"].([]interface{})
	qs := comps[0].(map[string]interface{})
	if qs["probability"] != float64(1) {
		t.Fatalf("QubitsSystem probability = %v, want 1", qs["probability"])
	}
	if qs["qubitPerm"] == nil {
		t.Fatal("QubitsSystem qubitPerm missing after migration")
	}
	m4 := comps[1].(map[string]interface{})
	if m4["storedProb"] != float64(1) {
		t.Fatalf("M4Gate storedProb = %v, want 1", m4["storedProb"])
	}
}

func TestMigrateCurrentPassesThrough(t *testing.T) {
	cur := `{"type":"RenderWindow","saveVersion":"0.9.0","showGrid":true,"window":{"name":"c"},"components":[{"type":"QubitsSystem","id":2,"probability":0.25,"qubitPerm":[0]}]}`
	if got := Migrate(cur); got != cur {
		t.Fatalf("current-version save was rewritten:\n%s", got)
	}
}

func TestMigrateKeepsMalformedLines(t *testing.T) {
	in := "not json\n"
	if got := Migrate(in); !strings.Contains(got, "not json") {
		t.Fatalf("malformed line lost: %q", got)
	}
}

func TestParseVersionOrder(t *testing.T) {
	if !(parseVersion("0.7.9") < parseVersion("0.8.0")) {
		t.Fatal("0.7.9 should sort below 0.8.0")
	}
	if !(parseVersion("") < parseVersion("0.8.0")) {
		t.Fatal("unnumbered should sort below 0.8.0")
	}
	if !(parseVersion("0.8.0") < parseVersion("0.8.1")) {
		t.Fatal("0.8.0 should sort below 0.8.1")
	}
	if !(parseVersion("0.8.2") < parseVersion("0.9.0")) {
		t.Fatal("0.8.2 should sort below 0.9.0")
	}
}

func sourceSave(t *testing.T, hooked bool) string {
	t.Helper()
	target := `"targetID":14`
	if !hooked {
		target = `"targetID":0`
	}
	return `{"type":"RenderWindow","id":1,"window":{"name":"c"},"components":[{"type":"SourceGate","id":10,"center":{"x":0,"y":0},"radius":90,"color":{"r":0,"g":0,"b":0,"a":255},"label":"S","amplitude":[{"real":8,"imag":0},{"real":6,"imag":0}],"modifierID":2,"outHook":{"type":"Hook","id":11,"center":{"x":0,"y":0},"radius":30,"color":{"r":0,"g":0,"b":0,"a":255},"isFixed":false,"weight":100,"isHooked":` + hookedStr(hooked) + `,` + target + `,"isOutput":true,"label":"O","allowQubitSystem":true,"allowLogicalBit":false}},{"type":"QubitsSystem","id":14,"center":{"x":100,"y":0},"radius":30,"color":{"r":0,"g":0,"b":0,"a":255},"isFixed":false,"weight":0,"hookID":11,"infoHookID":11,"isLogical":false,"origin":{"amplitudes":[{"real":1,"imag":0},{"real":0,"imag":0}],"modifierIDs":[9],"size":1},"qubitDeterminators":[{"center":{"x":150,"y":0},"radius":15,"color":{"r":0,"g":0,"b":0,"a":255},"modifierID":9,"id":15,"hookID":0,"qubitSystemID":14}],"qubitPerm":[0]}]}`
}

func hookedStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func compsByType(t *testing.T, w map[string]interface{}) map[string][]map[string]interface{} {
	t.Helper()
	out := map[string][]map[string]interface{}{}
	comps, ok := w["components"].([]interface{})
	if !ok {
		t.Fatal("components missing")
	}
	for _, c := range comps {
		cm := c.(map[string]interface{})
		typ, _ := cm["type"].(string)
		out[typ] = append(out[typ], cm)
	}
	return out
}

func assertUniqueIDs(t *testing.T, w map[string]interface{}) {
	t.Helper()
	seen := map[float64]string{}
	var walk func(v interface{}, where string)
	walk = func(v interface{}, where string) {
		switch n := v.(type) {
		case map[string]interface{}:
			for k, x := range n {
				if k == "id" {
					if id, ok := x.(float64); ok {
						if prev, dup := seen[id]; dup {
							t.Fatalf("duplicate id %v at %s (first at %s)", id, where, prev)
						}
						seen[id] = where
					}
					continue
				}
				walk(x, where)
			}
		case []interface{}:
			for _, x := range n {
				walk(x, where)
			}
		}
	}
	walk(w, "window")
}

func TestMigrateSourceGateHooked(t *testing.T) {
	var w map[string]interface{}
	if err := json.Unmarshal([]byte(Migrate(sourceSave(t, true))), &w); err != nil {
		t.Fatalf("migrated output is not valid JSON: %v", err)
	}
	byType := compsByType(t, w)
	if len(byType["SourceGate"]) != 0 {
		t.Fatal("SourceGate entry survived migration")
	}
	if len(byType["QubitsSystem"]) != 1 {
		t.Fatalf("QubitsSystem count = %d, want 1", len(byType["QubitsSystem"]))
	}
	assertUniqueIDs(t, w)
	qs := byType["QubitsSystem"][0]
	if qs["hookID"] != float64(0) || qs["infoHookID"] != float64(0) {
		t.Fatalf("system still linked to removed hook: hookID=%v infoHookID=%v", qs["hookID"], qs["infoHookID"])
	}
	origin := qs["origin"].(map[string]interface{})
	amps := origin["amplitudes"].([]interface{})
	if len(amps) != 2 {
		t.Fatalf("amplitudes len = %d, want 2", len(amps))
	}
	// 8/6 input must be normalized to 0.8/0.6 like the running source did.
	a0 := amps[0].(map[string]interface{})
	if a0["real"] != float64(0.8) || a0["imag"] != float64(0) {
		t.Fatalf("amp0 = %v, want {0.8 0}", a0)
	}
	mods := origin["modifierIDs"].([]interface{})
	if len(mods) != 1 || mods[0] != float64(2) {
		t.Fatalf("modifierIDs = %v, want [2]", mods)
	}
	if origin["size"] != float64(1) {
		t.Fatalf("size = %v, want 1", origin["size"])
	}
	dets := qs["qubitDeterminators"].([]interface{})
	if len(dets) != 1 {
		t.Fatalf("determinators len = %d, want 1 untouched", len(dets))
	}
	if dets[0].(map[string]interface{})["modifierID"] != float64(9) {
		t.Fatalf("determinator modifierID changed: %v", dets[0])
	}
}

func TestMigrateSourceGateUnhooked(t *testing.T) {
	var w map[string]interface{}
	if err := json.Unmarshal([]byte(Migrate(sourceSave(t, false))), &w); err != nil {
		t.Fatalf("migrated output is not valid JSON: %v", err)
	}
	byType := compsByType(t, w)
	if len(byType["SourceGate"]) != 0 {
		t.Fatal("SourceGate entry survived migration")
	}
	// Existing system untouched plus one synthesized standalone qubit.
	if len(byType["QubitsSystem"]) != 2 {
		t.Fatalf("QubitsSystem count = %d, want 2", len(byType["QubitsSystem"]))
	}
	var synth map[string]interface{}
	for _, qs := range byType["QubitsSystem"] {
		if qs["id"] != float64(14) {
			synth = qs
		}
	}
	if synth == nil {
		t.Fatal("synthesized system missing")
	}
	// Fixture ids run to 15 (determinator); the synth system must clear them.
	if synth["id"] != float64(16) {
		t.Fatalf("synth id = %v, want 16", synth["id"])
	}
	assertUniqueIDs(t, w)
	origin := synth["origin"].(map[string]interface{})
	amps := origin["amplitudes"].([]interface{})
	a0 := amps[0].(map[string]interface{})
	if a0["real"] != float64(0.8) {
		t.Fatalf("synth amp0 = %v, want normalized 0.8", a0)
	}
	if synth["hookID"] != float64(0) || synth["infoHookID"] != float64(0) {
		t.Fatalf("synth system linked: %v %v", synth["hookID"], synth["infoHookID"])
	}
}

func TestMigrateSourceGateIdempotent(t *testing.T) {
	once := Migrate(sourceSave(t, true))
	if twice := Migrate(once); twice != once {
		t.Fatalf("migration not idempotent:\n%s\n%s", once, twice)
	}
}

func TestMigrateRealSavesDropSources(t *testing.T) {
	for _, name := range []string{"Bug.qsim", "Tele2.qsim", "Telepor.qsim", "Telepor2.qsim"} {
		raw, err := os.ReadFile(filepath.Join("..", "saves", name))
		if err != nil {
			t.Fatalf("read save: %v", err)
		}
		if !strings.Contains(string(raw), `"type":"SourceGate"`) {
			t.Fatalf("%s has no SourceGate to migrate", name)
		}
		out := Migrate(string(raw))
		if strings.Contains(out, `"type":"SourceGate"`) {
			t.Fatalf("%s still contains SourceGate after migration", name)
		}
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var w map[string]interface{}
			if err := json.Unmarshal([]byte(line), &w); err != nil {
				t.Fatalf("%s migrated line is not valid JSON: %v", name, err)
			}
			assertUniqueIDs(t, w)
		}
	}
}

func TestMigrateTelepor2System28(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "saves", "Telepor2.qsim"))
	if err != nil {
		t.Fatalf("read save: %v", err)
	}
	var sys28 map[string]interface{}
	for _, line := range strings.Split(Migrate(string(raw)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var w map[string]interface{}
		if err := json.Unmarshal([]byte(line), &w); err != nil {
			t.Fatal(err)
		}
		for _, c := range w["components"].([]interface{}) {
			cm := c.(map[string]interface{})
			if cm["type"] == "QubitsSystem" && cm["id"] == float64(28) {
				sys28 = cm
			}
		}
	}
	if sys28 == nil {
		t.Fatal("system 28 missing after migration")
	}
	if sys28["hookID"] != float64(0) || sys28["infoHookID"] != float64(0) {
		t.Fatalf("system 28 still linked: %v %v", sys28["hookID"], sys28["infoHookID"])
	}
	origin := sys28["origin"].(map[string]interface{})
	amps := origin["amplitudes"].([]interface{})
	a0 := amps[0].(map[string]interface{})
	if a0["real"] != float64(0.8) || a0["imag"] != float64(0) {
		t.Fatalf("system 28 amp0 = %v, want {0.8 0}", a0)
	}
	mods := origin["modifierIDs"].([]interface{})
	if len(mods) != 1 || mods[0] != float64(2) {
		t.Fatalf("system 28 modifierIDs = %v, want [2]", mods)
	}
}
