package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type AnimatedSprite struct {
	image       *ebiten.Image
	width       int
	frame       int
	frameWidth  int
	frameHeight int
}

type SpritePack struct {
	attack *AnimatedSprite
	death  *AnimatedSprite
	hurt   *AnimatedSprite
	idle   *AnimatedSprite
	run    *AnimatedSprite
	walk   *AnimatedSprite
}

type Player struct {
	playerType    string
	x, y          float64
	attacking     bool
	running       bool
	movementAngle float32
	spritePack    *SpritePack
	hp            int
}

func NewSprite(fileName string, width int) *AnimatedSprite {
	spriteImage, _, err := ebitenutil.NewImageFromFileSystem(slimes, fileName) // Load the sprite image
	if err != nil {
		panic(err)
	}

	return &AnimatedSprite{
		image:       spriteImage,
		width:       width,
		frameWidth:  64,
		frameHeight: 64,
	}
}

func NewSpritePack(slimeName string) *SpritePack {
	sp := &SpritePack{}

	sp.attack = NewSprite(fmt.Sprintf("slimes/PNG/%[1]s/Attack/%[1]s_Attack_full.png", slimeName), 10)
	sp.death = NewSprite(fmt.Sprintf("slimes/PNG/%[1]s/Death/%[1]s_Death_full.png", slimeName), 10)
	sp.hurt = NewSprite(fmt.Sprintf("slimes/PNG/%[1]s/Hurt/%[1]s_Hurt_full.png", slimeName), 5)
	sp.idle = NewSprite(fmt.Sprintf("slimes/PNG/%[1]s/Idle/%[1]s_Idle_full.png", slimeName), 6)
	sp.run = NewSprite(fmt.Sprintf("slimes/PNG/%[1]s/Run/%[1]s_Run_full.png", slimeName), 8)
	sp.walk = NewSprite(fmt.Sprintf("slimes/PNG/%[1]s/Walk/%[1]s_Walk_full.png", slimeName), 8)

	return sp
}

func NewPlayer(playerType string, spritePack *SpritePack) (*Player, error) {
	return &Player{
		playerType: playerType,
		x:          100, // Center of the screen horizontally
		y:          100, // Center of the screen vertically
		spritePack: spritePack,
	}, nil
}

func (sprite *AnimatedSprite) GetCurrentSprite(frameCount int, movementAngle float32) (bool, *ebiten.Image) {
	if frameCount%5 == 0 {
		sprite.frame++
	}

	x := (sprite.frame % sprite.width) * sprite.frameWidth

	// Determine direction based on movementAngle
	var direction int
	switch {
	case movementAngle >= -45 && movementAngle < 45:
		direction = 3
	case movementAngle >= 45 && movementAngle < 135:
		direction = 0
	case movementAngle >= 135 || movementAngle < -135:
		direction = 2
	case movementAngle >= -135 && movementAngle < -45:
		direction = 1
	}

	isFinished := sprite.frame == sprite.width

	if sprite.frame >= sprite.width {
		sprite.frame = 0
	}

	return isFinished, sprite.image.SubImage(image.Rect(x, direction*sprite.frameHeight, x+sprite.frameWidth, direction*sprite.frameHeight+sprite.frameHeight)).(*ebiten.Image)
}

type Bullet struct {
	x, y         float32
	angle        float32
	speed        float32
	creationTime time.Time // Add creationTime to track bullet age
	hp           int
}

type Enemy struct {
	x, y      float64
	sprites   *SpritePack
	hp        int
	attacking time.Time
	done      chan bool
}

var enemyTypes = []string{"Slime2", "Slime3"}

func NewEnemy(slimeName string, startX, startY float64) *Enemy {
	e := &Enemy{
		x:       startX,
		y:       startY,
		sprites: NewSpritePack(slimeName),
		hp:      100,
		done:    make(chan bool),
	}
	go e.RandomMovement()
	return e

}

func (e *Enemy) RandomMovement() {
	ticker := time.NewTicker(1000 * time.Millisecond) // Change direction every 500ms
	defer ticker.Stop()

	var dx, dy float64
	for {
		select {
		case <-e.done:
			return
		case <-ticker.C:
			// Randomly change direction
			angle := rand.Float64() * 2 * math.Pi
			dx = math.Cos(angle)
			dy = math.Sin(angle)
			if rand.Intn(100) > 50 {
				e.attacking = time.Now().Add(time.Second * 2)
			}
		default:
			// Stop moving if HP is less than 1
			if e.hp < 1 {
				return
			}

			// Move the enemy
			e.x += dx * 1 // Adjust speed as needed
			e.y += dy * 1 // Adjust speed as needed

			// Ensure the enemy stays within bounds
			if e.x < 0 {
				e.x = 0
				dx = -dx
			} else if e.x > 1920 {
				e.x = 1920
				dx = -dx
			}

			if e.y < 0 {
				e.y = 0
				dy = -dy
			} else if e.y > 1080 {
				e.y = 1080
				dy = -dy
			}

			time.Sleep(16 * time.Millisecond) // Roughly 60 updates per second
		}
	}
}

type Game struct {
	menuOptions   []string
	selected      int
	inMenu        bool
	keys          *Keys
	exit          bool
	player        *Player
	debugLogs     []string
	lastLogTime   time.Time
	settings      bool
	bullets       []*Bullet
	frameCount    int // Add a frame counter
	gamepads      []ebiten.GamepadID
	enemies       []*Enemy // Add a slice to hold enemies
	enemiesKilled int      // Add a field to track the number of enemies killed
	gameOver      bool     // Add a field to track if the game is over
}

type KeyEvent struct {
	Key     ebiten.Key
	Pressed time.Duration
}

type Keys struct {
	pressedKeys map[ebiten.Key]time.Time
	eventChan   chan KeyEvent
}

func NewKeys() *Keys {
	return &Keys{
		pressedKeys: make(map[ebiten.Key]time.Time),
		eventChan:   make(chan KeyEvent, 1000),
	}
}

func (k *Keys) Update() {
	now := time.Now()
	for key := range k.pressedKeys {
		if !ebiten.IsKeyPressed(key) {
			delete(k.pressedKeys, key)
			continue
		}
		duration := now.Sub(k.pressedKeys[key])
		if duration > 90*time.Millisecond {
			k.eventChan <- KeyEvent{Key: key, Pressed: duration}
			delete(k.pressedKeys, key)
		}
	}

	// Iterate over all possible keys
	for key := ebiten.Key(0); key <= ebiten.KeyMax; key++ {
		if ebiten.IsKeyPressed(key) {
			if _, exists := k.pressedKeys[key]; !exists {
				k.pressedKeys[key] = now
			}
		}
	}
}

func (g *Game) handleKeys() {
	for event := range g.keys.eventChan {
		switch event.Key {
		case ebiten.KeyArrowDown:
			g.selected = (g.selected + 1) % len(g.menuOptions)
		case ebiten.KeyArrowUp:
			g.selected = (g.selected - 1 + len(g.menuOptions)) % len(g.menuOptions)
		case ebiten.KeyEnter:
			switch g.menuOptions[g.selected] {
			case "Start Game":
				g.inMenu = false
				if g.gameOver {
					g.resetGame()
				}
			case "Exit":
				g.exit = true
			case "Settings":
				g.inMenu = false
				g.settings = true
			}

		case ebiten.KeyEscape:
			g.settings = false
			g.inMenu = true
		case ebiten.KeyR:
			if g.gameOver {
				g.resetGame() // Restart the game if it's over
			}
		}
	}
}

func (g *Game) handleGamepadInput() bool {
	// Check for connected gamepads
	g.gamepads = inpututil.AppendJustConnectedGamepadIDs(g.gamepads)

	for i, id := range g.gamepads {
		if inpututil.IsGamepadJustDisconnected(id) {
			g.gamepads = slices.Delete(g.gamepads, i, i+1)
			continue
		}
		// Handle menu navigation
		if g.inMenu {
			if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton0) { // A button
				switch g.menuOptions[g.selected] {
				case "Start Game":
					g.inMenu = false
				case "Exit":
					g.exit = true
				case "Settings":
					g.inMenu = false
					g.settings = true
				}
			}
			if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton1) { // B button
				g.settings = false
				g.inMenu = true
			}
			if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton13) { // D-pad down
				g.selected = (g.selected + 1) % len(g.menuOptions)
			}
			if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton12) { // D-pad up
				g.selected = (g.selected - 1 + len(g.menuOptions)) % len(g.menuOptions)
			}
		} else {
			// Handle player movement
			dx := ebiten.GamepadAxisValue(id, 0) // Left stick horizontal
			dy := ebiten.GamepadAxisValue(id, 1) // Left stick vertical

			// Normalize the vector if both x and y are non-zero
			if dx != 0 && dy != 0 {
				length := float64(math.Sqrt(float64(dx*dx + dy*dy)))
				dx /= length
				dy /= length
			}

			// Update player position
			g.player.x += float64(dx * 2) // Convert dx to float64
			g.player.y += float64(dy * 2) // Convert dy to float64

			// Calculate movement angle
			movementAngle := float32(math.Atan2(float64(dy), float64(dx)) * (180 / math.Pi))
			g.player.movementAngle = movementAngle

			g.player.attacking = ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton2) // X button

			// Handle shooting action every 10th frame
			if g.frameCount%10 == 0 && ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton3) { // Y button
				bullet := &Bullet{
					x:            float32(g.player.x) + 32, // Convert x to float32
					y:            float32(g.player.y) + 32, // Convert y to float32
					angle:        g.player.movementAngle,
					speed:        5,
					creationTime: time.Now(), // Initialize creationTime
				}
				g.bullets = append(g.bullets, bullet)
			}
		}
	}
	return len(g.gamepads) > 0
}

func (g *Game) checkBulletCollisions() {
	var remainingBullets []*Bullet
	for _, bullet := range g.bullets {
		collided := false
		for _, enemy := range g.enemies {
			if enemy.hp < 1 {
				continue
			}
			// Define the enemy's bounding rectangle

			// EMEMY HIT BOX IS 32px *32px
			enemyRect := image.Rect(
				int(enemy.x)+16, int(enemy.y)+16,
				int(enemy.x)+enemy.sprites.idle.frameWidth-16,
				int(enemy.y)+enemy.sprites.idle.frameHeight-16,
			)

			// Check if the bullet is within the enemy's rectangle
			// The enemy is hit by bullets here wihch are a radius of 3px
			if enemyRect.Min.X <= int(bullet.x)+3 && int(bullet.x)-3 <= enemyRect.Max.X &&
				enemyRect.Min.Y <= int(bullet.y)+3 && int(bullet.y)-3 <= enemyRect.Max.Y {
				// Collision detected
				enemy.hp -= 20
				if enemy.hp <= 0 {
					g.enemiesKilled++ // Increment the enemies killed count
				}
				collided = true
				break
			}

		}
		if !collided {
			remainingBullets = append(remainingBullets, bullet)
		}
	}
	g.bullets = remainingBullets
}

func (g *Game) checkEnemyCollisions() {
	for _, enemy := range g.enemies {
		if enemy.hp <= 1 {
			continue
		}
		if enemy.attacking.After(time.Now()) {
			// Define the player's bounding rectangle
			playerRect := image.Rect(

				int(g.player.x)+16, int(g.player.y)+16,
				int(g.player.x)+g.player.spritePack.idle.frameWidth-16,
				int(g.player.y)+g.player.spritePack.idle.frameHeight-16,
			)

			// Define the enemy's bounding rectangle
			enemyRect := image.Rect(
				int(enemy.x)+16, int(enemy.y)+16,
				int(enemy.x)+enemy.sprites.idle.frameWidth-16,
				int(enemy.y)+enemy.sprites.idle.frameHeight-16,
			)

			// Check for collision
			if playerRect.Overlaps(enemyRect) {
				// Damage the player
				g.player.hp -= 10
			}
		}
	}
}

func (g *Game) Update() error {
	if g.exit {
		return fmt.Errorf("exit")
	}
	g.keys.Update()
	isConnected := g.handleGamepadInput()

	if g.inMenu {

	} else if !isConnected {
		if g.gameOver {
			return nil // Stop updating the game if it's over
		}
		g.player.running = ebiten.IsKeyPressed(ebiten.KeyShift)

		// Adjust the player's movement angle with A and D keys
		if ebiten.IsKeyPressed(ebiten.KeyA) {
			g.player.movementAngle -= 2 // Rotate left
		}
		if ebiten.IsKeyPressed(ebiten.KeyD) {
			g.player.movementAngle += 2 // Rotate right
		}

		// Calculate movement vector based on W and S keys
		var dx, dy float64
		rad := float64(g.player.movementAngle) * (math.Pi / 180)
		if ebiten.IsKeyPressed(ebiten.KeyW) {
			dx += math.Cos(rad) * 2
			dy += math.Sin(rad) * 2
		}
		if ebiten.IsKeyPressed(ebiten.KeyS) {
			dx -= math.Cos(rad) * 2
			dy -= math.Sin(rad) * 2
		}

		// Update player position
		g.player.x += dx
		g.player.y += dy
		if g.player.running {
			g.player.x += dx
			g.player.y += dy
		}

		// Handle swinging action
		g.player.attacking = ebiten.IsKeyPressed(ebiten.KeySpace)

		// Handle shooting action every 10th frame
		if g.frameCount%10 == 0 && g.player.attacking {
			bullet := &Bullet{
				x:            float32(g.player.x) + 32, // Convert x to float32
				y:            float32(g.player.y) + 32, // Convert y to float32
				angle:        g.player.movementAngle,
				speed:        10,
				creationTime: time.Now(), // Initialize creationTime
			}
			g.bullets = append(g.bullets, bullet)
		}
	}

	ebiten.SetFullscreen(true)
	if g.player.x <= 0 {
		g.player.x = 1920 // Wrap to the right edge
	} else if g.player.x >= 1920 {
		g.player.x = 0 // Wrap to the left edge
	}

	if g.player.y <= 0 {
		g.player.y = 1080 // Wrap to the bottom edge
	} else if g.player.y >= 1080 {
		g.player.y = 0 // Wrap to the top edge
	}

	// Update bullet positions and remove old bullets
	currentTime := time.Now()
	var activeBullets []*Bullet
	for _, bullet := range g.bullets {
		// Check if the bullet is older than 5 seconds
		if currentTime.Sub(bullet.creationTime) < 5*time.Second {
			rad := bullet.angle * (math.Pi / 180)
			bullet.x += bullet.speed * float32(math.Cos(float64(rad)))
			bullet.y += bullet.speed * float32(math.Sin(float64(rad)))
			activeBullets = append(activeBullets, bullet)
		}
	}
	g.bullets = activeBullets

	// Check for bullet collisions with enemies
	g.checkBulletCollisions()

	if g.frameCount%10 == 0 {
		// Check for enemy collisions with the player
		g.checkEnemyCollisions()
	}
	// Increment the frame counter
	g.frameCount++

	if g.player.hp <= 0 {
		g.gameOver = true // Set game over state when player dies
	}

	return nil
}

func (g *Game) drawMenu(screen *ebiten.Image) {
	// Set a default background color
	screen.Fill(color.RGBA{0, 0, 0, 255}) // Black background

	for i, option := range g.menuOptions {
		// Set the default selector color to orange
		selectorColor := color.RGBA{255, 165, 0, 255} // Orange as default
		if i == g.selected {
			// Change the color of the selector box based on the selected option
			switch g.menuOptions[g.selected] {
			case "Settings":
			case "Start Game":
				selectorColor = color.RGBA{0, 255, 0, 255} // Green for "Start Game"
			case "Exit":
				selectorColor = color.RGBA{255, 0, 0, 255} // Red for "Exit"
			}
			// Draw a colored rectangle behind the selected option
			vector.DrawFilledRect(screen, 15, float32(20+i*20), 150, 20, selectorColor, true)
		}
		ebitenutil.DebugPrintAt(screen, option, 20, 20+i*20)
	}
}

func drawArc(screen *ebiten.Image, x, y, radius, startAngle, endAngle float32, clr color.Color) {
	path := &vector.Path{}
	path.MoveTo(x, y)
	for angle := startAngle; angle <= endAngle; angle += 1 {
		rad := angle * (math.Pi / 180)
		path.LineTo(x+radius*float32(math.Cos(float64(rad))), y+radius*float32(math.Sin(float64(rad))))
	}
	path.Close()

	vertices, indices := path.AppendVerticesAndIndicesForFilling(nil, nil)

	// Create a new image to use as a source for the triangles
	src := ebiten.NewImage(1, 1)
	src.Fill(clr)

	op := &ebiten.DrawTrianglesOptions{}
	screen.DrawTriangles(vertices, indices, src, op)
}

func drawHPBar(screen *ebiten.Image, x, y float64, hp, maxHP int, hpColor color.Color) {
	barWidth := 50.0
	barHeight := 5.0
	hpRatio := float64(hp) / float64(maxHP)
	hpBarWidth := barWidth * hpRatio

	// Draw the background of the HP bar (gray)
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(barWidth), float32(barHeight), color.RGBA{128, 128, 128, 255}, true)

	// Draw the current HP (green)
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(hpBarWidth), float32(barHeight), hpColor, true)
}

func (g *Game) drawGameView(screen *ebiten.Image) {
	// Game drawing logic goes here
	ebitenutil.DebugPrint(screen, "Game is running...")

	if g.frameCount%10 == 0 {
		g.setDebugLogFirstSlot(fmt.Sprintf("loc: %f,%f", g.player.x, g.player.y))
	}

	// Draw the player sprite
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.player.x, g.player.y)

	// Draw the player's HP bar above the player
	drawHPBar(screen, g.player.x, g.player.y-10, g.player.hp, 100, color.RGBA{0, 255, 0, 255})

	// Draw an arrow pointing in the direction of the player's movement
	arrowLength := 20.0
	rad := float64(g.player.movementAngle) * (math.Pi / 180)
	arrowX := 32 + g.player.x + arrowLength*math.Cos(rad)
	arrowY := 32 + g.player.y + arrowLength*math.Sin(rad)
	drawArc(screen, float32(arrowX), float32(arrowY), 5, float32(rad-20), float32(rad+20), color.RGBA{255, 255, 0, 255}) // Yellow circle for arrow head

	sprite := g.player.spritePack.idle
	switch {
	case g.player.hp <= 0:
		sprite = g.player.spritePack.death
	case g.player.hp <= 50:
		sprite = g.player.spritePack.hurt
	case g.player.attacking:
		sprite = g.player.spritePack.attack
	case g.player.running:
		sprite = g.player.spritePack.run
	}

	isDone, currentPlayerSprite := sprite.GetCurrentSprite(g.frameCount, g.player.movementAngle)
	defer screen.DrawImage(currentPlayerSprite, op)
	if g.player.hp <= 0 && isDone {
		g.inMenu = true
		g.player.hp = 100
	}

	// Draw enemies
	newEnemies := []*Enemy{}
	for _, enemy := range g.enemies {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(enemy.x, enemy.y)

		// Draw the enemy's HP bar above the enemy
		drawHPBar(screen, enemy.x, enemy.y-10, enemy.hp, 100, color.RGBA{255, 0, 0, 255})

		// Use SubImage to get the desired part of the sprite
		var sprite *AnimatedSprite
		switch {
		case enemy.hp <= 0:
			sprite = enemy.sprites.death
		case enemy.hp <= 50:
			sprite = enemy.sprites.hurt
		case enemy.attacking.After(time.Now()):
			sprite = enemy.sprites.attack
		default:
			sprite = enemy.sprites.idle
		}

		//XXX Death animation is not working
		isDone, subImage := sprite.GetCurrentSprite(g.frameCount, 0)
		if !(enemy.hp <= 0 && isDone) {
			newEnemies = append(newEnemies, enemy)
		}

		// Draw the sub-image
		screen.DrawImage(subImage, op)
	}
	g.enemies = newEnemies

	// Draw bullets
	for _, bullet := range g.bullets {
		vector.DrawFilledCircle(screen, bullet.x, bullet.y, 3, color.RGBA{255, 255, 255, 255}, true) // White circle for bullets
	}
}

func (g *Game) setDebugLogFirstSlot(log string) {
	if len(g.debugLogs) == 0 {
		g.debugLogs = append(g.debugLogs, log)
	} else {
		g.debugLogs[0] = log
	}
}

func (g *Game) addDebugLog(log string) {
	if len(g.debugLogs) >= 10 {
		g.debugLogs = g.debugLogs[2:]
	}
	g.debugLogs = append(g.debugLogs, log)
}

func (g *Game) drawDebugLogs(screen *ebiten.Image) {
	// Set the position for the debug log box closer to the bottom right
	rec := screen.Bounds()
	x, y := rec.Max.X, rec.Max.Y
	width, height := x/3, y/3 // Adjusted size

	// Draw a semi-transparent background for the debug logs
	vector.DrawFilledRect(screen, float32(x)-float32(width), float32(y)-float32(height), float32(width), float32(height), color.RGBA{0, 0, 0, 128}, true)

	// Print each log line with smaller text
	for i, log := range g.debugLogs {
		ebitenutil.DebugPrintAt(screen, log, x+5, y+5+i*10) // Adjusted line spacing
	}
}

func (g *Game) drawSettings(screen *ebiten.Image) {
	// Example settings screen drawing logic
	screen.Fill(color.RGBA{50, 50, 50, 255}) // Dark gray background
	ebitenutil.DebugPrintAt(screen, "Settings", 20, 20)
	// Add more settings UI elements as needed
}

func (g *Game) drawGameOver(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, "Game Over", 300, 150)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Score: %d", g.enemiesKilled*100), 300, 200)
	ebitenutil.DebugPrintAt(screen, "Press R to restart", 300, 250)
}

func (g *Game) Draw(screen *ebiten.Image) {
	defer g.drawDebugLogs(screen)

	if g.inMenu {
		g.drawMenu(screen)
		return
	}

	if g.settings {
		g.drawSettings(screen)
		return
	}

	g.drawGameView(screen)

	if g.gameOver {
		g.drawGameOver(screen)
		return
	}

}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 1920, 1080
}

func (g *Game) resetGame() {
	g.player.hp = 100
	g.enemiesKilled = 0
	g.gameOver = false
	for _, enemy := range g.enemies {
		close(enemy.done)
	}
	g.enemies = []*Enemy{}

	for i := range 20 {
		// Generate random positions for the enemies
		x := rand.Intn(1920 / 2) // Assuming the screen width is 1920
		y := rand.Intn(1080 / 2) // Assuming the screen height is 1080
		g.enemies = append(g.enemies, NewEnemy(enemyTypes[(i%len(enemyTypes))], float64(x), float64(y)))
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	p, err := NewPlayer("circle", NewSpritePack("Slime1"))
	if err != nil {
		log.Fatal(err) // Log and exit if there's an error
	}
	p.hp = 100

	game := &Game{
		menuOptions: []string{"Start Game", "Settings", "Fun Stuff", "Exit"},
		selected:    0,
		inMenu:      false,
		keys:        NewKeys(),
		player:      p,
		debugLogs:   []string{},
		lastLogTime: time.Now(), // Initialize the last log time
	}

	// Example of adding an enemy

	for i := range 20 {
		// Generate random positions for the enemies
		x := rand.Intn(1920 / 2) // Assuming the screen width is 1920
		y := rand.Intn(1080 / 2) // Assuming the screen height is 1080
		game.enemies = append(game.enemies, NewEnemy(enemyTypes[(i%len(enemyTypes))], float64(x), float64(y)))
	}

	go game.handleKeys()
	// Example of adding a debug log
	game.addDebugLog("Game started")

	ebiten.SetWindowSize(1920, 1080) //1080p

	ebiten.SetWindowTitle("Basic Game Menu")
	ebiten.SetWindowResizable(true)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
