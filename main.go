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

func main() {
	InitMainWindow(1600, 900, "Floating Panels")

	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	// Create one or more draggable inner panels

	//panel := windows.NewWindow(200, 150, 400, 300)
	panel := windows.NewRenderWindow(0, 0, 1500, 800)
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
				if !e.IsDir() && filepath.Ext(e.Name()) == ".qsim" {
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
								idx := -1
								for i, c := range fw.GetElement() {
									if b, ok := c.(*components.Button); ok && b.Label == f {
										idx = i
										break
									}
								}
								if idx >= 0 {
									fw.RemoveComponent(idx + 1)
									fw.RemoveComponent(idx)
								}
							}
						}(name, fileWin))
					comps = append(comps, xBtn)
				}
			}

			fileWin.IsPanAllow(false)
			fileWin.IsZoomAllow(false)
			fileWin.IsVerticalScrollAllow(true)
			fileWin.PushComponent(comps...)
			fileWin.AddEffect(func() { effect.ScaleButtonsToWidth(fileWin) })
			fileWin.PinCamera(func() rl.Vector2 { return rl.Vector2{X: 0, Y: float32(math.Max(0, float64(fileWin.Camera.Target.Y)))} })
			winManager = append(winManager, fileWin)
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

	toolBar.PushComponent(ToolBut1, ToolBut2, ToolBut3, ToolBut4, ToolBut5, ToolButLoad, ToolButSave)

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
		effect.PinToRight(spawnBar)
	})
	// Keep the buttons at their designed screen positions when the bar is
	// clamped below its 800px design height, so the top buttons don't slide
	// under the title bar and get clipped.
	spawnBar.PinCamera(func() rl.Vector2 {
		return rl.Vector2{X: 0, Y: (float32(spawnBar.Height) - 800) / 2}
	})
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

	SpawnBut10 := components.NewToggleButton(25, -162.5, 50, 50, rl.LightGray, "I",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.Info) }, 30,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.Info)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.Info)
		})

	SpawnBut11 := components.NewToggleButton(-25, -112.5, 50, 50, rl.LightGray, "GQ",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.GQubit) }, 30,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.GQubit)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.GQubit)
		})

	SpawnBut12 := components.NewToggleButton(25, -112.5, 50, 50, rl.LightGray, "T",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.TextBox) }, 40,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.TextBox)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.TextBox)
		})

	SpawnBut13 := components.NewToggleButton(-25, -62.5, 50, 50, rl.LightGray, "L",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.LineDraw) }, 40,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.LineDraw)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.LineDraw)
		})

	SpawnBut14 := components.NewToggleButton(25, -62.5, 50, 50, rl.LightGray, "M2",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.Measure2) }, 30,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.Measure2)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.Measure2)
		})

	SpawnBut15 := components.NewToggleButton(-25, -12.5, 50, 50, rl.LightGray, "CP",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.Copy) }, 30,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.Copy)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.Copy)
		})

	SpawnBut16 := components.NewToggleButton(25, -12.5, 50, 50, rl.LightGray, "==",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.Compare) }, 20,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.Compare)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.Compare)
		})

	SpawnBut17 := components.NewToggleButton(-25, 37.5, 50, 50, rl.LightGray, "0/1",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.LogicButton) }, 20,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.LogicButton)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.LogicButton)
		})

	SpawnBut18 := components.NewToggleButton(25, 37.5, 50, 50, rl.LightGray, "!",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.LogicNot) }, 30,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.LogicNot)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.LogicNot)
		})

	SpawnBut19 := components.NewToggleButton(-25, 87.5, 50, 50, rl.LightGray, "&",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.LogicAnd) }, 30,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.LogicAnd)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.LogicAnd)
		})

	SpawnBut20 := components.NewToggleButton(25, 87.5, 50, 50, rl.LightGray, "|",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.LogicOr) }, 30,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.LogicOr)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.LogicOr)
		})

	SpawnBut21 := components.NewToggleButton(-25, 137.5, 50, 50, rl.LightGray, "LT",
		func() bool { return utils.IsMouseState(glob.MouseStateSpawn) && utils.IsSpawnState(glob.Light) }, 20,
		func() {
			utils.SetMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.Light)
		},
		func() {
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.ToggleSpawnState(glob.Light)
		})

	spawnBar.PushComponent(SpawnBut1, SpawnBut2, SpawnBut3, SpawnBut4, SpawnBut5, SpawnBut6, SpawnBut7, SpawnBut8, SpawnBut9, SpawnBut10, SpawnBut11, SpawnBut12, SpawnBut13, SpawnBut14, SpawnBut15, SpawnBut16, SpawnBut17, SpawnBut18, SpawnBut19, SpawnBut20, SpawnBut21)

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

	SpawnBut1.Tooltip = "Qubit system: a |0> qubit with its state grid"
	SpawnBut2.Tooltip = "Hadamard gate (1 qubit)"
	SpawnBut3.Tooltip = "Pauli-X / NOT gate (1 qubit)"
	SpawnBut4.Tooltip = "Pauli-Y gate (1 qubit)"
	SpawnBut5.Tooltip = "Pauli-Z gate (1 qubit)"
	SpawnBut6.Tooltip = "Controlled-X / CNOT gate (2 qubits)"
	SpawnBut7.Tooltip = "Controlled-Y gate (2 qubits)"
	SpawnBut8.Tooltip = "Controlled-Z gate (2 qubits)"
	SpawnBut9.Tooltip = "Measurement gate: |0> and |1> outcome branches"
	SpawnBut10.Tooltip = "Info table: amplitudes of a hooked system"
	SpawnBut11.Tooltip = "Source gate: continuously emits a custom qubit state"
	SpawnBut12.Tooltip = "Text box"
	SpawnBut13.Tooltip = "Line draw"
	SpawnBut14.Tooltip = "Collapse measurement: outputs the measured qubit and the remaining state; click to force 0/1"
	SpawnBut15.Tooltip = "Copy gate: create a logical copy of a hooked qubit system"
	SpawnBut16.Tooltip = "Compare gate: outputs 1 if two qubit systems are equal, 0 otherwise"

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
