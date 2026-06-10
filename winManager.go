package main

type pWindow interface {
	Draw()
	Update()
	PostUpdate()
	IsActive() bool
	SetActive(bool)
}

var winManager []pWindow

func shuffleWinManager() {
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
