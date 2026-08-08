package effect

import (
	"qsim/globals"
	"qsim/windows"
)

// PinToSide pins the window to the left or right screen edge only when it is
// dragged into the outer quarter of the screen: the left quarter pins it to
// the left edge, the right quarter pins it to the right edge. In the middle
// half it stays wherever it is dropped. The Y position is never touched.
var PinToSide = func(rw *windows.RenderWindow) {
	w := float32(globals.MainWindowWidth)
	center := float32(rw.X) + float32(rw.Width)/2
	if center < w/4 {
		rw.X = 0
	} else if center > w*3/4 {
		rw.X = int32(w - float32(rw.Width))
	}
}
