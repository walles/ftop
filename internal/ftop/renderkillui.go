package ftop

import (
	"fmt"
	"syscall"
	"time"

	"github.com/walles/ftop/internal/ui"
	"github.com/walles/moor/v2/twin"
)

func (u *Ui) renderKillUi(nextToScreenRow int) {
	w, h := u.screen.Size()

	// Dialog dimensions
	width := w - 6
	height := 5 // 3 content lines + 2 border lines

	// Horizontal positioning (centered)
	x0 := 3
	x1 := x0 + width - 1

	// Vertical positioning (above or below nextToScreenRow with 1 row gap)
	// Decide based on where there's more room
	roomAbove := nextToScreenRow - 1     // Rows available above (with 1 gap)
	roomBelow := h - nextToScreenRow - 2 // Rows available below (with 1 gap)

	var y0, y1 int
	if roomAbove >= roomBelow {
		// Place above
		y1 = nextToScreenRow - 2 // -2 for the gap
		y0 = y1 - (height - 1)
	} else {
		// Place below
		y0 = nextToScreenRow + 2 // +2 for the gap
		y1 = y0 + (height - 1)
	}

	// Clear the frame
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			u.screen.SetCell(x, y, twin.StyledRune{Rune: ' '})
		}
	}

	defer renderFrame(u.screen, u.theme, x0, y0, x1, y1, "Kill process")

	killer, ok := u.eventHandler.(*eventHandlerKill)
	if !ok {
		panic(fmt.Sprintf("Not a kill handler: %+v", u.eventHandler))
	}

	excuse := killer.getExcuse()
	if excuse != "" {
		// We have some excuse, tell the user the kill failed

		// "Failed to kill launchd(1): permission denied."
		// ""
		// "Press any key to continue."
		x := x0 + 1
		y := y0 + 1
		x += drawText(u.screen, x, y, x1, "Failed to kill "+killer.process.String()+": ", twin.StyleDefault)
		drawText(u.screen, x, y, x1,
			excuse,
			twin.StyleDefault.WithForeground(u.theme.HighlightedForeground()),
		)

		x = x0 + 1
		y += 2
		x += drawText(u.screen, x, y, x1, "Press ", u.theme.PromptActive())
		x += drawText(u.screen, x, y, x1, "any key", u.theme.PromptKey())
		drawText(u.screen, x, y, x1, " to continue.", u.theme.PromptActive())
		return
	}

	lastSignalTimestamp := killer.GetLastSignalTimestamp()
	if lastSignalTimestamp != nil {
		// Awaiting kill result

		x := x0 + 1
		y := y0 + 1
		x += drawText(u.screen, x, y, x1, "Killing ", twin.StyleDefault)
		x += drawText(u.screen, x, y, x1, killer.process.String(), twin.StyleDefault.WithForeground(u.theme.HighlightedForeground()))
		drawText(u.screen, x, y, x1, "...", twin.StyleDefault)

		y += 2
		x = x0 + 1
		signal := killer.GetLastSignal()
		name := signal.String()
		switch *signal {
		case syscall.SIGKILL:
			name = "SIGKILL"
		case syscall.SIGTERM:
			name = "SIGTERM"
		}
		drawText(u.screen, x, y, x1, name, twin.StyleDefault)

		loadBarRamp := ui.NewColorRamp(0.0, 1.0, u.theme.Background(), u.theme.HighlightedForeground())
		loadBar := ui.NewLoadBar(x0+1, x1-1, loadBarRamp)
		loadBar.SetWatermark(1.0) // Give the progress bar a background color

		elapsed := time.Since(*lastSignalTimestamp)
		loadFraction := elapsed.Seconds() / KillTimeout.Seconds()

		for fillX := x0 + 1; fillX < x1; fillX++ {
			loadBar.SetCellBackground(u.screen, fillX, y, loadFraction)
		}

		return
	}

	// Kill not attempted yet, tell user we are awaiting confirmation

	// "Press k to kill launchd(1)."
	x := x0 + 1
	x += drawText(u.screen, x, y0+1, x1, "Press ", u.theme.PromptActive())
	x += drawText(u.screen, x, y0+1, x1, "k", u.theme.PromptKey())
	x += drawText(u.screen, x, y0+1, x1, " to kill ", u.theme.PromptActive())
	x += drawText(u.screen, x, y0+1, x1,
		killer.process.String(),
		twin.StyleDefault.WithForeground(u.theme.HighlightedForeground()),
	)
	u.screen.SetCell(x, y0+1, twin.StyledRune{
		Rune:  '.',
		Style: u.theme.PromptActive(),
	})
}
