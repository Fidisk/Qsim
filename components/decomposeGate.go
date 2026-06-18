package components

import (
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
	for i := 0; i < int(tmp.InputCount); i++ {
		newHook := NewHook(x, y, glob.HookRadius, glob.HookColor)
		newHook.Label = "I" + strconv.Itoa(i)
		tmp.HookList = append(tmp.HookList, newHook)
	}
	for i := 0; i < int(tmp.OutputCount); i++ {
		outputHook := NewOutputHook(x, y, glob.OutputHookRadius, glob.OutputHookColor)
		tmp.HookList = append(tmp.HookList, outputHook)
		tmp.OutPutHook = append(tmp.OutPutHook, outputHook)
		tmp.OutPutHook[0].Label = "O" + strconv.Itoa(i)
	}
	return &tmp
}

func (c *DecomposeGate) CalculateOutPut() {

}
