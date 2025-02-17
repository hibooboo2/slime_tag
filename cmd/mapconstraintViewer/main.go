package main

import (
	"fmt"
	"image"
	"io/fs"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hibooboo2/slime_tag/assets"
	"github.com/hibooboo2/slime_tag/keys"
	"github.com/hibooboo2/slime_tag/wfc"
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
		keys: keys.NewKeys(),
	}

	go viewer.HandleKeys()

	var err error
	spriteImage, _, err = ebitenutil.NewImageFromFileSystem(assets.Resources, "resources/slimes/ProjectUtumno_full.png") // Load the sprite image
	if err != nil {
		panic(err)
	}

	if err := ebiten.RunGame(viewer); err != nil {
		log.Fatalf("Error running game: %v", err)
	}
}

type Viewer struct {
	keys        *keys.Keys
	constraints *wfc.ConstraintToSolveFor
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

	files := ebiten.DroppedFiles()
	if files != nil {
		err := fs.WalkDir(files, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Println("error walking directory", err)
			}
			log.Println("dropped file", path)
			info, err := d.Info()
			if err != nil {
				return fmt.Errorf("error getting info: %v", err)
			}
			if info.IsDir() {
				return nil
			}
			constraints, err := wfc.NewConstraintToSolveFor(files, path, 32)
			if err != nil {
				return nil
			}
			v.constraints = constraints
			return nil
		})
		if err != nil {
			return fmt.Errorf("error walking directory: %v", err)
		}
	}

	return nil
}

func (v *Viewer) getSubImage(spriteImage *ebiten.Image, x, y, size int) image.Image {
	rect := image.Rect(x*size, y*size, x*size+size, y*size+size)
	return spriteImage.SubImage(rect)
}

func (v *Viewer) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, "Map Constraint Viewer", 0, 0)
	if v.constraints == nil {
		ebitenutil.DebugPrintAt(screen, "No constraints loaded", 0, 50)
		return
	}
	for y := range v.constraints.Rules {
		for x := range v.constraints.Rules[y] {
			if !v.constraints.Rules[y][x].Occupied {
				continue
			}
			tile := v.getSubImage(spriteImage, v.constraints.Rules[y][x].X, v.constraints.Rules[y][x].Y, 32)

			op := &ebiten.DrawImageOptions{}
			corX, corY := float64(screenWidth)/4+float64(x*32), float64(screenHeight)/4+float64(y*32)
			op.GeoM.Translate(corX, corY)
			ebitenutil.DebugPrintAt(screen, "X", int(corX), int(corY))
			screen.DrawImage(tile.(*ebiten.Image), op)
		}
	}
}

func (v *Viewer) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}
