package main

import (
	"image"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/hibooboo2/slime_tag/ecs"
)

type Bullet struct {
	x, y         float32
	angle        float32
	speed        float32
	creationTime time.Time // Add creationTime to track bullet age
	hp           int
	collided     bool
	radius       float32
}

var _ ecs.Overlapper = &Bullet{}
var _ ecs.Drawer = &Bullet{}

func (b *Bullet) Remove() bool {
	return time.Since(b.creationTime) > time.Second*5 || b.collided
}

func (b *Bullet) Draw(screen *ebiten.Image, game ecs.Game) {
	// g := game.(*Game)
	vector.DrawFilledCircle(screen, b.x, b.y, float32(b.radius), color.RGBA{255, 255, 255, 255}, true) // White circle for bullets
}

func (b *Bullet) Overlaps(r image.Rectangle, game ecs.Game) bool {
	if b.collided {
		return false
	}
	// g := game.(*Game)
	ballRect := image.Rect(int(b.x-b.radius), int(b.y-b.radius), int(b.x+b.radius), int(b.y+b.radius))

	if ballRect.Overlaps(r) {
		b.collided = true
		return true
	}

	return false
}
