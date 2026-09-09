package migration

import (
	"encoding/json"
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
	if w["saveVersion"] != "0.8.1" {
		t.Fatalf("saveVersion = %v, want 0.8.1", w["saveVersion"])
	}
	if w["showGrid"] != true {
		t.Fatalf("showGrid = %v, want true", w["showGrid"])
	}
	comps := w["components"].([]interface{})
	qs := comps[0].(map[string]interface{})
	if qs["probability"] != float64(1) {
		t.Fatalf("QubitsSystem probability = %v, want 1", qs["probability"])
	}
	m4 := comps[1].(map[string]interface{})
	if m4["storedProb"] != float64(1) {
		t.Fatalf("M4Gate storedProb = %v, want 1", m4["storedProb"])
	}
}

func TestMigrateCurrentPassesThrough(t *testing.T) {
	cur := `{"type":"RenderWindow","saveVersion":"0.8.1","showGrid":true,"window":{"name":"c"},"components":[{"type":"QubitsSystem","id":2,"probability":0.25}]}`
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
}
