package ui

import (
	"github.com/walles/moor/v2/twin"
)

type ColorRamp struct {
	colors []twin.Color

	from float64
	to   float64
}

func NewColorRamp(from float64, to float64, c0 twin.Color, c1 twin.Color, extraColors ...twin.Color) ColorRamp {
	return ColorRamp{
		colors: append([]twin.Color{c0, c1}, extraColors...),
		from:   from,
		to:     to,
	}
}

func (cr ColorRamp) AtInt(value int) twin.Color {
	return cr.AtValue(float64(value))
}

func (cr ColorRamp) AtValue(value float64) twin.Color {
	var fraction float64
	if cr.to == cr.from {
		// Single-value ramp, always return the first color
		fraction = 0
	} else {
		fraction = (value - cr.from) / (cr.to - cr.from)
	}
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}

	if fraction == 1.0 {
		return cr.colors[len(cr.colors)-1]
	}

	c0Index := int(fraction * float64(len(cr.colors)-1))
	c1Index := c0Index + 1

	c0 := cr.colors[c0Index]
	c1 := cr.colors[c1Index]

	innerFraction := (fraction * float64(len(cr.colors)-1)) - float64(c0Index)

	return c0.Mix(c1, innerFraction)
}
