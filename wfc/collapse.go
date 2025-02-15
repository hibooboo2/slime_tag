package wfc

import "golang.org/x/exp/rand"

const (
	Width  = 80
	Height = 40
)

type Grid struct {
	Data [Height][Width]*Cell
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

var (
	constraints = map[string]Constraint{}
)

func NewGrid() *Grid {
	possibleValues := []string{}
	for c := range constraints {
		possibleValues = append(possibleValues, c)
	}
	g := &Grid{}
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

func (g *Grid) CollapseLowestEntropyCell() {
	for {
		lowestEntropy := -1
		lowestEntropyLoc := []Loc{}
		for y := range g.Data {
			for x := range g.Data[y] {
				cell := g.Data[y][x]
				if cell.Collapsed {
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
		loc := lowestEntropyLoc[rand.Intn(len(lowestEntropyLoc))]
		g.CollapseCell(loc.X, loc.Y)
	}
}

func (g *Grid) CollapseCell(x, y int) {
	allowedValues := g.Data[y][x].AllowedValues
	newValue := allowedValues[rand.Intn(len(allowedValues))]
	g.Data[y][x].Value = newValue
	g.Data[y][x].Collapsed = true
}

func (g *Grid) PropagateChanges(lastCellCollapsed Loc) {
	cellsToPropagate := []Loc{lastCellCollapsed}
	for {
		for _, cell := range cellsToPropagate {
			constraint := constraints[g.Data[cell.Y][cell.X].Value]
			for y := range constraint.AllowableNeighbors {
				for x := range constraint.AllowableNeighbors[y] {
					// Need to compare allowable neightbor values to allowed values of the cells
					// that are neighbors of the last cell collapsed.
					//Then add the neighbors to the last cell collapsed if any changed
				}
			}
		}
	}
}
