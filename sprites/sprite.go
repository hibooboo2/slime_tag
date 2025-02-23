package sprites

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hibooboo2/slime_tag/assets"
	"github.com/hibooboo2/slime_tag/ecs"
)

type AnimatedSprite struct {
	name               string
	image              *ebiten.Image
	width              int
	frame              int
	lastGameFrameCount int
	frameWidth         int
	frameHeight        int
}

type animatedSpriteID struct {
	Name  string
	Frame int
	image.Rectangle
}

var animatedSpritesCache = map[animatedSpriteID]*ebiten.Image{}

var spriteImageCache = map[string]*ebiten.Image{}

func NewSprite(fileName string, name string, width int) *AnimatedSprite {
	spriteImage, ok := spriteImageCache[fileName]
	if !ok {
		var err error
		spriteImage, _, err = ebitenutil.NewImageFromFileSystem(assets.Resources, fileName) // Load the sprite image
		if err != nil {
			panic(err)
		}
		spriteImageCache[fileName] = spriteImage
	}

	return &AnimatedSprite{
		name:        name,
		image:       spriteImage,
		width:       width,
		frameWidth:  64,
		frameHeight: 64,
	}
}

type SpritePack struct {
	SpriteWidth int
	Attack      *AnimatedSprite
	Death       *AnimatedSprite
	Hurt        *AnimatedSprite
	Idle        *AnimatedSprite
	Run         *AnimatedSprite
	Walk        *AnimatedSprite
}

func NewSlimeSpritePack(slimeName string) *SpritePack {
	sp := &SpritePack{
		SpriteWidth: 64,
	}

	sp.Attack = NewSprite(fmt.Sprintf("resources/slimes/PNG/%[1]s/Attack/%[1]s_Attack_full.png", slimeName), slimeName+"Attack", 10)
	sp.Death = NewSprite(fmt.Sprintf("resources/slimes/PNG/%[1]s/Death/%[1]s_Death_full.png", slimeName), slimeName+"Death", 10)
	sp.Hurt = NewSprite(fmt.Sprintf("resources/slimes/PNG/%[1]s/Hurt/%[1]s_Hurt_full.png", slimeName), slimeName+"Hurt", 5)
	sp.Idle = NewSprite(fmt.Sprintf("resources/slimes/PNG/%[1]s/Idle/%[1]s_Idle_full.png", slimeName), slimeName+"Idle", 6)
	sp.Run = NewSprite(fmt.Sprintf("resources/slimes/PNG/%[1]s/Run/%[1]s_Run_full.png", slimeName), slimeName+"Run", 8)
	sp.Walk = NewSprite(fmt.Sprintf("resources/slimes/PNG/%[1]s/Walk/%[1]s_Walk_full.png", slimeName), slimeName+"Walk", 8)

	return sp
}

func (sprite *AnimatedSprite) GetCurrentSprite(frameCount int, movementAngle float64) (bool, *ebiten.Image) {
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

	spriteRect := image.Rect(x, direction*sprite.frameHeight, x+sprite.frameWidth, direction*sprite.frameHeight+sprite.frameHeight)

	img, ok := animatedSpritesCache[animatedSpriteID{Name: sprite.name, Frame: sprite.frame, Rectangle: spriteRect}]
	if !ok {
		img = sprite.image.SubImage(spriteRect).(*ebiten.Image)
		animatedSpritesCache[animatedSpriteID{Name: sprite.name, Frame: sprite.frame, Rectangle: spriteRect}] = img
	}

	return isFinished, img
}

type Sprite struct {
	LocX, LocY       int
	SpriteX, SpriteY int
	Size             int
	Img              *ebiten.Image
}

var spritesCache = map[image.Rectangle]*ebiten.Image{}

func (hs *Sprite) Draw(screen *ebiten.Image, game ecs.Game) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(hs.LocX), float64(hs.LocY))
	screen.DrawImage(hs.Img, op)
}

func (hs *Sprite) Remove() bool {
	return false
}

func NewSpriteFromSpriteSheet(sprites *ebiten.Image, x, y, size, spriteX, spriteY int) *Sprite {
	hs := &Sprite{LocX: x, LocY: y, SpriteX: spriteX, SpriteY: spriteY, Size: size}
	rect := image.Rect(hs.SpriteX*hs.Size, hs.SpriteY*hs.Size, (hs.SpriteX*hs.Size)+hs.Size, (hs.SpriteY*hs.Size)+hs.Size)
	img, ok := spritesCache[rect]
	if !ok {
		img = sprites.SubImage(rect).(*ebiten.Image)
		spritesCache[rect] = img
	}
	hs.Img = img

	return hs
}
