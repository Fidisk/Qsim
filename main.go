package main

import (
	comp "qsim/components"

	rl "github.com/gen2brain/raylib-go/raylib"

	glob "qsim/globals"

	qub "qsim/qubits"
)

func main() {
	InitMainWindow(800, 600, "Floating Panels")

	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	// Create one or more draggable inner panels

	panel := NewWindow(200, 150, 400, 300)
	panel2 := NewRenderWindow(400, 150, 400, 300)

	//blob := comp.NewBlob(0, 0, 50)

	circle := comp.NewQubitsSystem(100, 100, 30, rl.Red)

	//qubit := comp.NewQubit(0, 0.1, 0.6, 0, 0.2, 0.3, 1.75)

	//circle.QubitList = append(circle.QubitList, qubit)

	test := qub.NewQubitStateManagerFrom([]float32{0.6, 0.8}, []int32{0, 1})

	circle.Assign(test)

	panel2.wComp = append(panel2.wComp, circle)

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
