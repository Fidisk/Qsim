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
