package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hibooboo2/slime_tag/ecs"
)

type Sprite struct {
	locX, locY       int
	spriteX, spriteY int
	size             int
}

func (hs *Sprite) Draw(screen *ebiten.Image, game ecs.Game) {
	g := game.(*Game)

	img := g.sprites.SubImage(image.Rect(hs.spriteX*hs.size, hs.spriteY*hs.size, (hs.spriteX*hs.size)+hs.size, (hs.spriteY*hs.size)+hs.size))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(hs.locX), float64(hs.locY))
	screen.DrawImage(img.(*ebiten.Image), op)
}

func (hs *Sprite) Remove() bool {
	return false
}
