package attributes

type Color struct {
	R int32
	G int32
	B int32
}

func NewColor(r, g, b int32) *Color {
	return &Color{
		R: r,
		G: g,
		B: b,
	}
}
