package main

import (
	"fmt"
	"image"
	"log"
	"os"

	"github.com/atotto/clipboard"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hibooboo2/slime_tag/assets"
	"github.com/hibooboo2/slime_tag/keys"
)

var spriteImage *ebiten.Image
var screenWidth, screenHeight int

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	monitor := ebiten.Monitor()
	if monitor == nil {
		log.Fatal("No primary monitor found")
	}

	screenWidth, screenHeight = monitor.Size()
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Sprite Viewer")
	ebiten.SetWindowResizable(true)

	log.Println("screenWidth", screenWidth, "screenHeight", screenHeight)

	// Create viewer instance after loading settings
	viewer := &Viewer{
		spriteSize: 32,
		spriteX:    4,
		spriteY:    26,
		keys:       keys.NewKeys(),
	}

	go viewer.HandleKeys()

	var err error
	spriteImage, _, err = ebitenutil.NewImageFromFileSystem(assets.Resources, os.Args[1]) // Load the sprite image
	if err != nil {
		panic(err)
	}

	if err := ebiten.RunGame(viewer); err != nil {
		log.Fatalf("Error running game: %v", err)
	}
}

type Viewer struct {
	spriteSize int
	spriteX    int
	spriteY    int
	keys       *keys.Keys
}

func (v *Viewer) HandleKeys() {
	for event := range v.keys.GetEventChan() {
		switch event.Key {
		case ebiten.KeyArrowUp:
			v.spriteY--
		case ebiten.KeyArrowDown:
			v.spriteY++
		case ebiten.KeyArrowLeft:
			v.spriteX--
		case ebiten.KeyArrowRight:
			v.spriteX++
		case ebiten.KeyEqual:
			v.spriteSize++
		case ebiten.KeyMinus:
			v.spriteSize--
		case ebiten.KeyEscape:
			os.Exit(0)
		case ebiten.KeyC:
			msg := fmt.Sprintf("v.getSubImage(spriteImage, %d, %d, %d)", v.spriteX, v.spriteY, v.spriteSize)
			copyToClipboard(msg)
		}

		// Ensure sprite is within screen bounds
		if v.spriteX < 0 {
			v.spriteX = 0
		} else if v.spriteX+v.spriteSize >= screenWidth {
			v.spriteX = screenWidth - v.spriteSize
		}
		if v.spriteY < 0 {
			v.spriteY = 0
		} else if v.spriteY+v.spriteSize >= screenHeight {
			v.spriteY = screenHeight - v.spriteSize
		}
	}
}

func (v *Viewer) Update() error {
	v.keys.Update()

	return nil
}

func (v *Viewer) getSubImage(spriteImage *ebiten.Image, x, y, size int) image.Image {
	rect := image.Rect(x*size, y*size, x*size+size, y*size+size)
	return spriteImage.SubImage(rect)
}

func (v *Viewer) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, "Sprite Viewer", 0, 0)

	msg := fmt.Sprintf("X: %d, Y: %d, Size: %d", v.spriteX, v.spriteY, v.spriteSize)
	// Display sprite position and size
	ebitenutil.DebugPrintAt(screen,
		msg,
		0,
		20)

	ebitenutil.DebugPrintAt(screen, "Press C to copy to clipboard", 0, 40)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(100, 100)

	screen.DrawImage(spriteImage, op)

	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(70, 20)
	op2.GeoM.Scale(2, 2)

	subImage := v.getSubImage(spriteImage, v.spriteX, v.spriteY, v.spriteSize)
	img := ebiten.NewImageFromImage(subImage)

	screen.DrawImage(img, op2)

	subImage2 := v.getSubImage(spriteImage, 5, 26, 32)
	img2 := ebiten.NewImageFromImage(subImage2)

	op3 := &ebiten.DrawImageOptions{}
	op3.GeoM.Translate(float64(v.spriteX)*float64(v.spriteSize)+100, float64(v.spriteY)*float64(v.spriteSize)+100)

	screen.DrawImage(img2, op3)
}

func (v *Viewer) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func copyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}
