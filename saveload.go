package main

import (
	"qsim/animation"
	"qsim/components"
	"qsim/utils"
	"qsim/windows"
)

// pendingShareState carries a share request from the Share button to the main
// loop, where the network call is actually executed (the web build requires
// JS interop on the main goroutine).
type pendingShareState struct {
	win    *windows.RenderWindow
	status *components.Label
}

// pendingShare is non-nil while a share upload is waiting to run.
var pendingShare *pendingShareState

// applyLoadedState replaces the current circuit panels with the windows
// decoded from a serialized circuit and returns the new active circuit panel
// (the last loaded RenderWindow, or nil if nothing was loaded).
func applyLoadedState(data string) *windows.RenderWindow {
	loaded := windows.LoadState(data)

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
	var panel *windows.RenderWindow
	for _, w := range loaded {
		if rw, ok := w.(*windows.RenderWindow); ok {
			winManager = append(winManager, rw)
			panel = rw
		}
	}
	return panel
}