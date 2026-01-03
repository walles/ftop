package ui

import "github.com/walles/moor/v2/twin"

type LoadBar struct {
	leftXinclusive  int
	rightXinclusive int

	ramp ColorRamp

	// Backwards means starting on the right and going left
	backwards bool
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

// Sets the background color of a cell based on the current load.
//
// Load fraction is between 0.0 and 1.0.
func (lb LoadBar) SetBgColor(updateMe *twin.Style, x int, loadFraction float64) {
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

	*updateMe = updateMe.WithBackground(color)
}
