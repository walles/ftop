package ftop

import (
	"testing"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

func TestNewUi_SetsInitialFilter(t *testing.T) {
	screen := twin.NewFakeScreen(80, 24)

	ui := NewUi(screen, themes.NewTheme("auto", nil), "firefox")

	assert.Equal(t, ui.filter, "firefox")
}

func TestNewUi_WithoutInitialFilter(t *testing.T) {
	screen := twin.NewFakeScreen(80, 24)

	ui := NewUi(screen, themes.NewTheme("auto", nil), "")

	assert.Equal(t, ui.filter, "")
}
