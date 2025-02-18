package main

import (
	"image"
	"log"
	"time"

	"github.com/hibooboo2/slime_tag/ecs"
	"github.com/hibooboo2/slime_tag/sound"
)

type HeadStone struct {
	Sprite
	deathTime          time.Time
	collidedWithPlayer bool
}

var _ ecs.Drawer = &HeadStone{}
var _ ecs.Overlapper = &HeadStone{}

func (hs *HeadStone) Remove() bool {
	if hs.collidedWithPlayer || time.Since(hs.deathTime) >= 5*time.Second {
		log.Printf("Player headstone removal. Collision: %t", hs.collidedWithPlayer)
		return true
	}
	return false
}

func (hs *HeadStone) Overlaps(r image.Rectangle, game ecs.Game) bool {
	g := game.(*Game)
	if hs.collidedWithPlayer {
		log.Printf("Headstone already collided: %t", hs.collidedWithPlayer)
		return true
	}

	// Define the headstone's bounding rectangle
	headstoneRect := image.Rect(
		hs.locX, hs.locY,
		hs.locX+hs.size, hs.locY+hs.size,
	)
	hs.collidedWithPlayer = headstoneRect.Overlaps(r)
	if hs.collidedWithPlayer {
		g.player.headstonesCollected += 1 // Increment headstones collected
		sound.Play("headstone")
		log.Printf("Headstone collided: %t adding point", hs.collidedWithPlayer)
		log.Printf("Headstones collected: %d", g.player.headstonesCollected)
		if g.player.headstonesCollected%10 == 0 {
			g.player.headstonePowerup = time.Now()
		}
	}

	return hs.collidedWithPlayer
}
