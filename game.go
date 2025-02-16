package main

import (
	"image"
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hibooboo2/slime_tag/ecs"
)

type Game struct {
	sprites           *ebiten.Image
	entities          *ecs.Entities
	menuOptions       []string
	resolutionOptions []string
	selected          int
	inMainMenu        bool
	keys              *Keys
	exit              bool
	player            *Player
	debugLogs         []string
	lastLogTime       time.Time
	settings          bool
	bullets           *ecs.Entities
	frameCount        int // Add a frame counter
	gamepads          []ebiten.GamepadID
	enemiesKilled     int           // Add a field to track the number of enemies killed
	gameOver          bool          // Add a field to track if the game is over
	lastEnemySpawn    time.Time     // Track when we last spawned an enemy
	spawnInterval     time.Duration // Current interval between enemy spawns
	scaleFactor       float32       // Add this line to define scaleFactor
	lastPowerUpSpawn  time.Time     // Add this line to track the last power-up spawn time
	floatingTexts     []*FloatingText
	settingsOptions   []string
	settingsSelected  int
	debugMode         bool
	showResolutions   bool // Whether to show resolution popup
	resolutionIdx     int  // Currently selected resolution

	headstonesCollected int // Add this line to track collected headstones
}

func (g *Game) checkEnemyBulletCollisions() {
	//XXX FIX THIS ITS BUGGED
	for b := range *g.bullets {
		bullet, ok := (*g.bullets)[b].(*Bullet)
		if !ok || bullet.collided {
			continue
		}
		for e := range *g.entities {
			enemy, ok := (*g.entities)[e].(*Enemy)
			if !ok {
				continue
			}
			if enemy.hpBar.currentHP < 1 {
				continue
			}
			// Define the enemy's bounding rectangle

			enemyRect := enemy.getRect()
			// Check if the bullet is within the enemy's rectangle
			// The enemy is hit by bullets here wihch are a radius of 3px
			if bullet.Overlaps(enemyRect, g) {
				// Collision detected
				enemy.hpBar.AddHP(-20)
				if enemy.hpBar.currentHP <= 0 {
					g.enemiesKilled++                                                              // Increment the enemies killed count
					g.AddFloatingText("+5 HP", g.player.x, g.player.y, color.RGBA{0, 255, 0, 255}) // Add healing text
					g.player.hpBar.AddHP(5)

					// Reduce spawn interval by 15ms per kill, with a minimum of 100ms
					g.spawnInterval -= 15 * time.Millisecond
					if g.spawnInterval < 100*time.Millisecond {
						g.spawnInterval = 100 * time.Millisecond // Ensure a minimum spawn interval
					}

					// Spawn a power-up every 5 enemies killed
					if g.enemiesKilled%5 == 0 {
						g.entities.Add(NewRandomPowerUp())
					}
				}
				break
			}

		}
	}
}

func (g *Game) checkPlayerCollisions() {
	for _, entitiy := range *g.entities {
		if e, ok := entitiy.(ecs.Overlapper); ok {
			// Define the player's bounding rectangle
			playerRect := image.Rect(
				int(g.player.x)+5, int(g.player.y)+5,
				int(g.player.x)+20, int(g.player.y)+20,
			)
			// Check for collision
			if e.Overlaps(playerRect, g) {
				log.Printf("Player overlapped with object.")
			}
		}
	}
}
