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

func DeleteObjectWithID(idList ...int32) {
	for _, id := range idList {
		ObjectList[id-1] = nil //Memory leak here
	}
}

func IsMouseState(state globals.MouseOperation) bool {
	return (globals.MouseState & state) != 0
}

func GetMouseState() globals.MouseOperation {
	return globals.MouseState
}

func SetMouseState(state globals.MouseOperation) {
	globals.MouseState = state
}

func ToggleMouseState(state globals.MouseOperation) {
	globals.MouseState |= state
}
