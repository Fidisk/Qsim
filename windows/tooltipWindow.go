package windows

import (
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"

	glob "qsim/globals"
)

type TooltipWindow struct {
	RenderWindow
	text string
}

func NewTooltipWindow() *TooltipWindow {
	tw := &TooltipWindow{}
	tw.RenderWindow = *NewRenderWindow(0, 0, 10, 10)
	tw.IsTitleBarVisible = false
	tw.CanDrag = false
	tw.CanResize = false
	tw.CanPan = false
	tw.CanZoom = false
	tw.CanSpawn = false
	tw.ShowGrid = false
	tw.SetPriority(300) // drawn last, above every other window
	return tw
}

func (tw *TooltipWindow) Update() {
	tw.text = glob.TooltipText
	glob.TooltipText = ""
	if tw.text == "" {
		return
	}

	const fontSize int32 = 14
	const padX int32 = 16
	const padY int32 = 12
	const lineGap int32 = 4

	lines := strings.Split(tw.text, "\n")
	w := int32(0)
	for _, l := range lines {
		if mw := rl.MeasureText(l, fontSize); mw > w {
			w = mw
		}
	}
	tw.Width = w + padX
	tw.Height = int32(len(lines))*(fontSize+lineGap) + padY - lineGap

	mouse := rl.GetMousePosition()
	x := int32(mouse.X) + 14
	y := int32(mouse.Y) + 14
	if x+tw.Width > int32(glob.MainWindowWidth) {
		x = int32(mouse.X) - tw.Width - 6
	}
	if y+tw.Height > int32(glob.MainWindowHeight) {
		y = int32(mouse.Y) - tw.Height - 6
	}
	tw.X = max(0, min(x, int32(glob.MainWindowWidth)-tw.Width))
	tw.Y = max(0, min(y, int32(glob.MainWindowHeight)-tw.Height))
}

func (tw *TooltipWindow) Draw() {
	if tw.text == "" {
		return
	}

	const fontSize int32 = 14
	const lineGap int32 = 4

	rect := rl.NewRectangle(float32(tw.X), float32(tw.Y), float32(tw.Width), float32(tw.Height))
	rl.DrawRectangleRounded(rect, 0.15, 4, rl.NewColor(25, 25, 25, 240))
	rl.DrawRectangleRoundedLinesEx(rect, 0.15, 4, 1, rl.Fade(rl.SkyBlue, 0.6))
	for i, l := range strings.Split(tw.text, "\n") {
		rl.DrawText(l, tw.X+8, tw.Y+6+int32(i)*(fontSize+lineGap), fontSize, rl.White)
	}
}

func (tw *TooltipWindow) PostUpdate() {}

func (tw *TooltipWindow) SaveState() string { return "" }
