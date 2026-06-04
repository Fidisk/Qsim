package main

import (
	comp "qsim/components"

	rl "github.com/gen2brain/raylib-go/raylib"

	glob "qsim/globals"

	qub "qsim/qubits"

	"qsim/windows"
)

func main() {
	InitMainWindow(800, 600, "Floating Panels")

	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	// Create one or more draggable inner panels

	panel := windows.NewWindow(200, 150, 400, 300)
	panel2 := windows.NewRenderWindow(400, 150, 400, 300)

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

	panel2.WComp = append(panel2.WComp, circle)
	panel2.WComp = append(panel2.WComp, circle2)

	testHook := comp.NewHook(200, 200, 30, rl.Blue)
	testHook2 := comp.NewHook(50, 50, 30, rl.Blue)
	testGate := comp.NewGate(200, 60, 30, rl.Lime, "XOR")

	testGate.HookList = append(testGate.HookList, testHook, testHook2)

	panel2.WComp = append(panel2.WComp, testGate)

	winManager = append(winManager, panel, panel2)

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
