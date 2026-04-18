package ftop

import (
	"fmt"
	"time"

	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/ui"
	"github.com/walles/ftop/internal/util"
	"github.com/walles/moor/v2/twin"
)

func (u *Ui) renderProcessInfoPane(y0, y1 int) {
	if u.pickedProcess == nil {
		panic("no process picked")
	}

	width, _ := u.screen.Size()
	x1 := width - 1

	defer renderFrame(u.screen, u.theme, 0, y0, x1, y1, "Process Info")

	// Render the process hierarchy

	bottomUpNames := make([]string, 0)
	for p := u.pickedProcess; p != nil; p = p.Parent() {
		bottomUpNames = append(bottomUpNames, p.Command())
	}

	// Always show the current process as highlighted. Before that, go from faded to almost-foreground.
	ramp := ui.NewColorRamp(0, float64(len(bottomUpNames))-1, u.theme.Foreground(), u.theme.FadedForeground())

	highlighted := twin.StyleDefault.WithForeground(u.theme.HighlightedForeground())
	plain := twin.StyleDefault.WithForeground(u.theme.Foreground())

	hierarchy := make([]twin.StyledRune, 0)
	for i := len(bottomUpNames) - 1; i >= 0; i-- {
		if len(hierarchy) > 0 {
			style := twin.StyleDefault.WithForeground(u.theme.FadedForeground())
			hierarchy = append(hierarchy, twin.StyledRune{Rune: '―', Style: style})
			hierarchy = append(hierarchy, twin.StyledRune{Rune: '―', Style: style})
		}

		var color twin.Color
		if i == 0 {
			// Always highlight the current process
			color = u.theme.HighlightedForeground()
		} else {
			color = ramp.AtInt(i)
		}
		style := twin.StyleDefault.WithForeground(color)

		name := bottomUpNames[i]
		for _, r := range name {
			hierarchy = append(hierarchy, twin.StyledRune{Rune: r, Style: style})
		}
	}

	// If the hierarchy string is too long, take out a part in the middle
	availableWidth := x1 - 1
	hierarchy = u.truncateToLength(hierarchy, availableWidth)

	y := y0 + 1
	for i, styledRune := range hierarchy {
		x := i + 1
		u.screen.SetCell(x, y, styledRune)
	}

	// Render timing info

	y += 2
	x := 1
	x += drawText(u.screen, x, y, x1, "Started ", plain)
	x += drawText(u.screen, x, y, x1, util.FormatDuration(time.Since(u.pickedProcess.StartTime())), highlighted)
	x += drawText(u.screen, x, y, x1, " ago at ", plain)

	startString := u.pickedProcess.StartTime().Format(DISPLAY_TIME_FORMAT)
	x += drawText(u.screen, x, y, x1, startString, highlighted)

	x += drawText(u.screen, x, y, x1, ". It used ", plain)
	x += drawText(u.screen, x, y, x1, util.FormatDuration(u.pickedProcess.CpuTimeOrZero()), highlighted)
	x += drawText(u.screen, x, y, x1, " CPU, or ", plain)

	percentCpu := 100.0 * float64(u.pickedProcess.CpuTimeOrZero()) / float64(time.Since(u.pickedProcess.StartTime()))
	x += drawText(u.screen, x, y, x1, util.FormatPercent(percentCpu), highlighted)
	drawText(u.screen, x, y, x1, ".", plain)

	// Render nativity info

	style := plain
	if u.pickedProcess.Nativity > 0 {
		style = highlighted
	}
	description := "spawned no child processes"
	if u.pickedProcess.Nativity > 1 {
		description = fmt.Sprintf("spawned %d child processes", u.pickedProcess.Nativity)
	} else if u.pickedProcess.Nativity == 1 {
		description = "spawned 1 child process"
	}

	y += 2
	x = 1
	x += drawText(u.screen, x, y, x1, u.pickedProcess.String(), plain)
	x += drawText(u.screen, x, y, x1, " ", plain)
	x += drawText(u.screen, x, y, x1, description, style)
	x += drawText(u.screen, x, y, x1, " in the last ", plain)
	x += drawText(u.screen, x, y, x1, util.FormatDuration(processes.NATIVITY_MAX_AGE), plain)
	drawText(u.screen, x, y, x1, ".", plain)
}

// If the hierarchy string is too long, take out a part in the middle
func (u *Ui) truncateToLength(runes []twin.StyledRune, maxLength int) []twin.StyledRune {
	if len(runes) <= maxLength {
		return runes
	}

	removeCount := len(runes) - maxLength + 1 // Plus one for the space we're inserting in the middle
	removeStartInclusive := (len(runes) - removeCount) / 2
	removeEndExclusive := removeStartInclusive + removeCount

	spaceIndex := removeStartInclusive

	truncated := make([]twin.StyledRune, maxLength)
	copy(truncated, runes[:removeStartInclusive])
	truncated[removeStartInclusive] = twin.StyledRune{Rune: ' ', Style: twin.StyleDefault}
	copy(truncated[removeStartInclusive+1:], runes[removeEndExclusive:])

	if len(truncated) >= 5 {
		// FIXME: Should we fade like this or should we have a more obvious ellipsis instead?
		u.fadeStyleTowardsBackground(&truncated[spaceIndex-2], 1.0/3.0)
		u.fadeStyleTowardsBackground(&truncated[spaceIndex-1], 2.0/3.0)
		u.fadeStyleTowardsBackground(&truncated[spaceIndex+1], 2.0/3.0)
		u.fadeStyleTowardsBackground(&truncated[spaceIndex+2], 1.0/3.0)
	}

	return truncated
}

// Higher fraction means closer to the background color
func (u *Ui) fadeStyleTowardsBackground(styledRune *twin.StyledRune, fraction float64) {
	faded := styledRune.Style.Foreground().Mix(u.theme.Background(), fraction)
	styledRune.Style = styledRune.Style.WithForeground(faded)
}
