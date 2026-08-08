package main

import (
	"qsim/utils"
	"qsim/windows"
	"sort"
)

type pWindow interface {
	Draw()
	Update()
	PostUpdate()
	IsActive() bool
	SetActive(bool)
	GetPriority() int32
	GetID() int32
	SaveState() string
}

var winManager []pWindow

// panel is the active circuit window (the loaded/saved circuit).
var panel *windows.RenderWindow

func shuffleWinManager() {
	// Stable sort keeps same-priority windows in their existing z-order
	// (unstable sort would reshuffle them every frame).
	sort.SliceStable(winManager, func(i, j int) bool {
		return winManager[i].GetPriority() > winManager[j].GetPriority()
	})

	// The window receiving the click is moved up, but only within its own
	// priority group: it never jumps above a window with higher priority.
	for i := max(0, len(winManager)-1); i > 0; i-- {
		if winManager[i].IsActive() && winManager[i-1].GetPriority() == winManager[i].GetPriority() {
			winManager[i-1], winManager[i] = winManager[i], winManager[i-1]
		}
		winManager[i].SetActive(false)
	}
	if len(winManager) == 0 {
		return
	}
	winManager[0].SetActive(false)
}

func DeleteWindowByID(id int32) {
	for i, w := range winManager {
		if w.GetID() == id {
			winManager = append(winManager[:i], winManager[i+1:]...)
			break
		}
	}
	utils.DeleteObjectWithID(id)
}

func SaveState() string {
	res := ""
	for _, w := range winManager {
		if rw, ok := w.(*windows.RenderWindow); ok && rw.CanSpawn {
			res += w.SaveState() + "\n"
		}
	}
	return res
}
