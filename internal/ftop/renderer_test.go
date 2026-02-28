package ftop

import (
	"strings"
	"testing"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

func TestRender_ShowsProcessInfoPaneImmediatelyFromPickedLine(t *testing.T) {
	screen := twin.NewFakeScreen(80, 24)
	ui := NewUi(screen, themes.NewTheme("auto", nil))

	pickedLine := 0
	ui.pickedLine = &pickedLine
	ui.pickedProcess = nil

	processesRaw := []processes.Process{
		{Pid: 42, Command: "picked", Username: "testuser", RssKb: 1000, CpuTime: toDuration(100)},
	}

	ui.Render(processesRaw, nil, nil)

	if ui.pickedProcess == nil {
		t.Fatalf("expected pickedProcess to be resolved during render")
	}

	assert.Equal(t, screenContainsText(screen, "Process Info"), true)
	assert.Equal(t, screenContainsText(screen, "Launched Commands"), false)
}

func screenContainsText(screen twin.Screen, text string) bool {
	width, height := screen.Size()
	for y := range height {
		rowRunes := make([]rune, 0, width)
		for x := range width {
			rowRunes = append(rowRunes, screen.GetCell(x, y).Rune)
		}

		if strings.Contains(string(rowRunes), text) {
			return true
		}
	}

	return false
}
