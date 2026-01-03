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

func (olb OverlappingLoadBars) SetBgColor(updateMe *twin.Style, x int, loadFractionA, loadFractionB float64) {
	if loadFractionA > loadFractionB {
		// Render the shorter one (B) last, so it ends up in front of A
		olb.a.SetBgColor(updateMe, x, loadFractionA)
		olb.b.SetBgColor(updateMe, x, loadFractionB)
	} else {
		// Render the shorter one (A) last, so it ends up in front of B
		olb.b.SetBgColor(updateMe, x, loadFractionB)
		olb.a.SetBgColor(updateMe, x, loadFractionA)
	}
}
