i now have a extend struct 

package main

type renderWindow struct {
	Window
	wComp []*component
}

the window will call each of the component once for each type of its function

type component interface {
	Draw()
	Update()
	PostUpdate()
}

when the window draw, it call draw, and so on

It will pass a pointer to itself to the component when it do

Implement for me a component that is a blob, which is a soft body slime that i can drag using the window