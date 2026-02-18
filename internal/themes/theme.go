package themes

import (
	"fmt"

	"github.com/walles/moor/v2/twin"
)

// NOTE: Use some online OKLCH color picker for experimenting with colors
type Theme struct {
	terminalBackground *twin.Color // nil means unknown
	fallbackBackground twin.Color  // Used if terminalBackground is nil

	highlightedForeground twin.Color

	// FIXME: Split into terminalForeground and fallbackForeground?
	foreground twin.Color

	loadBarMaxCpu twin.Color
	loadBarMaxRam twin.Color
	loadBarMaxIO  twin.Color

	border      twin.Color
	borderTitle twin.Color
}

func NewTheme(name string, bg *twin.Color) Theme {
	if name == "dark" {
		return newDarkTheme(bg)
	}

	if name == "light" {
		return newLightTheme(bg)
	}

	if name != "auto" {
		panic(fmt.Errorf("invalid theme name: <%s>", name))
	}

	if bg == nil {
		return newDarkTheme(nil)
	}

	distanceToBlack := bg.Distance(twin.NewColorHex(0x000000))
	distanceToWhite := bg.Distance(twin.NewColorHex(0xffffff))

	if distanceToBlack < distanceToWhite {
		return newDarkTheme(bg)
	} else {
		return newLightTheme(bg)
	}
}

func newDarkTheme(bg *twin.Color) Theme {
	return Theme{
		terminalBackground: bg,
		fallbackBackground: twin.NewColorHex(0x000000),
		foreground:         twin.NewColorHex(0xdddddd),

		highlightedForeground: twin.NewColorHex(0xbdebbe),

		loadBarMaxCpu: twin.NewColorHex(0x5f1f22),
		loadBarMaxRam: twin.NewColorHex(0x1e3568),
		loadBarMaxIO:  twin.NewColorHex(0xd0d020),

		border:      twin.NewColorHex(0x7070a0),
		borderTitle: twin.NewColorHex(0xffc0c0),
	}
}

func newLightTheme(bg *twin.Color) Theme {
	return Theme{
		terminalBackground: bg,
		fallbackBackground: twin.NewColorHex(0xffffff),
		foreground:         twin.NewColorHex(0x000000),

		highlightedForeground: twin.NewColorHex(0x009000),

		loadBarMaxCpu: twin.NewColorHex(0xffcccc),
		loadBarMaxRam: twin.NewColorHex(0xccccff),
		loadBarMaxIO:  twin.NewColorHex(0xaaaa55),

		border:      twin.NewColorHex(0x9090e0),
		borderTitle: twin.NewColorHex(0xb77d7d),
	}
}

func (t Theme) Background() twin.Color {
	if t.terminalBackground != nil {
		return *t.terminalBackground
	}
	return t.fallbackBackground
}

func (t Theme) Foreground() twin.Color {
	return t.foreground
}

func (t Theme) FadedForeground() twin.Color {
	return t.Foreground().Mix(t.Background(), 0.5)
}

func (t Theme) HighlightedForeground() twin.Color {
	return t.highlightedForeground
}

func (t Theme) LoadBarMin() twin.Color {
	return t.Background()
}

func (t Theme) LoadBarMaxRam() twin.Color {
	return t.loadBarMaxRam
}

func (t Theme) LoadBarMaxCpu() twin.Color {
	return t.loadBarMaxCpu
}

func (t Theme) LoadBarMaxIO() twin.Color {
	return t.loadBarMaxIO
}

func (t Theme) Border() twin.Color {
	return t.border
}

func (t Theme) BorderTitle() twin.Color {
	return t.borderTitle
}

func (t Theme) PromptActive() twin.Style {
	return twin.StyleDefault.WithForeground(t.Background().Mix(t.HighlightedForeground(), 0.7))
}

func (t Theme) PromptPassive() twin.Style {
	return twin.StyleDefault.WithForeground(t.HighlightedForeground()).WithAttr(twin.AttrDim)
}

// Style for a single cell containing a key that can be pressed to trigger an
// action, e.g. "K" in "Kill".
func (t Theme) PromptKey() twin.Style {
	return twin.StyleDefault.WithForeground(t.HighlightedForeground()).WithAttr(twin.AttrBold)
}
