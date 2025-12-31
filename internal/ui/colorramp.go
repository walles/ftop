package ui

import (
	"fmt"

	"github.com/walles/moor/v2/twin"
)

type ColorRamp struct {
	c0 twin.Color
	c1 twin.Color

	from float64
	to   float64
}

func NewColorRamp(c0 twin.Color, c1 twin.Color, from float64, to float64) ColorRamp {
	if to == from {
		panic(fmt.Sprintf("cannot ramp when from=to: %f", from))
	}

	return ColorRamp{
		c0:   c0,
		c1:   c1,
		from: from,
		to:   to,
	}
}

func (cr ColorRamp) AtInt(value int) twin.Color {
	return cr.AtValue(float64(value))
}

func (cr ColorRamp) AtValue(value float64) twin.Color {
	fraction := (value - cr.from) / (cr.to - cr.from)
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}

	return cr.c0.Mix(cr.c1, fraction)
}
