package wfc

import (
	"encoding/json"
	"io/fs"
	"slices"
	"strconv"

	"golang.org/x/exp/rand"
)

const (
	Width  = 40
	Height = 20
)

type RulesCell struct {
	X        int
	Y        int
	Occupied bool
}

type ConstraintToSolveFor struct {
	Rules [Height][Width]RulesCell
}

func NewConstraintToSolveFor(fs fs.FS, filename string, spriteSize int) (*ConstraintToSolveFor, error) {
	var cells []struct {
		X  int
		Y  int
		ID string `json:"id"`
	}
	data, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(data)
	dec.DisallowUnknownFields()
	err = dec.Decode(&cells)
	if err != nil {
		return nil, err
	}

	c := ConstraintToSolveFor{}

	// constraints := make(map[string]*Constraint)
	for _, cell := range cells {
		id, err := strconv.Atoi(cell.ID)
		if err != nil {
			continue
		}
		x := id / Width
		y := id % Height
		// log.Printf("cell: %s -> x: %d y: %d IMG: x: %d y: %d", cell.ID, x, y, cell.X, cell.Y)

		c.Rules[x][y] = RulesCell{X: -cell.X / spriteSize, Y: -cell.Y / spriteSize, Occupied: true}
	}

	// for y := range c.rules {
	// 	for x := range c.rules[y] {
	// 		id := fmt.Sprintf("%d,%d", x, y)
	// 		constraint, ok := constraints[id]
	// 		if !ok {
	// 			constraint = &Constraint{
	// 				Value:              id,
	// 				AllowableNeighbors: [3][3][]string{},
	// 			}
	// 			constraints[id] = constraint
	// 		}
	// 		//XXX need to implement this. Use rules to create constraints.
	// 		// Add each neighbor as a poissble allowed for each usage of the id.
	// 	}
	// }

	// return constraints
	return &c, nil
}

type Grid struct {
	Data        [Height][Width]*Cell
	Constraints map[string]Constraint
}

type Cell struct {
	AllowedValues []string
	Value         string
	Collapsed     bool
}

type Constraint struct {
	Value              string
	AllowableNeighbors [3][3][]string
}

func NewGrid(constraints map[string]Constraint) *Grid {
	possibleValues := []string{}
	for c := range constraints {
		possibleValues = append(possibleValues, c)
	}
	g := &Grid{
		Constraints: constraints,
	}
	for y := range g.Data {
		for x := range g.Data[y] {
			myPossibleValues := make([]string, len(possibleValues))
			copy(myPossibleValues, possibleValues)
			g.Data[y][x] = &Cell{
				AllowedValues: myPossibleValues,
			}
		}
	}
	return g
}

type Loc struct {
	X int
	Y int
}

func (g *Grid) CollapseCells() {
	for {
		lowestEntropy := -1
		lowestEntropyLoc := []Loc{}
		for y := range g.Data {
			for x := range g.Data[y] {
				cell := g.Data[y][x]
				if cell.Collapsed || len(cell.AllowedValues) == 0 {
					continue
				}
				if lowestEntropy == -1 || len(cell.AllowedValues) < lowestEntropy {
					lowestEntropy = len(cell.AllowedValues)
					lowestEntropyLoc = []Loc{
						{
							X: x,
							Y: y,
						},
					}
				} else if lowestEntropy == len(cell.AllowedValues) {
					lowestEntropyLoc = append(lowestEntropyLoc, Loc{
						X: x,
						Y: y,
					})
				}
			}
		}
		if lowestEntropy == -1 {
			break
		}
		loc := lowestEntropyLoc[rand.Intn(len(lowestEntropyLoc))]
		g.CollapseCell(loc)
		g.PropagateChanges(loc)
	}
}

func (g *Grid) CollapseCell(loc Loc) {
	allowedValues := g.Data[loc.Y][loc.X].AllowedValues
	newValue := allowedValues[rand.Intn(len(allowedValues))]
	g.Data[loc.Y][loc.X].Value = newValue
	g.Data[loc.Y][loc.X].Collapsed = true
}

func (g *Grid) PropagateChanges(lastCellCollapsed Loc) {
	cellsToPropagate := []Loc{lastCellCollapsed}
	for len(cellsToPropagate) > 0 {
		newCellsToPropagate := []Loc{}
		for _, cell := range cellsToPropagate {
			constraint := g.Constraints[g.Data[cell.Y][cell.X].Value]
			x, y := cell.X, cell.Y

			// Propagate to neighbors
			if g.PropagateCell(x, y, -1, -1, constraint) {
				newCellsToPropagate = append(newCellsToPropagate, Loc{
					X: x - 1,
					Y: y - 1,
				})
			}
			if g.PropagateCell(x, y, 0, -1, constraint) {
				newCellsToPropagate = append(newCellsToPropagate, Loc{
					X: x,
					Y: y - 1,
				})
			}
			if g.PropagateCell(x, y, 1, -1, constraint) {
				newCellsToPropagate = append(newCellsToPropagate, Loc{
					X: x + 1,
					Y: y - 1,
				})
			}
			if g.PropagateCell(x, y, -1, 0, constraint) {
				newCellsToPropagate = append(newCellsToPropagate, Loc{
					X: x - 1,
					Y: y,
				})
			}
			if g.PropagateCell(x, y, 1, 0, constraint) {
				newCellsToPropagate = append(newCellsToPropagate, Loc{
					X: x + 1,
					Y: y,
				})
			}
			if g.PropagateCell(x, y, -1, 1, constraint) {
				newCellsToPropagate = append(newCellsToPropagate, Loc{
					X: x - 1,
					Y: y + 1,
				})
			}
			if g.PropagateCell(x, y, 0, 1, constraint) {
				newCellsToPropagate = append(newCellsToPropagate, Loc{
					X: x,
					Y: y + 1,
				})
			}
			if g.PropagateCell(x, y, 1, 1, constraint) {
				newCellsToPropagate = append(newCellsToPropagate, Loc{
					X: x + 1,
					Y: y + 1,
				})
			}
		}
		cellsToPropagate = newCellsToPropagate
	}
}

func (g *Grid) PropagateCell(x, y int, xoff int, yoff int, constraint Constraint) bool {
	didChange := false
	if x+xoff >= 0 && y+yoff >= 0 && x+xoff < Width && y+yoff < Height {
		newAllowedValues := []string{}
		for _, val := range g.Data[y+yoff-1][x+xoff-1].AllowedValues {
			if !slices.Contains(constraint.AllowableNeighbors[1+yoff][1+xoff], val) {
				didChange = true
				continue
			}
			newAllowedValues = append(newAllowedValues, val)
		}
		g.Data[y+yoff-1][x+xoff-1].AllowedValues = newAllowedValues
	}
	return didChange
}
