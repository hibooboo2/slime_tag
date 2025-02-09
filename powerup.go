package main

import (
	"fmt"
	"math/rand"
	"time"
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
