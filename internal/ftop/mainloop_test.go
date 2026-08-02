package ftop

import (
	"testing"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

// Shutdown requests come from goroutines that just crashed, and the main loop
// can be idle waiting for events when one arrives. If the crashed goroutine was
// the one producing those events, nothing else is going to wake the main loop
// up, so the request has to do it itself.
func TestRequestShutdownWakesTheMainLoop(t *testing.T) {
	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")

	ui.RequestShutdown()

	assert.Equal(t, ui.done.Load(), true)
	assert.Equal(t, len(ui.events) > 0, true)
}
