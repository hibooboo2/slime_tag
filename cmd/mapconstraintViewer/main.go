package main

import (
	"image"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hibooboo2/slime_tag/assets"
	"github.com/hibooboo2/slime_tag/keys"
)

var spriteImage *ebiten.Image
var screenWidth, screenHeight int

func main() {
	file, err := os.Open("./constraints.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

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
		keys: keys.NewKeys(),
	}

	go viewer.HandleKeys()

	spriteImage, _, err = ebitenutil.NewImageFromFileSystem(assets.Resources, "resources/slimes/ProjectUtumno_full.png") // Load the sprite image
	if err != nil {
		panic(err)
	}

	if err := ebiten.RunGame(viewer); err != nil {
		log.Fatalf("Error running game: %v", err)
	}
}

type Viewer struct {
	keys *keys.Keys
}

func (v *Viewer) HandleKeys() {
	for event := range v.keys.GetEventChan() {
		switch event.Key {
		case ebiten.KeyEscape:
			os.Exit(0)
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

	ebitenutil.DebugPrintAt(screen, "Press C to copy to clipboard", 0, 40)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(screenWidth)/4, float64(screenHeight)/4)

	// viewPanel := spriteImage.SubImage()

	// screen.DrawImage(viewPanel.(*ebiten.Image), op)

}

func (v *Viewer) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}
