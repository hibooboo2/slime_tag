package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
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

type Player struct {
	playerType    string
	x, y          float64
	swinging      bool
	swingAngle    float32
	movementAngle float32
	sprite        *AnimatedSprite
}

func NewPlayer(playerType string, startX, startY float64, fileName string, spriteWidth int) (*Player, error) {
	spriteImage, _, err := ebitenutil.NewImageFromFileSystem(slimes, fileName) // Load the sprite image
	if err != nil {
		return nil, err // Return error if the image cannot be loaded
	}
	animatedSprite := AnimatedSprite{
		image:       spriteImage,
		width:       spriteWidth,
		frameWidth:  64,
		frameHeight: 64,
	}
	return &Player{
		playerType: playerType,
		x:          startX,
		y:          startY,
		sprite:     &animatedSprite,
	}, nil
}

func (sprite *AnimatedSprite) GetCurrentSprite(frameCount int, movementAngle float32) *ebiten.Image {
	x := 0

	if frameCount%5 == 0 {
		sprite.frame++
	}

	if frameCount > 0 {
		x = (sprite.frame % sprite.width) * sprite.frameWidth
	}

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

	return sprite.image.SubImage(image.Rect(x, direction*sprite.frameHeight, x+sprite.frameWidth, direction*sprite.frameHeight+sprite.frameHeight)).(*ebiten.Image)
}

type Bullet struct {
	x, y         float32
	angle        float32
	speed        float32
	creationTime time.Time // Add creationTime to track bullet age
}

type Enemy struct {
	x, y   float64
	sprite *ebiten.Image
}

func NewEnemy(fileName string, startX, startY float64) (*Enemy, error) {
	sprite, _, err := ebitenutil.NewImageFromFile(fileName) // Load the sprite image
	if err != nil {
		return nil, err // Return error if the image cannot be loaded
	}
	return &Enemy{
		x:      startX,
		y:      startY,
		sprite: sprite,
	}, nil
}

type Game struct {
	menuOptions []string
	selected    int
	inMenu      bool
	keys        *Keys
	exit        bool
	player      *Player
	debugLogs   []string
	lastLogTime time.Time
	settings    bool
	bullets     []*Bullet
	frameCount  int // Add a frame counter
	gamepads    []ebiten.GamepadID
	enemies     []*Enemy // Add a slice to hold enemies
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
			case "Exit":
				g.exit = true
			case "Settings":
				g.inMenu = false
				g.settings = true
			}
		case ebiten.KeyEscape:
			g.settings = false
			g.inMenu = true
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

			// Handle swinging action
			if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton2) && !g.player.swinging { // X button
				g.player.swinging = true
				g.player.swingAngle = 0
			}

			// Handle shooting action every 10th frame
			if g.frameCount%10 == 0 && ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton3) { // Y button
				bullet := &Bullet{
					x:            float32(g.player.x), // Convert x to float32
					y:            float32(g.player.y), // Convert y to float32
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

func (g *Game) Update() error {
	if g.exit {
		return fmt.Errorf("exit")
	}
	g.keys.Update()
	isConnected := g.handleGamepadInput()

	if g.inMenu {

	} else if !isConnected {
		// Calculate movement vector
		var dx, dy float32
		if ebiten.IsKeyPressed(ebiten.KeyW) {
			dy -= 2
		}
		if ebiten.IsKeyPressed(ebiten.KeyS) {
			dy += 2
		}
		if ebiten.IsKeyPressed(ebiten.KeyA) {
			dx -= 2
		}
		if ebiten.IsKeyPressed(ebiten.KeyD) {
			dx += 2
		}

		// Normalize the vector if both x and y are non-zero
		if dx != 0 && dy != 0 {
			length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
			dx /= length
			dy /= length
		}

		// Update player position
		g.player.x += float64(dx) // Convert dx to float64
		g.player.y += float64(dy) // Convert dy to float64

		// Calculate movement angle
		movementAngle := float32(math.Atan2(float64(dy), float64(dx)) * (180 / math.Pi))

		// Handle swinging action
		if ebiten.IsKeyPressed(ebiten.KeySpace) && !g.player.swinging {
			g.player.swinging = true
			g.player.swingAngle = 0
		}

		// Store the movement angle in the player struct
		g.player.movementAngle = movementAngle

		// Handle shooting action every 10th frame
		if g.frameCount%10 == 0 && ebiten.IsKeyPressed(ebiten.KeyE) {
			bullet := &Bullet{
				x:            float32(g.player.x), // Convert x to float32
				y:            float32(g.player.y), // Convert y to float32
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

	if g.player.swinging {
		g.player.swingAngle += 20 // Increment the swing angle three times faster
		if g.player.swingAngle >= 180 {
			g.player.swinging = false // End the swing after 180 degrees
		}
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

	// Increment the frame counter
	g.frameCount++

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

func (g *Game) drawGameView(screen *ebiten.Image) {
	// Game drawing logic goes here
	ebitenutil.DebugPrint(screen, "Game is running...")

	if g.frameCount%10 == 0 {
		g.setDebugLogFirstSlot(fmt.Sprintf("loc: %f,%f", g.player.x, g.player.y))
	}

	// Draw the player sprite
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.player.x, g.player.y)
	defer screen.DrawImage(g.player.sprite.GetCurrentSprite(g.frameCount, g.player.movementAngle), op)

	// Draw enemies
	for _, enemy := range g.enemies {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(enemy.x, enemy.y)

		// Define the rectangle area you want to draw from the sprite
		subImageRect := image.Rect(0, 0, 50, 50) // Example: top-left 50x50 area

		// Use SubImage to get the desired part of the sprite
		subImage := enemy.sprite.SubImage(subImageRect).(*ebiten.Image)

		// Draw the sub-image
		screen.DrawImage(subImage, op)
	}

	// Draw the swinging arc
	if g.player.swinging {
		startAngle := g.player.movementAngle - 90
		endAngle := startAngle + g.player.swingAngle
		drawArc(screen, float32(g.player.x), float32(g.player.y), 15, startAngle, endAngle, color.RGBA{255, 255, 0, 255}) // Convert x and y to float32
	}

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
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth / 2, outsideHeight / 2
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	p, err := NewPlayer("circle", 0, 0, "slimes/PNG/Slime1/Idle/Slime1_Idle_full.png", 6)
	if err != nil {
		log.Fatal(err) // Log and exit if there's an error
	}
	game := &Game{
		menuOptions: []string{"Start Game", "Settings", "Exit"},
		selected:    0,
		inMenu:      false,
		keys:        NewKeys(),
		player:      p,
		debugLogs:   []string{},
		lastLogTime: time.Now(), // Initialize the last log time
	}

	// Example of adding an enemy
	enemy, err := NewEnemy("slimes/PNG/Slime2/Idle/Slime2_Idle_full.png", 300, 200)
	if err != nil {
		log.Fatal(err)
	}
	game.enemies = append(game.enemies, enemy)

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
