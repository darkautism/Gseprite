# Gseprite

A simple program which can read ase file easily.

[GoDoc](https://godoc.org/github.com/darkautism/Gseprite)

## Work with ebiten example

![ebiten example](example.gif)

```
package main

import (
	"log"

	gseprite "github.com/darkautism/Gseprite"
	"github.com/hajimehoshi/ebiten/v2"
)

// Game implements ebiten.Game interface.
type Game struct {
	player *Sprites
}

// Update proceeds the game state.
// Update is called every tick (1/60 [s] by default).
func (g *Game) Update() error {
	// Write your game's logical update.
	return nil
}

type Sprites struct {
	player *gseprite.Gseprite
}

func (s *Sprites) Play(screen *ebiten.Image) {
	afps := ebiten.ActualFPS()
	if afps == 0 {
		afps = 60
	}
	screen.DrawImage(ebiten.NewImageFromImage(s.player.SpritesRender(float64(1000)/afps)), nil)
}

// Draw draws the game screen.
// Draw is called every frame (typically 1/60[s] for 60Hz display).
func (g *Game) Draw(screen *ebiten.Image) {

	g.player.Play(screen)
	// Write your game's rendering.
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
// If you don't have to adjust the screen size with the outside size, just return a fixed size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 320, 240
}

func main() {
	game := &Game{}
	sp, _ := gseprite.LoadAseprite("idle outline.ase")
	game.player = &Sprites{player: sp}
	// Specify the window size as you like. Here, a doubled size is specified.
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Quagsire")
	// Call ebiten.RunGame to start your game loop.
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

```