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

func shuffleWinManager() {
	sort.Slice(winManager, func(i, j int) bool {
		return winManager[i].GetPriority() > winManager[j].GetPriority()
	})

	for i := max(0, len(winManager)-1); i > 0; i-- {
		if winManager[i].IsActive() {
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
