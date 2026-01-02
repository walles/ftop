package ui

import "github.com/walles/moor/v2/twin"

type LoadBar struct {
	leftXinclusive  int
	rightXinclusive int

	ramp    ColorRamp
	bgColor twin.Color // For antialiasing

	// Backwards means starting on the right and going left
	backwards bool
}

// The ramp should go from 0.0 to 1.0
func NewLoadBar(leftXinclusive, rightXinclusive int, ramp ColorRamp, bgColor twin.Color) LoadBar {
	return LoadBar{
		leftXinclusive:  leftXinclusive,
		rightXinclusive: rightXinclusive,
		ramp:            ramp,
		bgColor:         bgColor,
		backwards:       false,
	}
}

func NewBackwardsLoadBar(leftXinclusive, rightXinclusive int, ramp ColorRamp, bgColor twin.Color) LoadBar {
	return LoadBar{
		leftXinclusive:  leftXinclusive,
		rightXinclusive: rightXinclusive,
		ramp:            ramp,
		bgColor:         bgColor,
		backwards:       true,
	}
}

// Sets the background color of a cell based on the current load.
//
// Load fraction is between 0.0 and 1.0.
func (lb LoadBar) SetBgColor(updateMe *twin.Style, x int, loadFraction float64, antiAlias bool) {
	if x < lb.leftXinclusive || x > lb.rightXinclusive {
		return
	}

	width := lb.rightXinclusive - lb.leftXinclusive + 1

	// How many cells should be colored?
	loadCells := float64(width) * loadFraction

	// How far into the load bar are we?
	relativeX := float64(x - lb.leftXinclusive)
	if lb.backwards {
		relativeX = float64(width-1) - relativeX
	}

	// Counting from where we are now, how many more cells need filling?
	cellsLeftToColor := loadCells - relativeX
	if cellsLeftToColor <= 0.0 {
		// No load bar here
		return
	}

	barFraction := relativeX / float64(width)
	color := lb.ramp.AtValue(barFraction)
	if cellsLeftToColor >= 1.0 {
		// Full color cell
		*updateMe = updateMe.WithBackground(color)
		return
	}

	antiAliasAmount := cellsLeftToColor // This is now between 0.0 and 1.0
	if antiAlias {
		// Anti-aliasing for the load bar's edge
		*updateMe = updateMe.WithBackground(lb.bgColor.Mix(color, antiAliasAmount))
	} else if antiAliasAmount >= 0.5 {
		// Round up and fill the cell completely
		*updateMe = updateMe.WithBackground(color)
	}
}
