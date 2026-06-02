package utils

import (
	"math"
	"qsim/globals"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func Sqr(l float32) float32 {
	return l * l
}

func Dist(l, r rl.Vector2) float32 {
	return float32(math.Sqrt(float64(Sqr(r.X-l.X) + Sqr(r.Y-l.Y))))
}

// This also allow this to store both concrete and pointer
// A whole can of worm is here
type Object interface {
}

// This implementation introduce memory leak
// F...
var ObjectList []Object

func GenerateID(val Object) int32 {
	//Dumb way to do this
	ObjectList = append(ObjectList, val)
	globals.CurrentID++
	return globals.CurrentID
}

func GetObjectFromID(id int32) Object {
	if id == 0 {
		return nil
	}
	if id > int32(len(ObjectList)) {
		return nil
	}
	return ObjectList[id-1]
}
