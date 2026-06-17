package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"qsim/components"
	glob "qsim/globals"
	"qsim/qubits"

	"qsim/windows"
)

func main() {
	InitMainWindow(1600, 900, "Floating Panels")

	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	// Create one or more draggable inner panels

	//panel := windows.NewWindow(200, 150, 400, 300)
	panel := windows.NewRenderWindow(0, 0, 1600, 800)
	//panel2 := windows.NewRenderWindow(0, 0, 1600, 900)
	toolBar := windows.NewRenderWindow(0, 800, 1600, 50)

	// === MINIMUM CHANGE: Add text panel beside previous panels ===
	myTextWindow := windows.NewTextWindow(
		"State Editor Panel",
		50, 400, 320, 150,
		"",   // Initial text inside the window box
		true, // IsEditable: set to true so you can type into it
		20,   // Max characters allowed
		func(finalText string) {
			println("Submitted text update:", finalText)
		},
	)
	myTextWindow.TextBuffer = "hello"

	var create = func(p *windows.RenderWindow) {
		q1 := components.NewQubitsSystem(100, 100, glob.QubitSystemRadius, glob.QubitSystemColor)
		q2 := components.NewQubitsSystem(100, 100, glob.QubitSystemRadius, glob.QubitSystemColor)
		q3 := components.NewQubitsSystem(100, 100, glob.QubitSystemRadius, glob.QubitSystemColor)

		q1State := qubits.NewQubitStateManagerFrom([]complex64{0.6, 0.8}, []int32{0})
		q2State := qubits.NewQubitStateManagerFrom([]complex64{1, 0}, []int32{1})
		q3State := qubits.NewQubitStateManagerFrom([]complex64{1, 0}, []int32{2})

		q1.Assign(q1State)
		q2.Assign(q2State)
		q3.Assign(q3State)

		p.PushComponent(q1, q2, q3)

		t := complex(float32(1/math.Sqrt(2)), 0)
		H1 := components.NewGate(100, 100, glob.GateRadius, glob.GateColor, "H", [][]complex64{{t, t}, {t, -t}}, 1)
		H2 := components.NewGate(100, 100, glob.GateRadius, glob.GateColor, "H", [][]complex64{{t, t}, {t, -t}}, 1)
		H3 := components.NewGate(100, 100, glob.GateRadius, glob.GateColor, "H", [][]complex64{{t, t}, {t, -t}}, 1)
		CNOT1 := components.NewGate(100, 100, glob.GateRadius, glob.GateColor, "CNOT", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 0}}, 2)
		CNOT2 := components.NewGate(100, 100, glob.GateRadius, glob.GateColor, "CNOT", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 0}}, 2)

		M1 := components.NewMeasurementGate(100, 100, glob.GateRadius, glob.GateColor, "M")
		M2 := components.NewMeasurementGate(100, 100, glob.GateRadius, glob.GateColor, "M")

		CX := components.NewGate(100, 100, glob.GateRadius, glob.GateColor, "CX", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 0}}, 2)
		CZ := components.NewGate(100, 100, glob.GateRadius, glob.GateColor, "CZ", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, -1}}, 2)

		p.PushComponent(H1, H2, H3, CNOT1, CNOT2, M1, M2, CX, CZ)
	}

	create(panel)

	//blob := comp.NewBlob(0, 0, 50)

	//circle := comp.NewQubitsSystem(100, 100, 30, rl.Red)
	//circle2 := comp.NewQubitsSystem(150, 100, 30, rl.Blue)

	//qubit := comp.NewQubit(0, 0.1, 0.6, 0, 0.2, 0.3, 1.75)

	//circle.QubitList = append(circle.QubitList, qubit)

	//test := qub.NewQubitStateManagerFrom([]complex64{0.5, 0.5, 0.5, 0.5}, []int32{0, 4})

	//test2 := qub.NewQubitStateManagerFrom([]complex64{0, 0.6, 0, 0.8}, []int32{0, 2})

	//test2.SwapColumn(0, 1)
	//test2.SwapColumn(0, 1)

	//circle.Assign(test)
	//circle2.Assign(test2)

	//circle.SetParent(panel2)
	//circle2.SetParent(panel2)

	//testHook := comp.NewHook(200, 200, 30, rl.Blue)
	//testHook2 := comp.NewHook(50, 50, 30, rl.Blue)

	//testGate := comp.NewGate(200, 60, 30, rl.Lime, "CNOT", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 0}}, 2)
	//testGate.SetParent(panel2)

	//testGate.HookList = append(testGate.HookList, testHook, testHook2)

	//panel2.PushComponent(circle, circle2)

	//panel2.PushComponent(testGate)

	toolBar.IsResizeAllow(false)
	toolBar.IsPanAllow(false)
	toolBar.Rename("Tool bar")

	state := true
	But1 := components.NewToggleButton(-750, 0, 100, 25, rl.LightGray, "TestA", &state, 20, func() {}, func() {})
	But2 := components.NewToggleButton(-650, 0, 100, 25, rl.LightGray, "TestA", &state, 20, func() {}, func() {})

	toolBar.PushComponent(But1, But2)

	winManager = append(winManager, panel, toolBar)

	for !rl.WindowShouldClose() {
		// Update main window resize
		glob.Refresh()

		UpdateMainWindow()

		// Update inner panels
		for _, d := range winManager {
			d.Update()
		}

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

		shuffleWinManager()
	}
}
