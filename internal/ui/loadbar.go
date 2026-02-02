package ui

import "github.com/walles/moor/v2/twin"

type LoadBar struct {
	leftXinclusive  int
	rightXinclusive int

	ramp ColorRamp

	watermark float64

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

func (lb *LoadBar) SetWatermark(watermark float64) {
	lb.watermark = watermark
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
	watermarkCells := float64(width) * lb.watermark

	// How far into the load bar are we?
	relativeX := float64(x - lb.leftXinclusive)
	if lb.backwards {
		relativeX = float64(width-1) - relativeX
	}

	// If we're currently at cell 0 (relativeX = 0.0), we should color it if
	// cellsToColor >= 0.5. Or in other words, bail if cellsToColor < 0.5.
	if cellsToColor < (relativeX + 0.5) {
		// Are we below the watermark?
		if relativeX < (watermarkCells + 0.5) {
			// Yep, color with a dimmed color
			color := lb.ramp.colors[0].Mix(lb.ramp.colors[1], 0.2)
			currentCell := screen.GetCell(x, y)
			screen.SetCell(x, y, twin.StyledRune{
				Rune:  currentCell.Rune,
				Style: currentCell.Style.WithBackground(color),
			})
		}

		return
	}

	var barFraction float64

	// If we have two cells to color, 0 and 1, and relativeX can be either 0 or
	// 1, we want barFraction to be either 0.0 or 1.0.
	if cellsToColor > 1.0 {
		barFraction = relativeX / (cellsToColor - 1.0)
	} else {
		// We have at most one cell to color, anti-alias it
		barFraction = relativeX
	}

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

	if currentCell.Rune == ' ' {
		setTopAndBottomColors(screen, x, y, colorA, colorB)
		return
	}

	// Not a space, need to pick one color

	var style twin.Style
	if colorA != nil && colorB == nil {
		// Have A but not B
		style = currentCell.Style.WithBackground(*colorA)
	} else if colorA == nil && colorB != nil {
		// Have B but not A
		style = currentCell.Style.WithBackground(*colorB)
	} else {
		// Have both A and B, pick the one with less coverage so it's visible
		if cellsToColorA < cellsToColorB {
			style = currentCell.Style.WithBackground(*colorA)
		} else {
			style = currentCell.Style.WithBackground(*colorB)
		}
	}

	screen.SetCell(x, y, twin.StyledRune{Rune: currentCell.Rune, Style: style})
}

// Replaces a cell with a half-block character colored with the given top and /
// or bottom colors.
func setTopAndBottomColors(screen twin.Screen, x int, y int, topColor, bottomColor *twin.Color) {
	if topColor == nil && bottomColor == nil {
		panic("Either top or bottom color or both must be set")
	}

	if topColor != nil && bottomColor == nil {
		// Color only the top half of the cell
		style := twin.StyleDefault.WithForeground(*topColor)
		screen.SetCell(x, y, twin.StyledRune{Rune: '▀', Style: style})
		return
	}

	if bottomColor != nil && topColor == nil {
		// Color only the bottom half of the cell
		style := twin.StyleDefault.WithForeground(*bottomColor)
		screen.SetCell(x, y, twin.StyledRune{Rune: '▄', Style: style})
		return
	}

	// Color both top and bottom halves
	style := twin.StyleDefault.WithForeground(*topColor).WithBackground(*bottomColor)
	screen.SetCell(x, y, twin.StyledRune{Rune: '▀', Style: style})
}
