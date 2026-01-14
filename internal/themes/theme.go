package themes

import "github.com/walles/moor/v2/twin"

// Hard coded dark theme
type Theme struct {
	bg *twin.Color
}

// NOTE: Use some online OKLCH color picker for experimenting with colors

func NewDarkTheme(bg *twin.Color) Theme {
	return Theme{bg: bg}
}

func (t Theme) LoadBarMin() twin.Color {
	return twin.NewColorHex(0x000000)
}

func (t Theme) LoadBarMaxRam() twin.Color {
	return twin.NewColorHex(0x2020ff)
}

func (t Theme) LoadBarMaxCpu() twin.Color {
	return twin.NewColorHex(0x801020)
}

func (t Theme) LoadBarMaxIO() twin.Color {
	return twin.NewColorHex(0xd0d020)
}

func (t Theme) Border() twin.Color {
	return twin.NewColorHex(0x7070a0)
}

func (t Theme) BorderTitle() twin.Color {
	return twin.NewColorHex(0xffc0c0)
}

func (t Theme) Top() twin.Color {
	return twin.NewColorHex(0xdddddd)
}

func (t Theme) Bottom() twin.Color {
	return t.Top().Mix(t.Background(), 0.5)
}

func (t Theme) Background() twin.Color {
	if t.bg != nil {
		return *t.bg
	}
	return twin.NewColorHex(0x000000)
}

func (t Theme) LoadLow() twin.Color {
	return twin.NewColorHex(0x00ff00)
}

func (t Theme) LoadMedium() twin.Color {
	return twin.NewColorHex(0xffff00)
}

func (t Theme) LoadHigh() twin.Color {
	return twin.NewColorHex(0xff0000)
}
