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

var QubitModifierID int32 = 0

func GenerateQubitModifierID() int32 {
	id := QubitModifierID
	AttributesManager.modifier = append(AttributesManager.modifier, NewName("Q"+strconv.Itoa(int(id))))
	QubitModifierID++
	return id
}
