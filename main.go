package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"

	"qsim/animation"
	"qsim/components"
	"qsim/effect"
	glob "qsim/globals"
	"qsim/utils"

	"qsim/windows"
)

// buildFileBrowser creates the "File Browser" dialog: one row per .qsim save
// in saves/ (a load button plus a red "x" delete button), followed by a
// single Close button that dismisses the dialog. Deleting a save rebuilds the
// whole dialog from the current directory instead of mutating the component
// list while it is being updated (which used to crash the update loop when
// the last row was removed).
func buildFileBrowser() *windows.RenderWindow {
	fileWin := windows.NewRenderWindow(200, 100, 400, 500)
	fileWin.Rename("File Browser")
	fileWin.SetPriority(200)
	fileWin.IsSpawnAllow(false)
	fileWin.ShowGrid = false

	savesDir := "saves"
	_ = os.MkdirAll(savesDir, 0755)
	entries, _ := os.ReadDir(savesDir)

	var comps []components.Component
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".qsim" {
			continue
		}
		name := e.Name()
		btn := components.NewButton(0, 0, 350, 25, rl.LightGray, name, 14,
			func(f string) func() {
				return func() {
					data, err := os.ReadFile(filepath.Join(savesDir, f))
					if err != nil {
						return
					}
					loaded := windows.LoadState(string(data))

					// Remove existing circuit panels
					animation.Reset()
					var keep []pWindow
					for _, w := range winManager {
						if rw, ok := w.(*windows.RenderWindow); ok && rw.CanSpawn {
							utils.DeleteObjectWithID(rw.GetID())
						} else {
							keep = append(keep, w)
						}
					}
					winManager = keep

					// Add loaded windows
					for _, w := range loaded {
						if rw, ok := w.(*windows.RenderWindow); ok {
							winManager = append(winManager, rw)
							panel = rw
						}
					}
					DeleteWindowByID(fileWin.GetID())
				}
			}(name))
		comps = append(comps, btn)
		xBtn := components.NewButton(0, 0, 25, 25, rl.Red, "x", 14,
			func(f string, fw *windows.RenderWindow) func() {
				return func() {
					os.Remove(filepath.Join("saves", f))
					DeleteWindowByID(fw.GetID())
					winManager = append(winManager, buildFileBrowser())
				}
			}(name, fileWin))
		comps = append(comps, xBtn)
	}

	closeBtn := components.NewButton(0, 0, 350, 25, rl.DarkGray, "Close", 14,
		func() { DeleteWindowByID(fileWin.GetID()) })
	comps = append(comps, closeBtn)

	fileWin.IsPanAllow(false)
	fileWin.IsZoomAllow(false)
	fileWin.IsVerticalScrollAllow(true)
	fileWin.PushComponent(comps...)
	fileWin.AddEffect(func() { effect.ScaleButtonsToWidth(fileWin) })
	fileWin.PinCamera(func() rl.Vector2 { return rl.Vector2{X: 0, Y: float32(math.Max(0, float64(fileWin.Camera.Target.Y)))} })
	return fileWin
}

// spawnObjectAt spawns the given component type at a screen position, as used
// by drag-out from the object bar: it finds the top-most spawnable window
// under the drop point and spawns there (reusing the window's click-to-spawn
// logic, which also turns the spawn mode back off). Nothing happens when the
// drop point is not over a spawnable window.
func spawnObjectAt(screenPos rl.Vector2, state glob.SpawnType) {
	for i := 0; i < len(winManager); i++ {
		rw, ok := winManager[i].(*windows.RenderWindow)
		if !ok || !rw.CanSpawn {
			continue
		}
		if rl.CheckCollisionPointRec(screenPos, rw.GetContentRect()) {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(state)
			world := rl.GetScreenToWorld2D(screenPos, rw.Camera)
			rw.OnClick(world)
			return
		}
	}
}

func main() {
	InitMainWindow(1600, 900, "Floating Panels")

	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	// Create one or more draggable inner panels

	//panel := windows.NewWindow(200, 150, 400, 300)
	panel = windows.NewRenderWindow(0, 0, 1500, 800)
	panel.Rename("Circuit")
	//panel2 := windows.NewRenderWindow(0, 0, 1600, 900)
	toolBar := windows.NewRenderWindow(0, 800, 1600, 50)
	toolBar.ShowGrid = false

	/*
		var create = func(p *windows.RenderWindow) {
			q1 := components.NewQubitsSystem(100, 100, glob.QubitSystemRadius, config.QubitSystemColor)
			q2 := components.NewQubitsSystem(100, 100, glob.QubitSystemRadius, config.QubitSystemColor)
			q3 := components.NewQubitsSystem(100, 100, glob.QubitSystemRadius, config.QubitSystemColor)

			q1State := qubits.NewQubitStateManagerFrom([]complex64{0.6, 0.8}, []int32{0})
			q2State := qubits.NewQubitStateManagerFrom([]complex64{1, 0}, []int32{1})
			q3State := qubits.NewQubitStateManagerFrom([]complex64{1, 0}, []int32{2})

			q1.Assign(q1State)
			q2.Assign(q2State)
			q3.Assign(q3State)

			p.PushComponent(q1, q2, q3)

			t := complex(float32(1/math.Sqrt(2)), 0)
			H1 := components.NewGate(100, 100, glob.GateRadius, config.GateColor, "H", [][]complex64{{t, t}, {t, -t}}, 1)
			H2 := components.NewGate(100, 100, glob.GateRadius, config.GateColor, "H", [][]complex64{{t, t}, {t, -t}}, 1)
			H3 := components.NewGate(100, 100, glob.GateRadius, config.GateColor, "H", [][]complex64{{t, t}, {t, -t}}, 1)
			CNOT1 := components.NewGate(100, 100, glob.GateRadius, config.GateColor, "CNOT", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 0}}, 2)
			CNOT2 := components.NewGate(100, 100, glob.GateRadius, config.GateColor, "CNOT", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 0}}, 2)

			M1 := components.NewMeasurementGate(100, 100, glob.GateRadius, config.GateColor, "M")
			M2 := components.NewMeasurementGate(100, 100, glob.GateRadius, config.GateColor, "M")

			CX := components.NewGate(100, 100, glob.GateRadius, config.GateColor, "CX", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 0}}, 2)
			CZ := components.NewGate(100, 100, glob.GateRadius, config.GateColor, "CZ", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, -1}}, 2)

			p.PushComponent(H1, H2, H3, CNOT1, CNOT2, M1, M2, CX, CZ)
		}

		create(panel)

		rows := []components.InfoRow{
			{"Speed and Something that burn my retina", 0.75, 3 + 4i},
			{"Power", 0.2, -1 + 2i},
		}
		table := components.NewInfoTable(400, 200, 260, 100, config.ColorBg, rows)

		t1 := components.NewInfoTable(400, 200, 260, 40, config.ColorBg, []components.InfoRow{})
		t2 := components.NewInfoTable(400, 200, 260, 40, config.ColorBg, []components.InfoRow{})
		t3 := components.NewInfoTable(400, 200, 260, 40, config.ColorBg, []components.InfoRow{})
		t4 := components.NewInfoTable(400, 200, 260, 40, config.ColorBg, []components.InfoRow{})
		t5 := components.NewInfoTable(400, 200, 260, 40, config.ColorBg, []components.InfoRow{})
		t6 := components.NewInfoTable(400, 200, 260, 40, config.ColorBg, []components.InfoRow{})

		testSource := components.NewSourceGate(300, 300, 100, config.GateColor, "Test", []complex64{0, 1}, 4)
		panel.PushComponent(testSource)

		panel.PushComponent(table, t1, t2, t3, t4, t5, t6)\
	*/

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
	toolBar.AddEffect(func() {
		effect.ScaleWidthToScreen(toolBar)
		toolBar.Height = 50
		effect.PinToBottom(toolBar, 50)
	})
	toolBar.IsSpawnAllow(false)
	toolBar.SetPriority(100)
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

	ToolButLoad := components.NewButton(-250, 0, 100, 25, rl.LightGray, "Load", 20,
		func() {
			winManager = append(winManager, buildFileBrowser())
		})

	ToolButSave := components.NewButton(-150, 0, 100, 25, rl.LightGray, "Save", 20,
		func() {
			fileWin := windows.NewRenderWindow(650, 100, 400, 50)
			fileWin.Rename("Save as")
			fileWin.SetPriority(200)
			fileWin.IsSpawnAllow(false)
			fileWin.IsPanAllow(false)
			fileWin.IsZoomAllow(false)
			fileWin.IsResizeAllow(false)
			fileWin.ShowGrid = false

			input := components.NewInput(0, 0, 400, 25, 16, 64, func(text string) {
				os.MkdirAll("saves", 0755)

				res := SaveState()

				os.WriteFile(filepath.Join("saves", text), []byte(res), 0644)
				DeleteWindowByID(fileWin.GetID())
			})
			input.SetText("untitled.qsim")
			fileWin.PushComponent(input)
			fileWin.PinCamera(func() rl.Vector2 { return rl.Vector2{X: 0, Y: 0} })
			winManager = append(winManager, fileWin)
		})

	ToolButReset := components.NewButton(-50, 0, 100, 25, rl.LightGray, "Reset", 20,
		func() {
			// Clear every component from the circuit panel.
			animation.Reset()
			elems := panel.GetElement()
			ids := make([]int32, 0, len(elems))
			for _, c := range elems {
				ids = append(ids, c.GetID())
			}
			panel.DeleteChildWithID(ids...)
		})

	toolBar.PushComponent(ToolBut1, ToolBut2, ToolBut3, ToolBut4, ToolBut5, ToolButLoad, ToolButSave, ToolButReset)

	spawnBar := windows.NewRenderWindow(1500, 0, 100, 800)
	spawnBar.IsResizeAllow(false)
	spawnBar.IsPanAllow(false)
	spawnBar.IsZoomAllow(false)
	spawnBar.IsDragAllow(true)
	spawnBar.IsSpawnAllow(false)
	spawnBar.ShowGrid = false
	spawnBar.SetPriority(100)
	spawnBar.AddEffect(func() {
		effect.ConstSize(spawnBar, 100, 800)
		effect.PinToSide(spawnBar)
	})
	// Keep the buttons at their designed screen positions when the bar is
	// clamped below its 800px design height, so the top buttons don't slide
	// under the title bar and get clipped. Scroll input is preserved like the
	// file browser: the wheel moves the camera and the pin only clamps it
	// back to the design position when scrolled past the top.
	spawnBar.PinCamera(func() rl.Vector2 {
		base := (float32(spawnBar.Height) - 800) / 2
		y := spawnBar.Camera.Target.Y
		if y < base {
			y = base
		}
		return rl.Vector2{X: 0, Y: y}
	})
	spawnBar.IsVerticalScrollAllow(true)
	spawnBar.Rename("Object")

	butByState := map[glob.SpawnType]*components.ToggleButton{}
	spawnBut := func(label string, fontSize int32, state glob.SpawnType) *components.ToggleButton {
		b := components.NewToggleButton(0, 0, 50, 50, rl.LightGray, label,
			func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(state) }, fontSize,
			func() {
				utils.SetMouseState(glob.MouseStateSpawn)
				utils.SetSpawnState(state)
			},
			func() {
				utils.ToggleMouseState(glob.MouseStateSpawn)
				utils.ToggleSpawnState(state)
			})
		// Activate the spawn mode at press time: the button stays held down
		// and the ghost preview follows the mouse while dragging to the
		// canvas, just like the click-to-select flow.
		b.OnPress = func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(state)
		}
		// Drag the button out of the bar and release over a circuit panel to
		// spawn that component at the drop point.
		b.OnDragOut = func(screenPos rl.Vector2) {
			spawnObjectAt(screenPos, state)
		}
		butByState[state] = b
		return b
	}

	// Spawn buttons, laid out in two columns and grouped by kind: quantum
	// sources, quantum gates, measurement, system tools, classical logic,
	// then drawing tools. Groups are separated by a small gap.
	spawnBtns := []components.Component{}
	row := float32(-370)
	addRow := func(left, right *components.ToggleButton) {
		left.Center = rl.Vector2{X: -25, Y: row}
		spawnBtns = append(spawnBtns, left)
		if right != nil {
			right.Center = rl.Vector2{X: 25, Y: row}
			spawnBtns = append(spawnBtns, right)
		}
		row += 50
	}
	gap := func() { row += 10 }

	// Quantum sources
	addRow(spawnBut("Q", 40, glob.Qubit), spawnBut("GQ", 30, glob.GQubit))
	gap()

	// Quantum gates
	addRow(spawnBut("H", 40, glob.Hadamard), spawnBut("X", 40, glob.X))
	addRow(spawnBut("Y", 40, glob.Y), spawnBut("Z", 40, glob.Z))
	addRow(spawnBut("CX", 30, glob.CX), spawnBut("CY", 30, glob.CY))
	addRow(spawnBut("CZ", 30, glob.CZ), spawnBut("cX", 30, glob.CBitX))
	addRow(spawnBut("cY", 30, glob.CBitY), spawnBut("cZ", 30, glob.CBitZ))
	gap()

	// Measurement
	addRow(spawnBut("M", 30, glob.Measurement), spawnBut("M2", 30, glob.Measure2))
	addRow(spawnBut("M3", 20, glob.Measure3), spawnBut("M4", 20, glob.Measure4))
	gap()

	// System tools
	addRow(spawnBut("CP", 30, glob.Copy), spawnBut("==", 20, glob.Compare))
	addRow(spawnBut("I", 30, glob.Info), spawnBut("U", 30, glob.ArbGate))
	gap()

	// Classical logic
	addRow(spawnBut("0/1", 20, glob.LogicButton), spawnBut("!", 30, glob.LogicNot))
	addRow(spawnBut("&", 30, glob.LogicAnd), spawnBut("|", 30, glob.LogicOr))
	addRow(spawnBut("LT", 20, glob.Light), nil)
	gap()

	// Drawing tools
	addRow(spawnBut("T", 40, glob.TextBox), spawnBut("L", 40, glob.LineDraw))

	spawnBar.PushComponent(spawnBtns...)

	animBar := windows.NewRenderWindow(0, 850, 1600, 50)
	animBar.IsResizeAllow(false)
	animBar.IsPanAllow(false)
	animBar.IsZoomAllow(false)
	animBar.IsDragAllow(true)
	animBar.IsSpawnAllow(false)
	animBar.ShowGrid = false
	animBar.SetPriority(100)
	animBar.EnableTitleBar(false)
	animBar.Rename("Animation")
	animBar.AddEffect(func() {
		effect.ScaleWidthToScreen(animBar)
		effect.PinToBottom(animBar, 0)
	})

	resetAnim := func() {
		animation.Anim.CurrentIdx = 0
		animation.Anim.SubIdx = 0
		animation.Anim.Progress = 0
		animation.Anim.State = animation.StatePaused
	}
	startPlay := func() {
		if animation.IsFinished() || len(animation.Anim.Gates) == 0 {
			animation.Reset()
			animation.Anim.Gates = animation.GenerateSteps(panel.WComp)
			if len(animation.Anim.Gates) == 0 {
				return
			}
		}
		animation.Anim.State = animation.StatePlaying
	}
	goBack := func() {
		if animation.Anim.CurrentIdx == 0 && animation.Anim.SubIdx == 0 {
			animation.Anim.Progress = 0
			animation.Anim.State = animation.StatePaused
			return
		}
		if animation.Anim.SubIdx > 0 {
			animation.Anim.SubIdx--
		} else {
			animation.Anim.CurrentIdx--
			ga := animation.CurrentAnim()
			if ga != nil {
				animation.Anim.SubIdx = len(ga.Inputs) + 1
			}
		}
		animation.Anim.Progress = 0
		animation.Anim.State = animation.StatePaused
	}
	skipStep := func() {
		if animation.IsFinished() {
			return
		}
		ga := animation.CurrentAnim()
		if ga == nil {
			return
		}
		maxSub := len(ga.Inputs) + 1
		if animation.Anim.SubIdx < maxSub {
			animation.Anim.SubIdx++
		} else {
			animation.Anim.SubIdx = 0
			animation.Anim.CurrentIdx++
		}
		animation.Anim.Progress = 0
		if animation.IsFinished() {
			animation.Anim.State = animation.StateIdle
		} else {
			animation.Anim.State = animation.StatePaused
		}
	}
	finishAnim := func() {
		if len(animation.Anim.Gates) > 0 {
			animation.Anim.CurrentIdx = len(animation.Anim.Gates)
			animation.Anim.Progress = 0
			animation.Anim.State = animation.StateIdle
		}
	}

	AnimButReset := components.NewButton(-200, 0, 70, 25, rl.LightGray, "|<", 20, resetAnim)
	AnimButBack := components.NewButton(-130, 0, 70, 25, rl.LightGray, "<", 20, goBack)
	AnimButPlay := components.NewToggleButton(-60, 0, 70, 25, rl.LightGray, ">",
		func() bool { return animation.Anim.State == animation.StatePlaying }, 20,
		startPlay,
		func() { animation.Anim.State = animation.StatePaused })
	AnimButSkip := components.NewButton(10, 0, 70, 25, rl.LightGray, ">>", 20, skipStep)
	AnimButEnd := components.NewButton(80, 0, 70, 25, rl.LightGray, ">|", 20, finishAnim)

	stepLabel := components.NewLabel(170, -7, "Gate: 0/0", 16, rl.White)

	// Timeline scrubber: drag to set the animation time (back and forth).
	scrubbing := false
	wasPlaying := false
	timeline := components.NewSlider(590, 2, 360, 10, rl.LightGray,
		func(v float32) {
			if !scrubbing {
				scrubbing = true
				wasPlaying = animation.Anim.State == animation.StatePlaying
			}
			if len(animation.Anim.Gates) == 0 {
				animation.Anim.Gates = animation.GenerateSteps(panel.WComp)
				if len(animation.Anim.Gates) == 0 {
					return
				}
			}
			animation.SetTime(float64(v) * animation.TotalDuration())
			if !animation.IsFinished() {
				animation.Anim.State = animation.StatePaused
			}
		},
		func(v float32) {
			if scrubbing && wasPlaying && !animation.IsFinished() {
				animation.Anim.State = animation.StatePlaying
			}
			scrubbing = false
		},
	)

	animBar.PushComponent(AnimButReset, AnimButBack, AnimButPlay, AnimButSkip, AnimButEnd, stepLabel, timeline)

	// Hover tooltips for the bar buttons (shown by the passive tooltip window).
	ToolBut1.Tooltip = "Normal mode: drag components, right-click a qubit to rename"
	ToolBut2.Tooltip = "Detach mode: click a hook or qubit to disconnect it"
	ToolBut3.Tooltip = "Fix mode: click a component to pin/unpin it"
	ToolBut4.Tooltip = "Spawn mode: click in the circuit to place the selected object"
	ToolBut5.Tooltip = "Erase mode: click a component to delete it"
	ToolButLoad.Tooltip = "Load a saved circuit"
	ToolButSave.Tooltip = "Save the circuit to saves/"

	butByState[glob.Qubit].Tooltip = "Qubit system: a |0> qubit with its state grid"
	butByState[glob.Hadamard].Tooltip = "Hadamard gate (1 qubit)"
	butByState[glob.X].Tooltip = "Pauli-X / NOT gate (1 qubit)"
	butByState[glob.Y].Tooltip = "Pauli-Y gate (1 qubit)"
	butByState[glob.Z].Tooltip = "Pauli-Z gate (1 qubit)"
	butByState[glob.CX].Tooltip = "Controlled-X / CNOT gate (2 qubits)"
	butByState[glob.CY].Tooltip = "Controlled-Y gate (2 qubits)"
	butByState[glob.CZ].Tooltip = "Controlled-Z gate (2 qubits)"
	butByState[glob.CBitX].Tooltip = "Bit-controlled X: applies X to the qubit while the control logical bit is 1"
	butByState[glob.CBitY].Tooltip = "Bit-controlled Y: applies Y to the qubit while the control logical bit is 1"
	butByState[glob.CBitZ].Tooltip = "Bit-controlled Z: applies Z to the qubit while the control logical bit is 1"
	butByState[glob.ArbGate].Tooltip = "Arbitrary gate: right-click to rename, then edit size and the matrix table; non-unitary matrices are auto-fixed on close; hover to preview"
	butByState[glob.Measurement].Tooltip = "Measurement gate: |0> and |1> outcome branches"
	butByState[glob.Measure2].Tooltip = "Collapse measurement: outputs the outcome as a logical bit plus the remaining state; click to force 0/1"
	butByState[glob.Measure3].Tooltip = "M3 measurement: like M2, outputs the collapsed qubit and the remaining state"
	butByState[glob.Info].Tooltip = "Info table: amplitudes of a hooked system"
	butByState[glob.GQubit].Tooltip = "Source gate: continuously emits a custom qubit state"
	butByState[glob.Copy].Tooltip = "Copy gate: create a copy of a hooked qubit system"
	butByState[glob.Compare].Tooltip = "Compare gate: outputs 1 if two qubit systems are equal, 0 otherwise"
	butByState[glob.LogicButton].Tooltip = "Logic button: click to toggle the output bit 0/1"
	butByState[glob.LogicNot].Tooltip = "NOT gate: inverts the input bit"
	butByState[glob.LogicAnd].Tooltip = "AND gate: outputs 1 when both inputs are 1"
	butByState[glob.LogicOr].Tooltip = "OR gate: outputs 1 when at least one input is 1"
	butByState[glob.Light].Tooltip = "Light: shines while the connected bit is 1"
	butByState[glob.TextBox].Tooltip = "Text box"
	butByState[glob.LineDraw].Tooltip = "Line draw"

	AnimButReset.Tooltip = "Reset animation"
	AnimButBack.Tooltip = "Previous step"
	AnimButPlay.Tooltip = "Play / pause the gate animation"
	AnimButSkip.Tooltip = "Skip to next step"
	AnimButEnd.Tooltip = "Jump to the end"

	winManager = append(winManager, panel, toolBar, spawnBar, animBar, windows.NewTooltipWindow())

	update := func() {
		// Update main window resize
		glob.Refresh()

		UpdateMainWindow()

		// Update inner panels
		for _, d := range winManager {
			d.Update()
		}

		animation.Update()

		if animation.Anim.State == animation.StatePlaying {
			AnimButPlay.Label = "||"
		} else {
			AnimButPlay.Label = ">"
		}

		gateNum := animation.Anim.CurrentIdx + 1
		if gateNum > len(animation.Anim.Gates) {
			gateNum = len(animation.Anim.Gates)
		}
		total := animation.TotalDuration()
		stepLabel.Text = fmt.Sprintf("Gate: %d/%d   %.1f / %.1f s", gateNum, len(animation.Anim.Gates), animation.CurrentTime(), total)
		if !scrubbing && total > 0 {
			timeline.Value = float32(animation.CurrentTime() / total)
			ticks := []float32{}
			acc := 0.0
			for i := range animation.Anim.Gates {
				acc += animation.GateDuration(&animation.Anim.Gates[i])
				if acc < total {
					ticks = append(ticks, float32(acc/total))
				}
			}
			timeline.Ticks = ticks
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

	runMainLoop(update)

	//fmt.Println(toolBar.Camera.Target)
}
