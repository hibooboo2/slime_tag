package main

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hibooboo2/slime_tag/ecs"
)

func NewRandomPowerUp() *PowerUp {
	p := &PowerUp{
		x:         float64(rand.Intn(screenWidth)),
		y:         float64(rand.Intn(screenHeight)),
		spawnTime: time.Now(),
		bonus:     rand.Intn(2),
	}
	switch p.bonus {
	case 0:
		p.icon = NewSprite(fmt.Sprintf("resources/slimes/PNG/%[1]s/Idle/%[1]s_Idle_full.png", "Slime3"), 6)
	case 1:
		p.icon = NewSprite(fmt.Sprintf("resources/slimes/PNG/%[1]s/Attack/%[1]s_Attack_full.png", "Slime3"), 10)
	}
	return p
}

type PowerUp struct {
	icon      *AnimatedSprite
	x, y      float64
	spawnTime time.Time
	bonus     int
	collided  bool
}

var _ ecs.Drawer = &PowerUp{}
var _ ecs.Overlapper = &PowerUp{}

func (pu *PowerUp) getRect() image.Rectangle {
	return image.Rect(
		int(pu.x)-20, int(pu.y)-20,
		int(pu.x)+20, int(pu.y)+20,
	)
}

func (pu *PowerUp) Overlaps(r image.Rectangle, game ecs.Game) bool {
	g := game.(*Game)
	if pu.collided {
		return true
	}

	pu.collided = pu.getRect().Overlaps(r)

	if !pu.collided {
		return false
	}

	switch pu.bonus {
	case 0:
		g.player.hpBar.AddHP(20)
		g.AddFloatingText("+20 HP", float64(pu.x), float64(pu.y), color.RGBA{0, 255, 0, 255})
	case 1:
		g.player.hpBar.AddHP(-10)
		g.AddFloatingText("-10 HP", float64(pu.x), float64(pu.y), color.RGBA{255, 0, 0, 255})
	}

	return true
}

func (pu *PowerUp) Draw(screen *ebiten.Image, game ecs.Game) {
	g := game.(*Game)
	// Draw the power-up icon at its location
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(pu.x, pu.y)
	_, img := pu.icon.GetCurrentSprite(g.frameCount, 0)
	screen.DrawImage(img, op)

}

func (pu *PowerUp) Remove() bool {
	return time.Since(pu.spawnTime) > time.Second*10 || pu.collided
}
