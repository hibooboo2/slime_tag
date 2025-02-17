package ecs

import (
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game interface{}

type Entities []any

type Drawer interface {
	Draw(screen *ebiten.Image, g Game)
	Remove() bool
}

type Overlapper interface {
	Overlaps(image.Rectangle, Game) bool
}

func (e *Entities) Draw(screen *ebiten.Image, g Game) {
	for i, entity := range *e {
		drawer, ok := entity.(Drawer)
		if ok {
			drawer.Draw(screen, g)
			if drawer.Remove() {
				log.Println("Removing entity: ", i)
				(*e)[i] = nil
			}
		}
	}
}

func (e *Entities) Add(entity any) {
	for i := range *e {
		if (*e)[i] == nil {
			(*e)[i] = entity
			log.Println("Added entity: ", i, "to existing spot.")
			return
		}
	}
	*e = append(*e, entity)
	log.Println("Added entity: ", len(*e)-1, "to new spot.")
}

func (e *Entities) Remove(entity any) bool {
	for i, ent := range *e {
		if ent == entity {
			(*e)[i] = nil
			return true
		}
	}
	return false
}
