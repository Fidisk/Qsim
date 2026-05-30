package attributes

type AttrManager struct {
	modifier []attributes
}

func newAttrManager() *AttrManager {
	red := NewColor(255, 0, 0)
	green := NewColor(0, 255, 0)
	blue := NewColor(0, 0, 255)
	square := NewSide(5, 4)
	hexa := NewSide(7, 4)
	tmp := []attributes{red, green, blue, square, hexa}
	return &AttrManager{
		modifier: tmp,
	}
}

var AttributesManager = newAttrManager()

func (a *AttrManager) Get(id int32) attributes {
	return a.modifier[id]
}
