package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/hibooboo2/slime_tag/assets"
	"github.com/hibooboo2/slime_tag/ecs"
	"github.com/hibooboo2/slime_tag/sound"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

var (
	screenWidth     = 0
	screenHeight    = 0
	backgroundImage *ebiten.Image
	statusBoxBg     *ebiten.Image
)

var gameFont font.Face

var baseResolution = 1920 // Base resolution for scaling

func init() {
	// Create a black background image
	backgroundImage = ebiten.NewImage(1, 1)
	backgroundImage.Fill(color.Black)

	tt, err := opentype.Parse(goregular.TTF)
	if err != nil {
		log.Fatal(err)
	}
	const dpi = 72
	gameFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    24,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Load and prepare the status box background
	caveImg, _, err := ebitenutil.NewImageFromFileSystem(assets.Resources, "resources/slimes/PNG/0.png")
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}

	// Create a new image with transparency for the status box
	statusBoxBg = ebiten.NewImage(caveImg.Bounds().Dx(), caveImg.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	op.ColorM.Scale(1, 1, 1, 0.5) // 50% transparency
	statusBoxBg.DrawImage(caveImg, op)

	// Use the same image for the background
	backgroundImage = caveImg // Use the same loaded image for the main background
}

type AnimatedSprite struct {
	image              *ebiten.Image
	width              int
	frame              int
	lastGameFrameCount int
	frameWidth         int
	frameHeight        int
}

type SpritePack struct {
	attack *AnimatedSprite
	death  *AnimatedSprite
	hurt   *AnimatedSprite
	idle   *AnimatedSprite
	run    *AnimatedSprite
	walk   *AnimatedSprite
}

type HPBar struct {
	lastDamageTime time.Time
	currentOpacity float64
	maxHP          int
	currentHP      int
	visible        bool
}

type Player struct {
	playerType          string
	x, y                float64
	attacking           bool
	running             bool
	movementAngle       float32
	spritePack          *SpritePack
	hpBar               *HPBar
	headstonesCollected int // Add this line to track collected headstones
	headstonePowerup    time.Time
}

var sprites = map[string]*ebiten.Image{}

func NewSprite(fileName string, width int) *AnimatedSprite {
	spriteImage, ok := sprites[fileName]
	if !ok {
		var err error
		spriteImage, _, err = ebitenutil.NewImageFromFileSystem(assets.Resources, fileName) // Load the sprite image
		if err != nil {
			panic(err)
		}
		sprites[fileName] = spriteImage
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

	sp.attack = NewSprite(fmt.Sprintf("resources/slimes/PNG/%[1]s/Attack/%[1]s_Attack_full.png", slimeName), 10)
	sp.death = NewSprite(fmt.Sprintf("resources/slimes/PNG/%[1]s/Death/%[1]s_Death_full.png", slimeName), 10)
	sp.hurt = NewSprite(fmt.Sprintf("resources/slimes/PNG/%[1]s/Hurt/%[1]s_Hurt_full.png", slimeName), 5)
	sp.idle = NewSprite(fmt.Sprintf("resources/slimes/PNG/%[1]s/Idle/%[1]s_Idle_full.png", slimeName), 6)
	sp.run = NewSprite(fmt.Sprintf("resources/slimes/PNG/%[1]s/Run/%[1]s_Run_full.png", slimeName), 8)
	sp.walk = NewSprite(fmt.Sprintf("resources/slimes/PNG/%[1]s/Walk/%[1]s_Walk_full.png", slimeName), 8)

	return sp
}

func NewPlayer(playerType string, spritePack *SpritePack) (*Player, error) {
	return &Player{
		playerType: playerType,
		x:          100, // Center of the screen horizontally
		y:          100, // Center of the screen vertically
		spritePack: spritePack,
		hpBar:      NewHPBar(100),
	}, nil
}

func (sprite *AnimatedSprite) GetCurrentSprite(frameCount int, movementAngle float32) (bool, *ebiten.Image) {
	fps := int(ebiten.ActualFPS())
	if fps == 0 {
		fps = 60
	}

	if fps > 7 && frameCount%(fps/7) == 0 {
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

	if sprite.lastGameFrameCount != frameCount-1 {
		sprite.frame = 0
	}

	if sprite.frame >= sprite.width {
		sprite.frame = 0
	}
	sprite.lastGameFrameCount = frameCount

	return isFinished, sprite.image.SubImage(image.Rect(x, direction*sprite.frameHeight, x+sprite.frameWidth, direction*sprite.frameHeight+sprite.frameHeight)).(*ebiten.Image)
}

type Intention int

const (
	Idle Intention = iota
	Attack
	Chase
)

type Enemy struct {
	x, y                float64
	sprites             *SpritePack
	hpBar               *HPBar
	lastIntentionChange time.Time
	intention           Intention
	speed               float64
	dx, dy              float64
	movementAngle       float32
	attacking           time.Time
	shouldRemove        bool
	soundPlayed         bool // New flag to track if the death sound has been played
}

var _ ecs.Overlapper = &Enemy{}
var _ ecs.Drawer = &Enemy{}

func (e *Enemy) getRect() image.Rectangle {
	hitBoxSize := 20
	return image.Rect(
		int(e.x)+e.sprites.idle.frameWidth/2-hitBoxSize/2, int(e.y)+e.sprites.idle.frameHeight/2-hitBoxSize/2,
		int(e.x)+e.sprites.idle.frameWidth/2+hitBoxSize/2, int(e.y)+e.sprites.idle.frameHeight/2+hitBoxSize/2,
	)
}

func (e *Enemy) Overlaps(r image.Rectangle, game ecs.Game) bool {
	g := game.(*Game)
	if !(g.frameCount%10 == 0) {
		return false
	}

	if e.intention == Attack {
		// Check for collision
		if r.Overlaps(e.getRect()) {
			// Damage the player
			g.player.hpBar.AddHP(-10)

			// Add floating text for damage
			g.AddFloatingText("-10 HP", g.player.x, g.player.y, color.RGBA{255, 0, 0, 255}) // Red text for damage
		}
	}

	return r.Overlaps(e.getRect())
}

func (e *Enemy) Draw(screen *ebiten.Image, game ecs.Game) {
	g := game.(*Game)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(e.x, e.y)

	// Draw the enemy's HP bar above the enemy
	drawHPBar(screen, e.x, e.y-10, e.hpBar, color.RGBA{255, 0, 0, 255})

	// Use SubImage to get the desired part of the sprite
	var sprite *AnimatedSprite
	switch {
	case e.hpBar.currentHP <= 0:
		sprite = e.sprites.death
		if !e.soundPlayed { // Check if the death sound has not been played
			if err := sound.Play("death"); err != nil {
				log.Println("Error playing sound:", err) // Log any errors
			}
			e.soundPlayed = true // Mark that the sound has been played
		}

	case e.hpBar.currentHP <= 50:
		sprite = e.sprites.hurt
	default:
		switch e.intention {
		case Idle:
			sprite = e.sprites.idle
		case Chase:
			sprite = e.sprites.run
		case Attack:
			sprite = e.sprites.attack
		}
	}

	isDone, subImage := sprite.GetCurrentSprite(g.frameCount, e.movementAngle)
	if e.hpBar.currentHP <= 0 && isDone {
		g.AddHeadStone(int(e.x), int(e.y))
		e.shouldRemove = true // Keep this line to mark for removal
	}

	screen.DrawImage(subImage, op)

	if g.debugMode {
		drawRect(e.getRect(), screen)
	}
}

func (e *Enemy) Remove() bool {
	return e.shouldRemove
}

var enemyTypes = []string{
	"Slime2",
	// "Slime3",
}

func NewEnemy(slimeName string, startX, startY float64) *Enemy {
	e := &Enemy{
		x:         startX,
		y:         startY,
		sprites:   NewSpritePack(slimeName),
		intention: Idle,
		hpBar:     NewHPBar(100),
	}
	return e
}

func (e *Enemy) RandomMovement(playerX, playerY float64) {
	if e.hpBar.currentHP < 1 {
		return
	}
	// Randomly change direction
	if time.Since(e.lastIntentionChange) > time.Duration(rand.Intn(1400)+600)*time.Millisecond {
		switch {
		case rand.Intn(100) > 40:
			e.intention = Attack
		case rand.Intn(100) > 15:
			// Change direction towards the player
			e.intention = Chase
			dx := playerX - e.x
			dy := playerY - e.y
			length := math.Sqrt(dx*dx + dy*dy)
			e.dx = (dx / length) * 2
			e.dy = (dy / length) * 2
			e.movementAngle = float32(math.Atan2(float64(dy), float64(dx)) * (180 / math.Pi))
		default:
			e.intention = Idle
			angle := rand.Float64() * 2 * math.Pi
			e.movementAngle = float32(angle)
			e.dx = math.Cos(angle)
			e.dy = math.Sin(angle)
		}
		e.lastIntentionChange = time.Now()
	}

	// Move the enemy
	e.x += e.dx * 1 // Adjust speed as needed
	e.y += e.dy * 1 // Adjust speed as needed

	// Ensure the enemy stays within bounds
	if e.x < 0 {
		e.x = 0
		e.dx = -e.dx
	} else if e.x > float64(screenWidth) {
		e.x = float64(screenWidth)
		e.dx = -e.dx
	}

	if e.y < 0 {
		e.y = 0
		e.dy = -e.dy
	} else if e.y > float64(screenHeight) {
		e.y = float64(screenHeight)
		e.dy = -e.dy
	}
}

type FloatingText struct {
	text      string
	x, y      float64
	color     color.Color
	startTime time.Time
	duration  time.Duration
	opacity   float64
}

func NewFloatingText(text string, x, y float64, color color.Color) *FloatingText {
	return &FloatingText{
		text:      text,
		x:         x,
		y:         y,
		color:     color,
		startTime: time.Now(),
		duration:  2 * time.Second,
		opacity:   1.0,
	}
}

func (ft *FloatingText) Update() {
	elapsed := time.Since(ft.startTime)
	if elapsed > ft.duration {
		ft.opacity = 0
		return
	}

	// Calculate the new opacity and position
	ft.opacity = 1 - float64(elapsed)/float64(ft.duration)
	ft.y += 0.5 // Move downward slowly
}

func (ft *FloatingText) Draw(screen *ebiten.Image) {
	if ft.opacity <= 0 {
		return
	}

	// Set the color with the current opacity
	r, g, b, _ := ft.color.RGBA()
	clr := color.RGBA{uint8(r), uint8(g), uint8(b), uint8(255 * ft.opacity)}

	// Draw the text
	text.Draw(screen, ft.text, gameFont, int(ft.x), int(ft.y), clr)
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
		switch {
		case g.settings:
			switch event.Key {
			case ebiten.KeyArrowDown:
				if g.showResolutions {
					g.resolutionIdx = (g.resolutionIdx + 1) % len(g.resolutionOptions)
				} else {
					g.settingsSelected = (g.settingsSelected + 1) % len(g.settingsOptions)
				}
			case ebiten.KeyArrowUp:
				if g.showResolutions {
					g.resolutionIdx = (g.resolutionIdx - 1 + len(g.resolutionOptions)) % len(g.resolutionOptions)
				} else {
					g.settingsSelected = (g.settingsSelected - 1 + len(g.settingsOptions)) % len(g.settingsOptions)
				}
			case ebiten.KeyEnter:
				if g.showResolutions {
					// Apply selected resolution
					g.settingsOptions[0] = "Resolution: " + g.resolutionOptions[g.resolutionIdx]
					g.showResolutions = false
					screenWidth, screenHeight = parseResolution(g.resolutionOptions[g.resolutionIdx])
					g.addDebugLog(fmt.Sprintf("Setting resolution to %d x %d", screenWidth, screenHeight))
					log.Println("resolution changed", screenWidth, screenHeight)
					ebiten.SetWindowSize(screenWidth, screenHeight)
					ebiten.SetFullscreen(true)
					homeDir, err := os.UserHomeDir()
					if err != nil {
						log.Fatalf("Error getting home directory: %v", err)
					}
					err = SaveSettings(path.Join(homeDir, ".slime_tag_settings.json"), g.debugMode)
					if err != nil {
						g.addDebugLog(fmt.Sprintf("Error saving settings: %v", err))
					}
				} else {
					switch g.settingsSelected {
					case 0: // Resolution
						g.showResolutions = true
					case 1: // Debug Mode
						g.debugMode = !g.debugMode
						if g.debugMode {
							g.settingsOptions[1] = "Toggle Debug Mode: On"
						} else {
							g.settingsOptions[1] = "Toggle Debug Mode: Off"
						}
						// Save settings when debug mode is toggled
						homeDir, err := os.UserHomeDir()
						if err == nil {
							SaveSettings(path.Join(homeDir, ".slime_tag_settings.json"), g.debugMode)
						}
					case 2: // Crosshair Selector
						// Implementation for Crosshair Selector
					case 3: // Back
						g.settings = false
						g.inMainMenu = true
					}
				}
			case ebiten.KeyEscape:
				if g.showResolutions {
					g.showResolutions = false
				} else {
					g.settings = false
					g.inMainMenu = true
				}
			}
		case g.inMainMenu:
			switch event.Key {
			case ebiten.KeyArrowDown:
				g.selected = (g.selected + 1) % len(g.menuOptions)
			case ebiten.KeyArrowUp:
				g.selected = (g.selected - 1 + len(g.menuOptions)) % len(g.menuOptions)
			case ebiten.KeyEnter:
				switch g.menuOptions[g.selected] {
				case "Start Game":
					g.inMainMenu = false
					g.paused = false
					if g.gameOver {
						g.resetGame()
					}
				case "Exit":
					g.exit = true
				case "Settings":
					g.inMainMenu = false
					g.settings = true
					g.paused = true
				}

			case ebiten.KeyEscape:
				g.settings = false
				g.inMainMenu = true
				g.paused = true
			case ebiten.KeyR:
				if g.gameOver {
					g.resetGame() // Restart the game if it's over
				}
			}
		default:
			switch event.Key {
			case ebiten.KeyEscape:
				g.settings = false
				g.inMainMenu = true
			case ebiten.KeyR:
				if g.gameOver {
					g.resetGame() // Restart the game if it's over
				}
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
		if g.inMainMenu {
			if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton0) { // A button
				switch g.menuOptions[g.selected] {
				case "Start Game":
					g.inMainMenu = false
				case "Exit":
					g.exit = true
				case "Settings":
					g.inMainMenu = false
					g.settings = true
				}
			}
			if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton1) { // B button
				g.settings = false
				g.inMainMenu = true
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
					radius:       4,
					creationTime: time.Now(), // Initialize creationTime
					hp:           20,
				}
				g.bullets.Add(bullet)
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

	if g.inMainMenu {
		g.paused = true
	} else if !isConnected {
		if g.gameOver || g.paused {
			return nil // Stop updating the game if it's over or paused is "true"
		}
		g.player.running = ebiten.IsKeyPressed(ebiten.KeyShift)

		// Calculate movement based on WASD keys
		dx, dy := 0.0, 0.0
		speed := 2.0
		if g.player.running {
			speed = 4.0
		}

		if ebiten.IsKeyPressed(ebiten.KeyW) {
			dy -= speed
		}
		if ebiten.IsKeyPressed(ebiten.KeyS) {
			dy += speed
		}
		if ebiten.IsKeyPressed(ebiten.KeyA) {
			dx -= speed
		}
		if ebiten.IsKeyPressed(ebiten.KeyD) {
			dx += speed
		}

		// Update player position
		g.player.x += dx
		g.player.y += dy

		// Calculate movement angle for sprite direction
		if dx != 0 || dy != 0 {
			g.player.movementAngle = float32(math.Atan2(dy, dx) * (180 / math.Pi))
		}

		// Handle shooting action with left mouse button
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && g.frameCount%10 == 0 {
			g.player.attacking = true

			// Create bullet aimed at mouse position
			bullet := &Bullet{
				x:            float32(g.player.x) + 32,
				y:            float32(g.player.y) + 32,
				angle:        calculateAngleToMouse(g.player.x, g.player.y),
				speed:        10,
				creationTime: time.Now(),
				radius:       4,
				hp:           20,
			}
			if time.Since(g.player.headstonePowerup) < 10*time.Second {
				bullet.hp = 50
				if err := sound.Play("powerattack"); err != nil {
					log.Println("Error playing sound:", err) // Log any errors
				}
			} else if err := sound.Play("attack"); err != nil {
				log.Println("Error playing sound:", err) // Log any errors
			}

			g.bullets.Add(bullet) // Add bullet to the bullets entity list
		} else {
			g.player.attacking = false
		}

	}
	if g.paused {
		return nil
	}
	if g.player.x <= 0 {
		g.player.x = float64(screenWidth) // Wrap to the right edge
	} else if g.player.x >= float64(screenWidth) {
		g.player.x = 0 // Wrap to the left edge
	}

	if g.player.y <= 0 {
		g.player.y = float64(screenHeight) // Wrap to the bottom edge
	} else if g.player.y >= float64(screenHeight) {
		g.player.y = 0 // Wrap to the top edge
	}

	for i := len(*g.bullets) - 1; i >= 0; i-- {
		bullet, ok := (*g.bullets)[i].(*Bullet)
		if !ok {
			continue
		}
		rad := bullet.angle * (math.Pi / 180)
		bullet.x += bullet.speed * float32(math.Cos(float64(rad))) // Update bullet position based on speed and angle
		bullet.y += bullet.speed * float32(math.Sin(float64(rad))) // Update bullet position based on speed and angle
	}

	g.player.hpBar.updateOpacity()
	for _, e := range *g.entities {
		enemy, ok := e.(*Enemy)
		if !ok {
			continue
		}
		enemy.RandomMovement(g.player.x, g.player.y)
		enemy.hpBar.updateOpacity()
	}

	if g.frameCount%3 == 0 {
		// Check for bullet collisions with enemies
		g.checkEnemyBulletCollisions()

		// Check for enemy collisions with the player
		g.checkPlayerCollisions()
	}

	// Check if all enemies are defeated
	totalEnemies := 0
	for _, enemy := range *g.entities {
		_, ok := enemy.(*Enemy)
		if ok {
			totalEnemies++
		}
	}
	if totalEnemies == 0 {
		// Spawn 20 new enemies
		for i := range 20 {
			x := rand.Intn(screenWidth / 2)
			y := rand.Intn(screenHeight / 2)
			g.entities.Add(NewEnemy(enemyTypes[(i%len(enemyTypes))], float64(x), float64(y)))
		}
	}
	// Increment the frame counter
	g.frameCount++

	if g.player.hpBar.currentHP <= 0 {
		g.gameOver = true // Set game over state when player dies
		g.addDebugLog("Game Over")
		g.AddFloatingText("Game Over", g.player.x, g.player.y, color.RGBA{255, 0, 0, 255})
	}

	// Initialize spawn interval if it's zero
	if g.spawnInterval == 0 {
		g.spawnInterval = 5 * time.Second
		g.lastEnemySpawn = time.Now()
	}

	// Check if it's time to spawn a new enemy
	if time.Since(g.lastEnemySpawn) >= g.spawnInterval {
		x := rand.Intn(screenWidth / 2)
		y := rand.Intn(screenHeight / 2)
		g.entities.Add(NewEnemy(enemyTypes[rand.Intn(len(enemyTypes))], float64(x), float64(y)))
		g.lastEnemySpawn = time.Now()
	}

	if time.Since(g.lastPowerUpSpawn) >= 10*time.Second {
		g.entities.Add(NewRandomPowerUp())
		g.lastPowerUpSpawn = time.Now()
	}

	// Update floating texts
	var activeTexts []*FloatingText
	for _, ft := range g.floatingTexts {
		ft.Update()
		if ft.opacity > 0 {
			activeTexts = append(activeTexts, ft)
		}
	}
	g.floatingTexts = activeTexts

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
	op.ColorM.Scale(1, 1, 1, 0.8)
	screen.DrawTriangles(vertices, indices, src, op)
}

func drawHPBar(screen *ebiten.Image, x, y float64, hpBar *HPBar, hpColor color.Color) {
	if !hpBar.visible {
		return
	}

	barWidth := 50.0
	barHeight := 5.0
	hpRatio := float64(hpBar.currentHP) / float64(hpBar.maxHP)
	hpBarWidth := barWidth * hpRatio

	// Draw the background of the HP bar (gray)
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(barWidth), float32(barHeight), color.RGBA{128, 128, 128, 255}, true)

	// Draw the current HP
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(hpBarWidth), float32(barHeight), hpColor, true)
}

func (g *Game) drawGameView(screen *ebiten.Image) {
	if g.frameCount%10 == 0 {
		g.setDebugLogFirstSlot(fmt.Sprintf("loc: %f,%f", g.player.x, g.player.y))
	}

	// Draw the player sprite
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.player.x, g.player.y)

	// Draw the player's HP bar above the player
	drawHPBar(screen, g.player.x, g.player.y-10, g.player.hpBar, color.RGBA{0, 255, 0, 255})

	// Draw an arrow pointing in the direction of the player's movement
	arrowLength := 20.0
	rad := float64(g.player.movementAngle) * (math.Pi / 180)
	arrowX := 32 + g.player.x + arrowLength*math.Cos(rad)
	arrowY := 32 + g.player.y + arrowLength*math.Sin(rad)
	drawArc(screen, float32(arrowX), float32(arrowY), 5, float32(rad-20), float32(rad+20), color.RGBA{255, 255, 0, 255}) // Yellow circle for arrow head

	sprite := g.player.spritePack.idle
	switch {
	case g.player.hpBar.currentHP <= 0:
		sprite = g.player.spritePack.death
	case g.player.hpBar.currentHP <= 50:
		sprite = g.player.spritePack.hurt
	case g.player.attacking:
		sprite = g.player.spritePack.attack
	case g.player.running:
		sprite = g.player.spritePack.run
	}

	playerRect := image.Rect(
		int(g.player.x)+g.player.spritePack.idle.frameWidth/2-10, int(g.player.y)+g.player.spritePack.idle.frameHeight/2-10,
		int(g.player.x)+g.player.spritePack.idle.frameWidth/2+10, int(g.player.y)+g.player.spritePack.idle.frameHeight/2+10,
	)

	if g.debugMode {
		drawRect(playerRect, screen)
	}

	isDone, currentPlayerSprite := sprite.GetCurrentSprite(g.frameCount, g.player.movementAngle)
	defer screen.DrawImage(currentPlayerSprite, op)
	if g.player.hpBar.currentHP <= 0 && isDone {
		g.inMainMenu = true
		g.player.hpBar.currentHP = 100
	}

	// Draw floating texts
	for _, ft := range g.floatingTexts {
		ft.Draw(screen)
	}

	g.entities.Draw(screen, g)
	g.bullets.Draw(screen, g)
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
		g.debugLogs = g.debugLogs[1:]
	}
	g.debugLogs = append(g.debugLogs, log)
}

func (g *Game) AddHeadStone(x, y int) {
	g.entities.Add(&HeadStone{Sprite: Sprite{locX: x, locY: y, spriteX: 0 + rand.Intn(5), spriteY: 1, size: 32}, deathTime: time.Now()})
}

func (g *Game) drawDebugLogs(screen *ebiten.Image) {
	// Set the position for the debug log box closer to the bottom right
	rec := screen.Bounds()
	x, y := rec.Max.X, rec.Max.Y
	width, height := x/3, y/3 // Adjusted size

	// Draw a semi-transparent background for the debug logs
	vector.DrawFilledRect(screen, float32(x)-float32(width), float32(y)-float32(height), float32(width), float32(height), color.RGBA{100, 0, 223, 128}, true)

	// Print each log line with smaller text
	for i, log := range g.debugLogs {
		ebitenutil.DebugPrintAt(screen, log, x+5, y+5+i*10) // Adjusted line spacing
	}
}

func (g *Game) drawSettings(screen *ebiten.Image) {
	screen.Fill(color.RGBA{50, 50, 50, 255}) // Dark gray background

	// Draw main settings menu
	for i, option := range g.settingsOptions {
		if i == g.settingsSelected {
			// Highlight selected option
			vector.DrawFilledRect(screen, 15, float32(20+i*30), 300, 25, color.RGBA{255, 165, 0, 255}, true)
		}
		ebitenutil.DebugPrintAt(screen, option, 20, 20+i*30)
	}

	// Draw resolution popup if active
	if g.showResolutions {
		// Draw popup background
		popupX := 320 // Position to the right of settings menu
		vector.DrawFilledRect(screen, float32(popupX), 20, 200, float32(len(g.resolutionOptions)*30+10), color.RGBA{70, 70, 70, 255}, true)

		// Draw resolution options
		for i, res := range g.resolutionOptions {
			if i == g.resolutionIdx {
				vector.DrawFilledRect(screen, float32(popupX), float32(20+i*30), 190, 25, color.RGBA{255, 165, 0, 255}, true)
			}
			ebitenutil.DebugPrintAt(screen, res, popupX+5, 20+i*30)
		}
	}
}

func (g *Game) drawGameOver(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, "Game Over", 300, 150)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Score: %d", (g.enemiesKilled*100)+(g.player.headstonesCollected*500)), 300, 200)
	ebitenutil.DebugPrintAt(screen, "Press R to restart", 300, 250)
}

type StatusBox struct {
	width, height float32
	x, y          float32
	enemies       int
	killed        int
	headstones    int
	scale         float32
}

func (box *StatusBox) Draw(screen *ebiten.Image, g *Game) {
	// Calculate scale based on both dimensions
	baseWidth := float32(1920)
	baseHeight := float32(1080)
	scaleX := float32(screenWidth) / baseWidth
	scaleY := float32(screenHeight) / baseHeight
	scale := float32(math.Min(float64(scaleX), float64(scaleY)))

	// Box should take up consistent percentage of screen
	box.width = float32(screenWidth) * 0.2
	box.height = float32(screenHeight) * 0.15
	margin := float32(20) * scale
	box.x = float32(screenWidth) - box.width - margin
	box.y = margin

	// Draw the background image scaled to the status box size
	op := &ebiten.DrawImageOptions{}
	op.ColorM.Scale(1, 1, 1, 0.5) // Set transparency to 50%

	// Scale the image to fit the status box dimensions
	op.GeoM.Scale(float64(box.width)/float64(statusBoxBg.Bounds().Dx()), float64(box.height)/float64(statusBoxBg.Bounds().Dy()))
	op.GeoM.Translate(float64(box.x), float64(box.y))

	// Draw the background image
	screen.DrawImage(backgroundImage, op) // Ensure backgroundImage is the loaded cave background

	// Scale text relative to box height to prevent overlap
	textScale := (box.height / 162.0) * 0.6 // Scale text to 60% of the box height ratio

	type statusRow struct {
		label   string
		value   string
		yOffset float32
	}
	texts := []statusRow{
		{"Enemies:", fmt.Sprintf("%d", box.enemies), box.height * 0.25},
		{"Kills:", fmt.Sprintf("%d", box.killed), box.height * 0.45},
		{"Score:", fmt.Sprintf("%d", (box.killed*100)+(500*box.headstones)), box.height * 0.65},
		{"FPS:", fmt.Sprintf("%.0f", ebiten.ActualFPS()), box.height * 0.85},
		{"Headstones:", fmt.Sprintf("%d", g.player.headstonesCollected), box.height * 1.05},
	}
	if time.Since(g.player.headstonePowerup) < 10*time.Second {
		texts = append(texts, statusRow{"Powerup Time Left:", fmt.Sprintf("%s", ((10 * time.Second) - time.Since(g.player.headstonePowerup)).Truncate(time.Second)), box.height * 1.25})
	}

	for _, txt := range texts {
		padding := box.width * 0.1
		labelBounds := text.BoundString(gameFont, txt.label)
		valueBounds := text.BoundString(gameFont, txt.value)
		labelWidth := float32(labelBounds.Dx()) * textScale
		valueWidth := float32(valueBounds.Dx()) * textScale

		// Adjust padding for left and right
		leftPadding := box.width * 0.05
		rightPadding := box.width * 0.2

		// Additional scaling for score value to prevent overflow
		localScale := textScale
		if txt.label == "Score:" && valueWidth+labelWidth+leftPadding+rightPadding > box.width {
			localScale = (box.width - labelWidth - rightPadding) / float32(valueBounds.Dx())
			valueWidth = float32(valueBounds.Dx()) * localScale
		}

		text.Draw(screen, txt.label, gameFont,
			int(box.x+padding),
			int(box.y+txt.yOffset),
			color.RGBA{255, 165, 0, 255})

		text.Draw(screen, txt.value, gameFont,
			int(box.x+box.width-padding-valueWidth),
			int(box.y+txt.yOffset),
			color.RGBA{255, 165, 0, 255})
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Get the current screen size
	screenWidth, screenHeight := screen.Size()

	// Draw the game background if needed
	// screen.DrawImage(backgroundImage, nil) // Uncomment if you have a separate game background

	totalEnemiesAlive := 0
	for _, e := range *g.entities {
		_, ok := e.(*Enemy)
		if !ok {
			continue
		}
		totalEnemiesAlive++
	}

	// Create a status box object
	statusBox := &StatusBox{
		width:      float32(screenWidth) * 0.2,
		height:     float32(screenHeight) * 0.15,
		x:          float32(screenWidth) - (float32(screenWidth)*0.2 - 20),
		y:          20,
		enemies:    totalEnemiesAlive,
		killed:     g.enemiesKilled,
		headstones: g.player.headstonesCollected,
		scale:      g.scaleFactor,
	}

	// Draw the status box background
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(statusBox.width)/float64(statusBoxBg.Bounds().Dx()), float64(statusBox.height)/float64(statusBoxBg.Bounds().Dy()))
	op.GeoM.Translate(float64(statusBox.x), float64(statusBox.y))
	screen.DrawImage(statusBoxBg, op) // Draw the status box background

	// Draw other elements in the status box as needed
	// ...

	if g.debugMode {
		defer g.drawDebugLogs(screen)
	}

	if g.inMainMenu {
		g.drawMenu(screen)
		return
	}

	if g.settings {
		g.drawSettings(screen)
		return
	}

	g.drawGameView(screen)

	statusBox.Draw(screen, g)

	if g.gameOver {
		g.drawGameOver(screen)
		return
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) resetGame() {
	g.player.hpBar.currentHP = 100
	g.player.headstonesCollected = 0
	g.enemiesKilled = 0
	g.gameOver = false
	g.entities = &ecs.Entities{}
	g.paused = false // Set paused to false when the game resets

	for i := range 20 {
		// Generate random positions for the enemies
		x := rand.Intn(screenWidth / 2)  // Assuming the screen width is 1920
		y := rand.Intn(screenHeight / 2) // Assuming the screen height is 1080
		g.entities.Add(NewEnemy(enemyTypes[(i%len(enemyTypes))], float64(x), float64(y)))
	}
	g.spawnInterval = 30 * time.Second
	g.lastEnemySpawn = time.Now()
}

func calculateAngleToMouse(playerX, playerY float64) float32 {
	mouseX, mouseY := ebiten.CursorPosition()
	dx := float64(mouseX) - (playerX + 32) // +32 to aim from center of player
	dy := float64(mouseY) - (playerY + 32)
	return float32(math.Atan2(dy, dx) * (180 / math.Pi))
}

// GetMaxScreenSize returns the maximum screen size for the primary monitor.
func GetMaxScreenSize() (int, int) {
	monitor := ebiten.Monitor()
	if monitor == nil {
		log.Fatal("No primary monitor found")
	}

	width, height := monitor.Size()
	return width, height
}

func main() {
	// Get the maximum screen size for the primary monitor
	// screenWidth, screenHeight = GetMaxScreenSize()
	sprites, _, err := ebitenutil.NewImageFromFileSystem(assets.Resources, "resources/slimes/ProjectUtumno_full.png") // Load the sprite image
	if err != nil {
		panic(err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting home directory: %v", err)
	}

	rand.Seed(time.Now().UnixNano())

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	p, err := NewPlayer("circle", NewSpritePack("Slime1"))
	if err != nil {
		log.Fatal(err) // Log and exit if there's an error
	}
	p.hpBar.currentHP = 100

	// Load settings before creating game instance
	settings, err := LoadSettings(path.Join(homeDir, ".slime_tag_settings.json"))
	if err != nil {
		screenWidth, screenHeight = GetMaxScreenSize()
		SaveSettings(path.Join(homeDir, ".slime_tag_settings.json"), false)
	} else {
		resolution := strings.Split(settings.Resolution, "x")
		screenWidth, _ = strconv.Atoi(resolution[0])
		screenHeight, _ = strconv.Atoi(resolution[1])
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Basic Game Menu")
	ebiten.SetFullscreen(true)

	log.Println("screenWidth", screenWidth, "screenHeight", screenHeight)

	// Create game instance after loading settings
	game := &Game{
		menuOptions: []string{"Start Game", "Settings", "Fun Stuff", "Exit"},
		settingsOptions: []string{
			"Resolution: 1920x1080",
			"Toggle Debug Mode: Off",
			"Crosshair Selector",
			"Back",
		},
		entities:         &ecs.Entities{},
		bullets:          &ecs.Entities{},
		selected:         0,
		settingsSelected: 0,
		inMainMenu:       true,
		keys:             NewKeys(),
		player:           p,
		debugLogs:        []string{},
		lastLogTime:      time.Now(),
		scaleFactor:      1.0,
		debugMode:        settings.DebugMode,
		sprites:          sprites,
		paused:           true,
	}

	if game.debugMode {
		game.settingsOptions[1] = "Toggle Debug Mode: On"
	}

	game.resolutionOptions = GetTopResolutions(screenWidth, screenHeight)

	// Example of adding an enemy

	for i := range 20 {
		// Generate random positions for the enemies
		x := rand.Intn(screenWidth / 2)  // Assuming the screen width is 1920
		y := rand.Intn(screenHeight / 2) // Assuming the screen height is 1080
		game.entities.Add(NewEnemy(enemyTypes[(i%len(enemyTypes))], float64(x), float64(y)))
	}

	go game.handleKeys()
	// Example of adding a debug log
	game.addDebugLog("Game started")

	for !(game.exit && game.inMainMenu) {
		if err := ebiten.RunGame(game); err != nil {
			log.Fatalf("Error running game: %v", err)
		}

	}
}

func (g *Game) AddFloatingText(text string, x, y float64, color color.Color) {
	g.floatingTexts = append(g.floatingTexts, NewFloatingText(text, x, y, color))
}

type Settings struct {
	Resolution string `json:"resolution"`
	Fullscreen bool   `json:"fullscreen"`
	DebugMode  bool   `json:"debugMode"`
	// Add other settings as needed
}

func SaveSettings(filePath string, debugMode bool) error {
	settings := Settings{
		Resolution: fmt.Sprintf("%dx%d", screenWidth, screenHeight),
		Fullscreen: true, // Assuming fullscreen is enabled
		DebugMode:  debugMode,
		// Add other settings as needed
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func LoadSettings(filePath string) (Settings, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Settings{}, err

	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, err
	}

	// Update game settings
	resolution := strings.Split(settings.Resolution, "x")
	if len(resolution) == 2 {
		width, _ := strconv.Atoi(resolution[0])
		height, _ := strconv.Atoi(resolution[1])
		screenWidth, screenHeight = width, height
		ebiten.SetWindowSize(screenWidth, screenHeight)
		ebiten.SetFullscreen(settings.Fullscreen)
	}

	return settings, nil
}

func loadImage(path string) (*ebiten.Image, error) {
	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func drawRect(r image.Rectangle, screen *ebiten.Image) {
	// Draw the player's bounding box
	vector.StrokeRect(screen,
		float32(r.Min.X), float32(r.Min.Y),
		float32(r.Dx()), float32(r.Dy()),
		1, color.RGBA{125, 21, 92, 255}, false)
}
