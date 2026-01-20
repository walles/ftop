package ui

import (
	"testing"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/assert"
)

func TestBase(t *testing.T) {
	test_me := NewColorRamp(5.0, 7.0, twin.NewColorHex(0x000000), twin.NewColorHex(0xffffff))
	assert.Equal(t, test_me.AtValue(5.0), twin.NewColorHex(0x000000))
	assert.Equal(t, test_me.AtValue(7.0), twin.NewColorHex(0xffffff))

	assert.Equal(t, test_me.AtValue(6.0), twin.NewColorHex(0x808080))
}

func TestBackwardsRamp(t *testing.T) {
	test_me := NewColorRamp(1.0, -1.0, twin.NewColorHex(0x000000), twin.NewColorHex(0xffffff))
	assert.Equal(t, test_me.AtValue(1.0), twin.NewColorHex(0x000000))
	assert.Equal(t, test_me.AtValue(-1.0), twin.NewColorHex(0xffffff))

	assert.Equal(t, test_me.AtValue(0.0), twin.NewColorHex(0x808080))
}

func TestSingleValueRamp(t *testing.T) {
	test_me := NewColorRamp(3.0, 3.0, twin.NewColorHex(0x000000), twin.NewColorHex(0xffffff))

	// Should go for the first color to make top-to-bottom ramps look good
	assert.Equal(t, test_me.AtValue(3.0), twin.NewColorHex(0x000000))
}

func TestThreeColorRamp(t *testing.T) {
	test_me := NewColorRamp(0.0, 2.0, twin.NewColorHex(0xff0000), twin.NewColorHex(0x00ff00), twin.NewColorHex(0x0000ff))

	// Check the edges
	assert.Equal(t, test_me.AtValue(0.0), twin.NewColorHex(0xff0000))
	assert.Equal(t, test_me.AtValue(1.0), twin.NewColorHex(0x00ff00))
	assert.Equal(t, test_me.AtValue(2.0), twin.NewColorHex(0x0000ff))

	// Check the middles
	assert.Equal(t, test_me.AtValue(0.5), twin.NewColorHex(0x808000))
	assert.Equal(t, test_me.AtValue(1.5), twin.NewColorHex(0x008080))
}

func TestThreeColorsBackwards(t *testing.T) {
	test_me := NewColorRamp(2.0, 0.0, twin.NewColorHex(0xff0000), twin.NewColorHex(0x00ff00), twin.NewColorHex(0x0000ff))

	// Check the edges
	assert.Equal(t, test_me.AtValue(2.0), twin.NewColorHex(0xff0000))
	assert.Equal(t, test_me.AtValue(1.0), twin.NewColorHex(0x00ff00))
	assert.Equal(t, test_me.AtValue(0.0), twin.NewColorHex(0x0000ff))

	// Check the middles
	assert.Equal(t, test_me.AtValue(1.5), twin.NewColorHex(0x808000))
	assert.Equal(t, test_me.AtValue(0.5), twin.NewColorHex(0x008080))
}
