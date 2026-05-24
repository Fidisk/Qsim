package components

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ----------------------------------------------------------------------
// Blob – a soft‑body slime that can be dragged
// ----------------------------------------------------------------------
type Blob struct {
	center     rl.Vector2 // current center (in content coordinates)
	radius     float32
	particles  []verletPoint
	nParticles int

	// dragging state
	isDragging bool
	dragOffset rl.Vector2

	// physics constants
	stiffness float32
	damping   float32
}

type verletPoint struct {
	pos, prev rl.Vector2
}

// NewBlob creates a new slime Blob at (x, y) relative to the **content area**.
func NewBlob(x, y, radius float32) *Blob {
	n := 12
	b := &Blob{
		center:     rl.Vector2{X: x, Y: y},
		radius:     radius,
		nParticles: n,
		particles:  make([]verletPoint, n),
		stiffness:  0.5,
		damping:    0.98,
	}

	for i := 0; i < n; i++ {
		angle := 2 * math.Pi * float32(i) / float32(n)
		px := x + radius*float32(math.Cos(float64(angle)))
		py := y + radius*float32(math.Sin(float64(angle)))
		b.particles[i] = verletPoint{
			pos:  rl.Vector2{X: px, Y: py},
			prev: rl.Vector2{X: px, Y: py},
		}
	}
	return b
}

// ----------------------------------------------------------------------
// Component interface methods
// ----------------------------------------------------------------------

// Update must be called every frame. The parameters define the **content area**
// (x, y = top‑left corner, width, height = dimensions).
func (b *Blob) Update(contentX, contentY, contentWidth, contentHeight float32) {
	// 1. Mouse position relative to the content area
	mouse := rl.GetMousePosition()
	mouseRel := rl.Vector2{
		X: mouse.X - contentX,
		Y: mouse.Y - contentY,
	}

	// 2. Drag detection
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		dist := rl.Vector2Distance(mouseRel, b.center)
		if dist < b.radius*1.5 {
			b.isDragging = true
			b.dragOffset = rl.Vector2Subtract(b.center, mouseRel)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
		b.isDragging = false
	}

	// 3. Move center while dragging
	if b.isDragging {
		b.center = rl.Vector2Add(mouseRel, b.dragOffset)
		// Optional: keep inside content area
		b.center = clampToContent(b.center, b.radius, contentX, contentY, contentWidth, contentHeight)
	}

	// 4. Verlet physics
	dt := rl.GetFrameTime()
	dtSq := dt * dt

	for i := range b.particles {
		p := &b.particles[i]
		toCenter := rl.Vector2Subtract(b.center, p.pos)
		dist := rl.Vector2Length(toCenter)
		if dist > 0 {
			force := (dist - b.radius) * b.stiffness
			acc := rl.Vector2Scale(rl.Vector2Normalize(toCenter), force)
			velocity := rl.Vector2Scale(rl.Vector2Subtract(p.pos, p.prev), b.damping)
			newPos := rl.Vector2Add(p.pos, rl.Vector2Add(velocity, rl.Vector2Scale(acc, dtSq)))
			p.prev = p.pos
			p.pos = newPos
		}
	}
}

// Draw renders the Blob. Parameters are the same content area as in Update.
func (b *Blob) Draw(contentX, contentY, contentWidth, contentHeight float32) {
	n := b.nParticles
	if n < 3 {
		return
	}

	verts := make([]rl.Vector2, n+2)
	verts[0] = b.center
	for i := 0; i < n; i++ {
		verts[i+1] = b.particles[i].pos
	}
	verts[n+1] = b.particles[0].pos

	color := rl.NewColor(50, 220, 100, 200)
	rl.DrawTriangleFan(verts, color)

	outlineColor := rl.NewColor(20, 180, 80, 255)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		rl.DrawLineV(b.particles[i].pos, b.particles[j].pos, outlineColor)
	}
}

func (b *Blob) PostUpdate() {}

// ----------------------------------------------------------------------
// Helper
// ----------------------------------------------------------------------
func clampToContent(pos rl.Vector2, r, contentX, contentY, contentW, contentH float32) rl.Vector2 {
	pos.X = rl.Clamp(pos.X, contentX+r, contentX+contentW-r)
	pos.Y = rl.Clamp(pos.Y, contentY+r, contentY+contentH-r)
	return pos
}
