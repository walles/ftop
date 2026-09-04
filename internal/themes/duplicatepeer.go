package themes

import (
	"math"

	"github.com/walles/twin"
)

// Picks a color for marking a peer process that recurs within one connections
// listing, derived from a theme's own foreground and self-highlight colors.
//
// Value and saturation both match the self color's, so the result is a straight
// hue rotation of it. Hue is chosen to sit as far as possible from whichever of
// the two colors have a hue at all — an achromatic color (gray, black, white)
// has none, and contributes nothing to that choice.
func pickDuplicatePeerColor(foreground twin.Color, self twin.Color) twin.Color {
	foregroundHue, foregroundSaturation, _ := rgbToHsv(foreground)
	selfHue, selfSaturation, selfValue := rgbToHsv(self)

	var hues []float64
	if foregroundSaturation > 0 {
		hues = append(hues, foregroundHue)
	}
	if selfSaturation > 0 {
		hues = append(hues, selfHue)
	}

	hue := hueFarthestFrom(hues)
	value := selfValue

	return hsvToRgb(hue, selfSaturation, value)
}

// The hue, in degrees 0-360, as far as possible from every hue in from.
//
// With no hues to avoid there is nothing to optimize against, so this falls
// back to an arbitrary fixed hue. With one, the answer is its exact opposite.
// With two, the answer is the midpoint of the larger of the two arcs between
// them, that being the point on the circle that is farthest from whichever of
// the two it is closest to.
//
// Not implemented for more than two hues, that would panic.
func hueFarthestFrom(from []float64) float64 {
	switch len(from) {
	case 0:
		return 0

	case 1:
		return math.Mod(from[0]+180, 360)

	case 2:
		lo, hi := from[0], from[1]
		if lo > hi {
			lo, hi = hi, lo
		}

		arc1 := hi - lo
		arc2 := 360 - arc1
		if arc2 > arc1 {
			return math.Mod(hi+arc2/2, 360)
		}

		return math.Mod(lo+arc1/2, 360)

	default:
		panic("Not implemented for more than two colors")
	}
}

// Converts a color into hue (degrees, 0-360), saturation and value (both
// 0-1).
//
// Hue is 0 for a fully desaturated color, same as for red: no hue is what zero
// saturation means, red is only ever a coincidence of the formula.
func rgbToHsv(rgbColor twin.Color) (hue float64, saturation float64, value float64) {
	r16, g16, b16, _ := rgbColor.RGBA()
	r := float64(uint8(r16>>8)) / 255
	g := float64(uint8(g16>>8)) / 255
	b := float64(uint8(b16>>8)) / 255

	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	delta := max - min

	value = max
	if delta == 0 {
		return 0, 0, value
	}

	saturation = delta / max

	switch max {
	case r:
		hue = 60 * math.Mod((g-b)/delta, 6)
	case g:
		hue = 60 * ((b-r)/delta + 2)
	default:
		hue = 60 * ((r-g)/delta + 4)
	}

	if hue < 0 {
		hue += 360
	}

	return hue, saturation, value
}

// Converts hue (degrees, 0-360), saturation and value (both 0-1) into a
// color.
func hsvToRgb(hue float64, saturation float64, value float64) twin.Color {
	c := value * saturation
	x := c * (1 - math.Abs(math.Mod(hue/60, 2)-1))
	m := value - c

	var r, g, b float64
	switch {
	case hue < 60:
		r, g, b = c, x, 0
	case hue < 120:
		r, g, b = x, c, 0
	case hue < 180:
		r, g, b = 0, c, x
	case hue < 240:
		r, g, b = 0, x, c
	case hue < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	red := uint8(math.Round((r + m) * 255))
	green := uint8(math.Round((g + m) * 255))
	blue := uint8(math.Round((b + m) * 255))

	return twin.NewColor24Bit(red, green, blue)
}
