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

	rect := image.Rect(hs.spriteX*hs.size, hs.spriteY*hs.size, (hs.spriteX*hs.size)+hs.size, (hs.spriteY*hs.size)+hs.size)

	img, ok := g.spritesCache[rect]
	if !ok {
		img = g.sprites.SubImage(rect).(*ebiten.Image)
		g.spritesCache[rect] = img
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(hs.locX), float64(hs.locY))
	screen.DrawImage(img, op)
}

func (hs *Sprite) Remove() bool {
	return false
}
