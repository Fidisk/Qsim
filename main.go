package main

import (
	comp "qsim/components"

	rl "github.com/gen2brain/raylib-go/raylib"

	glob "qsim/globals"

	qub "qsim/qubits"

	"qsim/windows"
)

func main() {
	InitMainWindow(1600, 900, "Floating Panels")

	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	// Create one or more draggable inner panels

	//panel := windows.NewWindow(200, 150, 400, 300)
	panel2 := windows.NewRenderWindow(0, 0, 1600, 900)

	// === MINIMUM CHANGE: Add text panel beside previous panels ===
	myTextWindow := windows.NewInputWindow(
		"State Editor Panel",
		50, 400, 320, 150,
		20,
		func(finalText string) {
			println("Submitted text update:", finalText)
		},
	)
	myTextWindow.TextBuffer = "hello"

	//blob := comp.NewBlob(0, 0, 50)

	circle := comp.NewQubitsSystem(100, 100, 30, rl.Red)
	circle2 := comp.NewQubitsSystem(150, 100, 30, rl.Blue)

	//qubit := comp.NewQubit(0, 0.1, 0.6, 0, 0.2, 0.3, 1.75)

	//circle.QubitList = append(circle.QubitList, qubit)

	test := qub.NewQubitStateManagerFrom([]float32{0.5, 0.5, 0.5, 0.5}, []int32{0, 4})

	test2 := qub.NewQubitStateManagerFrom([]float32{0, 0.6, 0, 0.8}, []int32{0, 2})

	circle.Assign(test)
	circle2.Assign(test2)

	circle.SetParent(panel2)
	circle2.SetParent(panel2)

	//testHook := comp.NewHook(200, 200, 30, rl.Blue)
	//testHook2 := comp.NewHook(50, 50, 30, rl.Blue)

	testGate := comp.NewGate(200, 60, 30, rl.Lime, "CNOT", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 0}}, 2)

	//testGate.HookList = append(testGate.HookList, testHook, testHook2)

	panel2.PushComponent(circle, circle2)

	panel2.PushComponent(testGate)

	winManager = append(winManager, panel2, myTextWindow)

	for !rl.WindowShouldClose() {
		// Update main window resize
		glob.Refresh()

		UpdateMainWindow()

		// Update inner panels
		for _, d := range winManager {
			d.Update()
		}

		shuffleWinManager()

		// Draw everything
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		for i := max(0, len(winManager)-1); i >= 0; i-- {
			winManager[i].Draw()
		}

		rl.EndDrawing()

		for _, d := range winManager {
			d.PostUpdate()
		}
	}
}
