package components

import (
	"qsim/config"
	glob "qsim/globals"
	"qsim/utils"
	"strconv"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type DecomposeGate struct {
	Gate
}

func NewDecomposeGate(x, y, radius float32, color rl.Color, label string) *DecomposeGate {
	tmp := DecomposeGate{
		Gate{
			Circle:      *NewCircle(x, y, radius, color),
			Label:       label,
			InputCount:  1,
			OutputCount: 0,
		},
	}
	tmp.ID = utils.GenerateID(&tmp)
	tmp.SetWeight(glob.GateWeight)
	vertSpacing := glob.HookRadius * 1.5
	for i := 0; i < int(tmp.InputCount); i++ {
		offY := (float32(i) - float32(tmp.InputCount-1)/2) * vertSpacing
		newHook := NewHook(x-glob.GateToHookDist, y+offY, glob.HookRadius, config.HookColor)
		newHook.Label = "I" + strconv.Itoa(i)
		tmp.HookList = append(tmp.HookList, newHook)
	}
	for i := 0; i < int(tmp.OutputCount); i++ {
		offY := (float32(i) - float32(tmp.OutputCount-1)/2) * vertSpacing
		outputHook := NewOutputHook(x+glob.GateToHookDist, y+offY, glob.OutputHookRadius, config.OutputHookColor)
		tmp.HookList = append(tmp.HookList, outputHook)
		tmp.OutPutHook = append(tmp.OutPutHook, outputHook)
		tmp.OutPutHook[i].Label = "O" + strconv.Itoa(i)
		tmp.OutPutHook[i].AllowQubitSystem = true
	}
	return &tmp
}

func (c *DecomposeGate) CalculateOutPut() {

}

func (c *DecomposeGate) GetID() int32 {
	return c.ID
}
