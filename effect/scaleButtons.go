package effect

import (
	"qsim/components"
	"qsim/windows"
)

var ScaleButtonsToWidth = func(rw *windows.RenderWindow) {
	elems := rw.GetElement()
	if len(elems) == 0 {
		return
	}

	btnHeight := float32(25)
	numPairs := len(elems) / 2

	for i := 0; i < numPairs; i++ {
		fileBtn := elems[i*2]
		xBtn := elems[i*2+1]

		if fb, ok := fileBtn.(*components.Button); ok {
			if xb, ok := xBtn.(*components.Button); ok {
				totalWidth := fb.Width + xb.Width
				scale := float32(rw.Width) / totalWidth

				newFileWidth := fb.Width * scale

				fb.Width = newFileWidth

				fb.Center.X = -float32(xb.Width) / 2
				fb.Center.Y = float32(i-1)*btnHeight - float32(rw.Width)/2

				xb.Center.X = float32(newFileWidth) / 2
				xb.Center.Y = float32(i-1)*btnHeight - float32(rw.Width)/2
			}
		}
	}

	if len(elems) > 0 {
	}
}
