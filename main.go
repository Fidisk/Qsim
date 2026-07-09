package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"qsim/components"
	"qsim/effect"
	glob "qsim/globals"
	"qsim/qubits"
	"qsim/utils"

	"qsim/windows"
)

func main() {
	InitMainWindow(1600, 900, "Floating Panels")

	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	// Create one or more draggable inner panels

	//panel := windows.NewWindow(200, 150, 400, 300)
	panel := windows.NewRenderWindow(0, 0, 1500, 800)
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

	rows := []components.InfoRow{
		{"Speed and Something that burn my retina", 0.75, 3 + 4i},
		{"Power", 0.2, -1 + 2i},
	}
	table := components.NewInfoTable(400, 200, 260, 100, glob.ColorBg, rows)

	t1 := components.NewInfoTable(400, 200, 260, 40, glob.ColorBg, []components.InfoRow{})
	t2 := components.NewInfoTable(400, 200, 260, 40, glob.ColorBg, []components.InfoRow{})
	t3 := components.NewInfoTable(400, 200, 260, 40, glob.ColorBg, []components.InfoRow{})
	t4 := components.NewInfoTable(400, 200, 260, 40, glob.ColorBg, []components.InfoRow{})
	t5 := components.NewInfoTable(400, 200, 260, 40, glob.ColorBg, []components.InfoRow{})
	t6 := components.NewInfoTable(400, 200, 260, 40, glob.ColorBg, []components.InfoRow{})

	testSource := components.NewSourceGate(300, 300, 100, glob.GateColor, "Test", []complex64{0, 1}, 4)
	panel.PushComponent(testSource)

	panel.PushComponent(table, t1, t2, t3, t4, t5, t6)

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
	toolBar.IsZoomAllow(false)
	toolBar.IsDragAllow(true)
	toolBar.PinCamera(func() rl.Vector2 { return rl.Vector2{X: float32(toolBar.Width)/2.0 - 800, Y: 0} })
	toolBar.AddEffect(func() { effect.ScaleWidthToScreen(toolBar) })
	toolBar.IsSpawnAllow(false)
	toolBar.Rename("Tool bar")

	ToolBut1 := components.NewToggleButton(-750, 0, 100, 25, rl.LightGray, "Normal",
		func() bool { return utils.IsMouseState(glob.MouseStateNormal) }, 20,
		func() { utils.SetMouseState(glob.MouseStateNormal) },
		func() { utils.ToggleMouseState(glob.MouseStateNormal) })
	ToolBut2 := components.NewToggleButton(-650, 0, 100, 25, rl.LightGray, "Detach",
		func() bool { return utils.IsMouseState(glob.MouseStateDetach) }, 20,
		func() { utils.SetMouseState(glob.MouseStateDetach) },
		func() { utils.ToggleMouseState(glob.MouseStateDetach) })
	ToolBut3 := components.NewToggleButton(-550, 0, 100, 25, rl.LightGray, "Fix",
		func() bool { return utils.IsMouseState(glob.MouseStateFix) }, 20,
		func() { utils.SetMouseState(glob.MouseStateFix) },
		func() { utils.ToggleMouseState(glob.MouseStateFix) })
	ToolBut4 := components.NewToggleButton(-450, 0, 100, 25, rl.LightGray, "Spawn",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) }, 20,
		func() { utils.SetMouseState(glob.MouseStateSpawn) },
		func() { utils.ToggleMouseState(glob.MouseStateSpawn) })
	ToolBut5 := components.NewToggleButton(-350, 0, 100, 25, rl.LightGray, "Erase",
		func() bool { return utils.IsMouseState(glob.MouseStateErase) }, 20,
		func() { utils.SetMouseState(glob.MouseStateErase) },
		func() { utils.ToggleMouseState(glob.MouseStateErase) })

	toolBar.PushComponent(ToolBut1, ToolBut2, ToolBut3, ToolBut4, ToolBut5)

	spawnBar := windows.NewRenderWindow(1500, 0, 100, 800)
	spawnBar.IsResizeAllow(false)
	spawnBar.IsPanAllow(false)
	spawnBar.IsZoomAllow(false)
	spawnBar.IsDragAllow(true)
	spawnBar.IsSpawnAllow(false)
	spawnBar.Rename("Object")

	SpawnBut1 := components.NewToggleButton(-25, -362.5, 50, 50, rl.LightGray, "Q",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.Qubit) }, 40,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.Qubit)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.Qubit)
		})

	SpawnBut2 := components.NewToggleButton(25, -362.5, 50, 50, rl.LightGray, "H",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.Hadamard) }, 40,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.Hadamard)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.Hadamard)
		})

	SpawnBut3 := components.NewToggleButton(-25, -312.5, 50, 50, rl.LightGray, "X",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.X) }, 40,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.X)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.X)
		})

	SpawnBut4 := components.NewToggleButton(25, -312.5, 50, 50, rl.LightGray, "Y",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.Y) }, 40,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.Y)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.Y)
		})

	SpawnBut5 := components.NewToggleButton(-25, -262.5, 50, 50, rl.LightGray, "Z",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.Z) }, 40,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.Z)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.Z)
		})

	SpawnBut6 := components.NewToggleButton(25, -262.5, 50, 50, rl.LightGray, "CX",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.CX) }, 30,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.CX)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.CX)
		})

	SpawnBut7 := components.NewToggleButton(-25, -212.5, 50, 50, rl.LightGray, "CY",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.CY) }, 30,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.CY)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.CY)
		})

	SpawnBut8 := components.NewToggleButton(25, -212.5, 50, 50, rl.LightGray, "CZ",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.CZ) }, 30,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.CZ)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.CZ)
		})

	SpawnBut9 := components.NewToggleButton(-25, -162.5, 50, 50, rl.LightGray, "M",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.Measurement) }, 30,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.Measurement)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.Measurement)
		})

	spawnBar.PushComponent(SpawnBut1, SpawnBut2, SpawnBut3, SpawnBut4, SpawnBut5, SpawnBut6, SpawnBut7, SpawnBut8, SpawnBut9)

	winManager = append(winManager, panel, toolBar, spawnBar)

	update := func() {
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

	for !rl.WindowShouldClose() {
		update()

		//fmt.Println(toolBar.Camera.Target)
	}
}
