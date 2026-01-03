package ui

import "github.com/walles/moor/v2/twin"

type LoadBar struct {
	leftXinclusive  int
	rightXinclusive int

	ramp ColorRamp

	// Backwards means starting on the right and going left
	backwards bool
}

type OverlappingLoadBars struct {
	a LoadBar
	b LoadBar
}

// The ramp should go from 0.0 to 1.0
func NewLoadBar(leftXinclusive, rightXinclusive int, ramp ColorRamp) LoadBar {
	return LoadBar{
		leftXinclusive:  leftXinclusive,
		rightXinclusive: rightXinclusive,
		ramp:            ramp,
		backwards:       false,
	}
}

func NewBackwardsLoadBar(leftXinclusive, rightXinclusive int, ramp ColorRamp) LoadBar {
	return LoadBar{
		leftXinclusive:  leftXinclusive,
		rightXinclusive: rightXinclusive,
		ramp:            ramp,
		backwards:       true,
	}
}

func NewOverlappingLoadBars(leftXinclusive, rightXinclusive int, rampA ColorRamp, rampB ColorRamp) OverlappingLoadBars {
	return OverlappingLoadBars{
		a: NewLoadBar(leftXinclusive, rightXinclusive, rampA),
		b: NewLoadBar(leftXinclusive, rightXinclusive, rampB),
	}
}

// Sets the background color of a cell based on the current load.
//
// Load fraction is between 0.0 and 1.0.
func (lb LoadBar) SetCellBackground(screen twin.Screen, x int, y int, loadFraction float64) {
	if x < lb.leftXinclusive || x > lb.rightXinclusive {
		return
	}

	width := lb.rightXinclusive - lb.leftXinclusive + 1

	// How many cells should be colored?
	cellsToColor := float64(width) * loadFraction

	// How far into the load bar are we?
	relativeX := float64(x - lb.leftXinclusive)
	if lb.backwards {
		relativeX = float64(width-1) - relativeX
	}

	// If we're currently at cell 0 (relativeX = 0.0), we should color it if
	// cellsToColor >= 0.5. Or in other words, bail if cellsToColor < 0.5.
	if cellsToColor < (relativeX + 0.5) {
		return
	}

	barFraction := relativeX / cellsToColor
	color := lb.ramp.AtValue(barFraction)

	currentCell := screen.GetCell(x, y)
	screen.SetCell(x, y, twin.StyledRune{
		Rune:  currentCell.Rune,
		Style: currentCell.Style.WithBackground(color),
	})
}

func (olb OverlappingLoadBars) SetCellBackground(screen twin.Screen, x int, y int, loadFractionA, loadFractionB float64) {
	if x < olb.a.leftXinclusive || x > olb.a.rightXinclusive {
		return
	}

	width := olb.a.rightXinclusive - olb.a.leftXinclusive + 1

	// How many cells should be colored?
	cellsToColorA := float64(width) * loadFractionA
	cellsToColorB := float64(width) * loadFractionB

	// How far into the load bar are we?
	relativeX := float64(x - olb.a.leftXinclusive)
	if olb.a.backwards {
		relativeX = float64(width-1) - relativeX
	}

	barFractionA := relativeX / cellsToColorA
	var colorA *twin.Color
	if cellsToColorA >= (relativeX + 0.5) {
		color := olb.a.ramp.AtValue(barFractionA)
		colorA = &color
	}

	barFractionB := relativeX / cellsToColorB
	var colorB *twin.Color
	if cellsToColorB >= (relativeX + 0.5) {
		color := olb.b.ramp.AtValue(barFractionB)
		colorB = &color
	}

	if colorA == nil && colorB == nil {
		return
	}

	currentCell := screen.GetCell(x, y)

	if colorA != nil && colorB == nil {
		// Have A but not B
		style := currentCell.Style.WithBackground(*colorA)
		screen.SetCell(x, y, twin.StyledRune{Rune: currentCell.Rune, Style: style})
		return
	}

	if colorB != nil && colorA == nil {
		// Have B but not A
		style := currentCell.Style.WithBackground(*colorB)
		screen.SetCell(x, y, twin.StyledRune{Rune: currentCell.Rune, Style: style})
		return
	}

	// Have both A and B

	if currentCell.Rune != ' ' {
		// Set background based on whichever bar is shortest, since that one
		// should be on top so it's visible
		style := currentCell.Style.WithBackground(*colorA)
		if cellsToColorB < cellsToColorA {
			style = currentCell.Style.WithBackground(*colorB)
		}
		screen.SetCell(x, y, twin.StyledRune{Rune: currentCell.Rune, Style: style})
		return
	}

	// Have both A and B, and since the current cell is ' ' we can show both
	// using half-cell Unicode characters. In this case we just put A on top.
	halfBlockRune := '▀'
	style := currentCell.Style.WithBackground(*colorB).WithForeground(*colorA)
	screen.SetCell(x, y, twin.StyledRune{Rune: halfBlockRune, Style: style})
}
