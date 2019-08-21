package main

import (
	"image/png"
	"os"

	"github.com/darkautism/gseprite"
)

func main() {
	g, _ := gseprite.LoadAseprite("idle outline.ase")
	img := g.Frames[0].Render()
	f, err := os.Create("img.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	png.Encode(f, img)
}
