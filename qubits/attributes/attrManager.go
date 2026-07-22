package attributes

import (
	"strconv"
)

type AttrManager struct {
	modifier []attributes
}

func newAttrManager() *AttrManager {
	tmp := []attributes{}
	return &AttrManager{
		modifier: tmp,
	}
}

var AttributesManager = newAttrManager()

func (a *AttrManager) Get(id int32) attributes {
	return a.modifier[id]
}

// Len reports the number of registered modifiers, for bounds checks before Get.
func (a *AttrManager) Len() int {
	return len(a.modifier)
}

// SetName replaces the attribute of a qubit modifier with the given display
// name. Out-of-range ids are ignored.
func (a *AttrManager) SetName(id int32, val string) {
	if id < 0 || int(id) >= len(a.modifier) {
		return
	}
	a.modifier[id] = NewName(val)
}

var QubitModifierID int32 = 0

func GenerateQubitModifierID() int32 {
	id := QubitModifierID
	AttributesManager.modifier = append(AttributesManager.modifier, NewName("Q"+strconv.Itoa(int(id))))
	QubitModifierID++
	return id
}
