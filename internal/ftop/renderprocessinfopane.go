package ftop

import (
	"time"

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
		bottomUpNames = append(bottomUpNames, p.Command)
	}

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

		name := bottomUpNames[i]
		color := ramp.AtInt(i)
		style := twin.StyleDefault.WithForeground(color)

		if i == 0 {
			style = highlighted
		}

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

	y += 1
	x := 1
	x += drawText(u.screen, x, y, x1, "Started ", plain)
	x += drawText(u.screen, x, y, x1, util.FormatDuration(time.Since(u.pickedProcess.StartTime())), highlighted)
	x += drawText(u.screen, x, y, x1, " ago at ", plain)

	startString := u.pickedProcess.StartTime().Format("2006-01-02 15:04:05")
	drawText(u.screen, x, y, x1, startString, highlighted)
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
