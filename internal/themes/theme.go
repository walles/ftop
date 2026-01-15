package themes

import "github.com/walles/moor/v2/twin"

type Theme struct {
	terminalBackground *twin.Color // nil means unknown
	fallbackBackground twin.Color  // Used if terminalBackground is nil

	// FIXME: Split into terminalForeground and fallbackForeground?
	foreground twin.Color

	loadBarMin    twin.Color
	loadBarMaxRam twin.Color
	loadBarMaxCpu twin.Color
	loadBarMaxIO  twin.Color

	border      twin.Color
	borderTitle twin.Color

	loadLow    twin.Color
	loadMedium twin.Color
	loadHigh   twin.Color
}

// NOTE: Use some online OKLCH color picker for experimenting with colors

func NewDarkTheme(bg *twin.Color) Theme {
	return Theme{
		terminalBackground: bg,
		fallbackBackground: twin.NewColorHex(0x000000),
		foreground:         twin.NewColorHex(0xdddddd),

		loadBarMin:    twin.NewColorHex(0x000000),
		loadBarMaxRam: twin.NewColorHex(0x2020ff),
		loadBarMaxCpu: twin.NewColorHex(0x801020),
		loadBarMaxIO:  twin.NewColorHex(0xd0d020),

		border:      twin.NewColorHex(0x7070a0),
		borderTitle: twin.NewColorHex(0xffc0c0),

		loadLow:    twin.NewColorHex(0x00ff00),
		loadMedium: twin.NewColorHex(0xffff00),
		loadHigh:   twin.NewColorHex(0xff0000),
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

func (t Theme) LoadBarMin() twin.Color {
	return t.loadBarMin
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

func (t Theme) LoadLow() twin.Color {
	return t.loadLow
}

func (t Theme) LoadMedium() twin.Color {
	return t.loadMedium
}

func (t Theme) LoadHigh() twin.Color {
	return t.loadHigh
}
