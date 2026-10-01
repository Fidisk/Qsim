// Package migration upgrades .qsim save files between format versions.
//
// Save files carry the format version in the "saveVersion" key of every
// window object (see globals.SaveVersion). On load, lines with a missing
// (unnumbered) or lower version are rewritten through every migration step
// up to the current version, in order. Each step's Apply receives the raw
// window object (map[string]interface{}) and may normalize the fields the
// newer version expects.
//
// To add a new format version: bump globals.SaveVersion, append a Step here
// whose To is the new version, and keep the Apply idempotent (old saves may
// already contain some of the fields it sets).
package migration

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"qsim/globals"
)

// CurrentVersion is the version every save is stamped with (and migrated
// toward on load).
const CurrentVersion = globals.SaveVersion

// Step migrates a save from the previous version to To.
type Step struct {
	To    string
	Apply func(w map[string]interface{})
}

// steps is ordered by target version; Migrate applies every step whose
// target is above the file's current version.
var steps = []Step{
	{
		To: "0.8.0",
		Apply: func(w map[string]interface{}) {
			if w["type"] != "RenderWindow" {
				return
			}
			w["saveVersion"] = "0.8.0"
			if w["showGrid"] == nil {
				w["showGrid"] = true
			}
			if comps, ok := w["components"].([]interface{}); ok {
				for _, c := range comps {
					cm, ok := c.(map[string]interface{})
					if !ok {
						continue
					}
					switch cm["type"] {
					case "QubitsSystem":
						if cm["probability"] == nil {
							cm["probability"] = float64(1)
						}
					case "M4Gate":
						if cm["storedProb"] == nil {
							cm["storedProb"] = float64(1)
						}
					}
				}
			}
		},
	},
	{
		To: "0.8.1",
		Apply: func(w map[string]interface{}) {
			if w["type"] != "RenderWindow" {
				return
			}
			w["saveVersion"] = "0.8.1"
			if comps, ok := w["components"].([]interface{}); ok {
				for _, c := range comps {
					cm, ok := c.(map[string]interface{})
					if !ok {
						continue
					}
					if cm["type"] != "CollapseGate" && cm["type"] != "M4Gate" {
						continue
					}
					if v, ok := cm["forceMode"]; ok {
						fm := int32(v.(float64))
						if fm == 0 {
							cm["forceMode"] = float64(1)
						}
					}
				}
			}
		},
	},
	{
		To: "0.8.2",
		Apply: func(w map[string]interface{}) {
			if w["type"] != "RenderWindow" {
				return
			}
			w["saveVersion"] = "0.8.2"
			if comps, ok := w["components"].([]interface{}); ok {
				for _, c := range comps {
					cm, ok := c.(map[string]interface{})
					if !ok || cm["type"] != "QubitsSystem" {
						continue
					}
					if cm["qubitPerm"] != nil {
						continue
					}
					n := 0
					if origin, ok := cm["origin"].(map[string]interface{}); ok {
						if mods, ok := origin["modifierIDs"].([]interface{}); ok {
							n = len(mods)
						}
					}
					perm := make([]interface{}, n)
					for i := range perm {
						perm[i] = float64(i)
					}
					cm["qubitPerm"] = perm
				}
			}
		},
	},
	{
		To: "0.9.0",
		Apply: func(w map[string]interface{}) {
			if w["type"] != "RenderWindow" {
				return
			}
			w["saveVersion"] = "0.9.0"
			comps, ok := w["components"].([]interface{})
			if !ok {
				return
			}
			byID := map[float64]map[string]interface{}{}
			for _, c := range comps {
				cm, ok := c.(map[string]interface{})
				if !ok {
					continue
				}
				if id, ok := cm["id"].(float64); ok {
					byID[id] = cm
				}
			}
			maxID := maxNestedID(w)
			kept := make([]interface{}, 0, len(comps))
			var added []interface{}
			for _, c := range comps {
				cm, ok := c.(map[string]interface{})
				if !ok || cm["type"] != "SourceGate" {
					kept = append(kept, c)
					continue
				}
				if sys := convertSourceGate(cm, byID, &maxID); sys != nil {
					added = append(added, sys)
				}
			}
			w["components"] = append(kept, added...)
		},
	},
}

// sourceGateAmps returns the engine steady-state amplitude for a saved
// source: plain numeric pairs rescaled so probabilities sum to 1 (mirroring
// qubits.Normalize, which the source applied every frame), symbolic {expr}
// entries and zero vectors passing through untouched.
func sourceGateAmps(raw interface{}) []interface{} {
	list, ok := raw.([]interface{})
	if !ok || len(list) != 2 {
		return []interface{}{
			map[string]interface{}{"real": float64(1), "imag": float64(0)},
			map[string]interface{}{"real": float64(0), "imag": float64(0)},
		}
	}
	re := make([]float64, 2)
	im := make([]float64, 2)
	for i, a := range list {
		m, ok := a.(map[string]interface{})
		if !ok {
			return append([]interface{}{}, list...)
		}
		if _, hasExpr := m["expr"]; hasExpr {
			return append([]interface{}{}, list...)
		}
		re[i], _ = m["real"].(float64)
		im[i], _ = m["imag"].(float64)
	}
	sum := 0.0
	for i := range re {
		sum += re[i]*re[i] + im[i]*im[i]
	}
	if sum == 0 {
		return append([]interface{}{}, list...)
	}
	inv := 1 / math.Sqrt(sum)
	out := make([]interface{}, 2)
	for i := range re {
		out[i] = map[string]interface{}{"real": re[i] * inv, "imag": im[i] * inv}
	}
	return out
}

func clearHookLink(obj map[string]interface{}, key string, hookID float64) {
	if v, ok := obj[key].(float64); ok && v == hookID {
		obj[key] = float64(0)
	}
}

// maxNestedID returns the largest numeric "id" anywhere in the window object,
// including nested hooks and determinators: synthesized components must not
// collide with those (component-level ids alone routinely undercount).
func maxNestedID(w map[string]interface{}) float64 {
	maxID := float64(0)
	var walk func(v interface{})
	walk = func(v interface{}) {
		switch t := v.(type) {
		case map[string]interface{}:
			for k, x := range t {
				if k == "id" {
					if id, ok := x.(float64); ok && id > maxID {
						maxID = id
					}
					continue
				}
				walk(x)
			}
		case []interface{}:
			for _, x := range t {
				walk(x)
			}
		}
	}
	walk(w)
	return maxID
}

// convertSourceGate replaces a saved SourceGate with a standalone raw qubit:
// the system fed by the source output keeps its wiring and determinators and
// takes over the (normalized) source amplitude, detached from the removed
// hook. When no output system exists in the file, a minimal standalone
// system is synthesized at the source position instead. Returns the
// synthesized system, or nil when the existing output system was reused.
func convertSourceGate(src map[string]interface{}, byID map[float64]map[string]interface{}, maxID *float64) map[string]interface{} {
	amps := sourceGateAmps(src["amplitude"])
	mod := float64(0)
	if v, ok := src["modifierID"].(float64); ok {
		mod = v
	}
	var outID, targetID float64
	if oh, ok := src["outHook"].(map[string]interface{}); ok && oh != nil {
		outID, _ = oh["id"].(float64)
		if hooked, _ := oh["isHooked"].(bool); hooked {
			targetID, _ = oh["targetID"].(float64)
		}
	}
	if target, ok := byID[targetID]; ok && targetID != 0 && target["type"] == "QubitsSystem" {
		if origin, ok := target["origin"].(map[string]interface{}); ok && origin != nil {
			origin["amplitudes"] = amps
			origin["modifierIDs"] = []interface{}{mod}
			origin["size"] = float64(1)
		} else {
			target["origin"] = map[string]interface{}{
				"amplitudes":  amps,
				"modifierIDs": []interface{}{mod},
				"size":        float64(1),
			}
		}
		if outID != 0 {
			for _, c := range byID {
				if c["type"] != "QubitsSystem" {
					continue
				}
				clearHookLink(c, "hookID", outID)
				clearHookLink(c, "infoHookID", outID)
			}
		}
		return nil
	}
	*maxID++
	center := map[string]interface{}{"x": float64(0), "y": float64(0)}
	if c, ok := src["center"].(map[string]interface{}); ok && c != nil {
		center = c
	}
	return map[string]interface{}{
		"type": "QubitsSystem", "id": *maxID,
		"center": center, "radius": float64(30),
		"color": map[string]interface{}{"r": float64(0), "g": float64(121), "b": float64(241), "a": float64(255)},
		"isFixed": false, "hookID": float64(0), "infoHookID": float64(0),
		"isLogical": false, "probability": float64(1),
		"origin": map[string]interface{}{
			"amplitudes":  amps,
			"modifierIDs": []interface{}{mod},
			"size":        float64(1),
		},
		"qubitDeterminators": []interface{}{},
		"qubitPerm":          []interface{}{float64(0)},
	}
}

// parseVersion splits "major.minor.patch" into a comparable value; missing
// or malformed parts count as 0, so unnumbered saves sort below everything.
func parseVersion(v string) int64 {
	parts := strings.Split(v, ".")
	var n int64
	for i := 0; i < 3 && i < len(parts); i++ {
		p, err := strconv.ParseInt(parts[i], 10, 32)
		if err != nil {
			p = 0
		}
		n = n<<20 | p
	}
	return n
}

// Migrate rewrites save text so every window line carries the current
// version. Lines already at the current version and unparseable lines are
// passed through untouched.
func Migrate(data string) string {
	cur := parseVersion(CurrentVersion)
	if cur == 0 {
		return data
	}
	lines := strings.Split(data, "\n")
	var out []string
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		var w map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &w); err != nil {
			out = append(out, line) // keep malformed lines for the loader
			continue
		}
		ver := "0.0.0"
		if v, ok := w["saveVersion"].(string); ok {
			ver = v
		}
		changed := false
		for parseVersion(ver) < cur {
			applied := false
			for _, s := range steps {
				if parseVersion(s.To) > parseVersion(ver) {
					s.Apply(w)
					ver = s.To
					applied = true
					changed = true
					break
				}
			}
			if !applied {
				break // no further steps: leave as-is
			}
		}
		if !changed {
			out = append(out, line)
			continue
		}
		b, err := json.Marshal(w)
		if err != nil {
			out = append(out, line)
			continue
		}
		out = append(out, string(b))
	}
	return strings.Join(out, "\n")
}
