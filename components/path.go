// Vibe code this to handle queso-SVG, better to change to real SVG when it is finally supported
package components

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Command types
type CmdType int

const (
	Move CmdType = iota
	Line
	Quad
	Cubic
	Close
)

type Command struct {
	Type CmdType
	P    [4]rl.Vector2 // up to 4 control points
}

// Path holds a series of commands
type Path struct {
	Commands []Command
}

func (p *Path) Flatten(tolerance float32) [][]rl.Vector2 {
	const epsilon = 1e-6
	const maxDepth = 16 // safety cap (2^16 = 65536 segments max)

	var subpaths [][]rl.Vector2
	var current rl.Vector2
	var subpath []rl.Vector2
	var subpathStart *rl.Vector2

	addPoint := func(pt rl.Vector2) {
		if len(subpath) == 0 {
			subpath = append(subpath, pt)
			return
		}
		last := subpath[len(subpath)-1]
		if math.Abs(float64(pt.X-last.X)) > epsilon || math.Abs(float64(pt.Y-last.Y)) > epsilon {
			subpath = append(subpath, pt)
		}
	}

	var flattenQuad func(p0, p1, p2 rl.Vector2, depth int)
	flattenQuad = func(p0, p1, p2 rl.Vector2, depth int) {
		dx := p2.X - p0.X
		dy := p2.Y - p0.Y
		length := float32(math.Hypot(float64(dx), float64(dy)))
		if length < epsilon {
			addPoint(p2)
			return
		}
		// Distance from p1 to chord p0-p2
		dist := float32(math.Abs(float64((p1.X-p0.X)*dy-(p1.Y-p0.Y)*dx)) / float64(length))
		if dist <= tolerance || depth >= maxDepth {
			addPoint(p2)
			return
		}
		// Split at t=0.5
		mid1 := rl.Vector2{X: (p0.X + p1.X) / 2, Y: (p0.Y + p1.Y) / 2}
		mid2 := rl.Vector2{X: (p1.X + p2.X) / 2, Y: (p1.Y + p2.Y) / 2}
		mid := rl.Vector2{X: (mid1.X + mid2.X) / 2, Y: (mid1.Y + mid2.Y) / 2}
		flattenQuad(p0, mid1, mid, depth+1)
		flattenQuad(mid, mid2, p2, depth+1)
	}

	var flattenCubic func(p0, p1, p2, p3 rl.Vector2, depth int)
	flattenCubic = func(p0, p1, p2, p3 rl.Vector2, depth int) {
		dx := p3.X - p0.X
		dy := p3.Y - p0.Y
		length := float32(math.Hypot(float64(dx), float64(dy)))
		if length < epsilon {
			addPoint(p3)
			return
		}
		// Distances of p1 and p2 from chord p0-p3
		dist1 := float32(math.Abs(float64((p1.X-p0.X)*dy-(p1.Y-p0.Y)*dx)) / float64(length))
		dist2 := float32(math.Abs(float64((p2.X-p0.X)*dy-(p2.Y-p0.Y)*dx)) / float64(length))
		if (dist1 <= tolerance && dist2 <= tolerance) || depth >= maxDepth {
			addPoint(p3)
			return
		}
		// Split at t=0.5
		mid1 := rl.Vector2{X: (p0.X + p1.X) / 2, Y: (p0.Y + p1.Y) / 2}
		mid2 := rl.Vector2{X: (p1.X + p2.X) / 2, Y: (p1.Y + p2.Y) / 2}
		mid3 := rl.Vector2{X: (p2.X + p3.X) / 2, Y: (p2.Y + p3.Y) / 2}
		mid12 := rl.Vector2{X: (mid1.X + mid2.X) / 2, Y: (mid1.Y + mid2.Y) / 2}
		mid23 := rl.Vector2{X: (mid2.X + mid3.X) / 2, Y: (mid2.Y + mid3.Y) / 2}
		mid := rl.Vector2{X: (mid12.X + mid23.X) / 2, Y: (mid12.Y + mid23.Y) / 2}
		flattenCubic(p0, mid1, mid12, mid, depth+1)
		flattenCubic(mid, mid23, mid3, p3, depth+1)
	}

	for _, cmd := range p.Commands {
		switch cmd.Type {
		case Move:
			if len(cmd.P) > 0 {
				if len(subpath) > 0 {
					subpaths = append(subpaths, subpath)
					subpath = nil
				}
				subpathStart = &cmd.P[0]
				current = cmd.P[0]
				addPoint(current)
			}
		case Line:
			if len(cmd.P) > 0 {
				addPoint(cmd.P[0])
				current = cmd.P[0]
			}
		case Quad:
			if len(cmd.P) >= 2 {
				p0 := current
				p1 := cmd.P[0]
				p2 := cmd.P[1]
				flattenQuad(p0, p1, p2, 0)
				current = p2
			}
		case Cubic:
			if len(cmd.P) >= 3 {
				p0 := current
				p1 := cmd.P[0]
				p2 := cmd.P[1]
				p3 := cmd.P[2]
				flattenCubic(p0, p1, p2, p3, 0)
				current = p3
			}
		case Close:
			if subpathStart != nil {
				addPoint(*subpathStart)
				current = *subpathStart
				subpathStart = nil
			}
		}
	}
	if len(subpath) > 0 {
		subpaths = append(subpaths, subpath)
	}
	return subpaths
}

func ParseSVGPath(d string) *Path {
	// Remove commas and extra spaces
	d = strings.ReplaceAll(d, ",", " ")
	tokens := strings.Fields(d)
	var cmds []Command
	var x, y float64
	i := 0
	for i < len(tokens) {
		cmd := tokens[i]
		i++
		switch cmd {
		case "M":
			if i+1 >= len(tokens) {
				break
			}
			x, _ = strconv.ParseFloat(tokens[i], 32)
			y, _ = strconv.ParseFloat(tokens[i+1], 32)
			i += 2
			cmds = append(cmds, Command{Type: Move, P: [4]rl.Vector2{{X: float32(x), Y: float32(y)}}})
		case "L":
			if i+1 >= len(tokens) {
				break
			}
			x, _ = strconv.ParseFloat(tokens[i], 32)
			y, _ = strconv.ParseFloat(tokens[i+1], 32)
			i += 2
			cmds = append(cmds, Command{Type: Line, P: [4]rl.Vector2{{X: float32(x), Y: float32(y)}}})
		case "Q":
			if i+3 >= len(tokens) {
				break
			}
			x1, _ := strconv.ParseFloat(tokens[i], 32)
			y1, _ := strconv.ParseFloat(tokens[i+1], 32)
			x2, _ := strconv.ParseFloat(tokens[i+2], 32)
			y2, _ := strconv.ParseFloat(tokens[i+3], 32)
			i += 4
			cmds = append(cmds, Command{Type: Quad, P: [4]rl.Vector2{{X: float32(x1), Y: float32(y1)}, {X: float32(x2), Y: float32(y2)}}})
		case "C":
			if i+5 >= len(tokens) {
				break
			}
			x1, _ := strconv.ParseFloat(tokens[i], 32)
			y1, _ := strconv.ParseFloat(tokens[i+1], 32)
			x2, _ := strconv.ParseFloat(tokens[i+2], 32)
			y2, _ := strconv.ParseFloat(tokens[i+3], 32)
			x3, _ := strconv.ParseFloat(tokens[i+4], 32)
			y3, _ := strconv.ParseFloat(tokens[i+5], 32)
			i += 6
			cmds = append(cmds, Command{Type: Cubic, P: [4]rl.Vector2{{X: float32(x1), Y: float32(y1)}, {X: float32(x2), Y: float32(y2)}, {X: float32(x3), Y: float32(y3)}}})
		case "Z", "z":
			cmds = append(cmds, Command{Type: Close})
		default:
			// skip unknown
		}
	}
	fmt.Println(d, cmds)
	return &Path{Commands: cmds}
}
