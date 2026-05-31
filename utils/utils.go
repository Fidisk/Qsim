package utils

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func Sqr(l float32) float32 {
	return l * l
}

func Dist(l, r rl.Vector2) float32 {
	return float32(math.Sqrt(float64(Sqr(r.X-l.X) + Sqr(r.Y-l.Y))))
}
