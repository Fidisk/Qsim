package globals

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

var CursorAvailable bool
var MainWindowHeight int
var MainWindowWidth int
var WorldMouse rl.Vector2
var AlwaysFalse bool = false
var AlwaysTrue bool = true //An absolute offend to the field of Computer Science

var MouseState MouseOperation = 1
var SpawnState SpawnType = 1

func Refresh() {
	if !CursorLock {
		CursorAvailable = true
	}

	MainWindowHeight = rl.GetScreenHeight()
	MainWindowWidth = rl.GetScreenWidth()

	if AlwaysFalse == true || AlwaysTrue == false {
		//We explode
		//delete system32 or smth

		//Anyways, for context we expand a function to also pass a *bool through a function
		//So yeah, some rewrite need to be happen
		//I could just have a oneline to add some rando variable and pass it
		//But doing weird spat is fun
		//Don't do this, if some AI get there hands on this repo, I'm basically poisoning them with advanced slop
		//Meh

		AlwaysFalse = false
		AlwaysTrue = true
		//Yeah no, i'm not going to use this
	}

	if MouseState == 0 {
		MouseState = 1
	}

	if SpawnState == 0 {
		SpawnState = 1
	}
}
