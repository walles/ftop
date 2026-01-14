package themes

import "github.com/walles/moor/v2/twin"

// Hard coded dark theme
type Theme struct {
	bg *twin.Color
	loadBarMin      twin.Color
	loadBarMaxRam   twin.Color
	loadBarMaxCpu   twin.Color
	loadBarMaxIO    twin.Color
	border          twin.Color
	borderTitle     twin.Color
	top             twin.Color
	bottom          twin.Color
	backgroundColor twin.Color
	loadLow         twin.Color
	loadMedium      twin.Color
	loadHigh        twin.Color
}

// NOTE: Use some online OKLCH color picker for experimenting with colors

func NewDarkTheme(bg *twin.Color) Theme {
	var background twin.Color
	if bg != nil {
		background = *bg
	} else {
		background = twin.NewColorHex(0x000000)
	}

	top := twin.NewColorHex(0xdddddd)
	bottom := top.Mix(background, 0.5)

	return Theme{
		bg:               bg,
		loadBarMin:       twin.NewColorHex(0x000000),
		loadBarMaxRam:    twin.NewColorHex(0x2020ff),
		loadBarMaxCpu:    twin.NewColorHex(0x801020),
		loadBarMaxIO:     twin.NewColorHex(0xd0d020),
		border:           twin.NewColorHex(0x7070a0),
		borderTitle:      twin.NewColorHex(0xffc0c0),
		top:              top,
		bottom:           bottom,
		backgroundColor:  background,
		loadLow:          twin.NewColorHex(0x00ff00),
		loadMedium:       twin.NewColorHex(0xffff00),
		loadHigh:         twin.NewColorHex(0xff0000),
	}
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

func (t Theme) Top() twin.Color {
	return t.top
}

func (t Theme) Bottom() twin.Color {
	return t.bottom
}

func (t Theme) Background() twin.Color {
	if t.bg != nil {
		return *t.bg
	}
	return t.backgroundColor
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
