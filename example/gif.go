package main

import (
	"image/gif"
	"os"

	"github.com/darkautism/gseprite"
)

func main() {
	g, _ := gseprite.LoadAseprite("idle outline.ase")
	img := g.GIF()
	f, err := os.Create("img.gif")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	gif.EncodeAll(f, &img)
}
