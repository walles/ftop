package ui

import (
	"testing"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/assert"
)

func TestMinMax(t *testing.T) {
	test_me := NewColorRamp(twin.NewColorHex(0x000000), twin.NewColorHex(0xffffff), 5.0, 7.0)
	assert.Equal(t, test_me.AtValue(5.0), twin.NewColorHex(0x000000))
	assert.Equal(t, test_me.AtValue(7.0), twin.NewColorHex(0xffffff))
}

func TestBackwardsRamp(t *testing.T) {
	test_me := NewColorRamp(twin.NewColorHex(0x000000), twin.NewColorHex(0xffffff), 1.0, -1.0)
	assert.Equal(t, test_me.AtValue(1.0), twin.NewColorHex(0x000000))
	assert.Equal(t, test_me.AtValue(-1.0), twin.NewColorHex(0xffffff))
}
