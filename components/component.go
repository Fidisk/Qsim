package components

type Component interface {
	Draw(contentX, contentY, contentWidth, contentHeight float32)
	Update(contentX, contentY, contentWidth, contentHeight float32)
	PostUpdate()
}
