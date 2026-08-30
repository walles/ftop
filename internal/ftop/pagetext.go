package ftop

import (
	"io"
	"strings"

	"github.com/rivo/uniseg"
	"github.com/walles/moor/v2/twin"
)

type pageText struct {
	// Where the page goes, one line at a time. The pager renders whatever has
	// arrived so far, so the early sections are readable while the slow ones
	// are still being composed.
	out io.Writer

	titleStyle  twin.Style
	borderStyle twin.Style
}

// Concatenates the parts and hands them over in a single write.
//
// Styled lines come in a lot of pieces, and the pager is on the other end of a
// pipe, so putting the line together first saves it a handover per piece.
//
// Write errors are ignored: the only reader is the pager, and once that one is
// gone there is nobody left to show the rest of the page to.
func (pt *pageText) write(parts ...string) {
	var line strings.Builder
	for _, part := range parts {
		line.WriteString(part)
	}

	_, _ = io.WriteString(pt.out, line.String())
}

// Appends a line feed at the end of the provided string
func (pt *pageText) writeLine(line string) {
	pt.write(line, "\n")
}

func (pt *pageText) writeTitle(title string) {
	const width = 80
	trailerLength := max(2, width-2-uniseg.StringWidth(title))
	trailer := strings.Repeat("─", trailerLength)

	// "24 bit" is fine here, if the terminal doesn't support it, the pager will
	// just down sample it as needed.
	pt.write(
		pt.borderStyle.RenderUpdateFrom(twin.StyleDefault, twin.ColorCount24bit),
		"──",
		pt.titleStyle.RenderUpdateFrom(pt.borderStyle, twin.ColorCount24bit),
		title,
		pt.borderStyle.RenderUpdateFrom(pt.titleStyle, twin.ColorCount24bit),
		trailer,
		twin.StyleDefault.RenderUpdateFrom(pt.borderStyle, twin.ColorCount24bit),
		"\n",
		"\n",
	)
}

// The styled text fancy, with spaces appended until it fills width columns, or
// fancy alone if it fills them already.
//
// plain is fancy without the styling, and is what the padding is measured over:
// the escape sequences fancy carries take up no columns, so counting them would
// leave a styled string short of where an unstyled one of the same width lands.
// Pass the same string twice for text with no styling to it.
func rightPadded(plain string, fancy string, width int) string {
	padding := max(0, width-uniseg.StringWidth(plain))

	return fancy + strings.Repeat(" ", padding)
}

// Wraps s in the theme's highlight color, ending in whatever style the line
// started out in.
func (u *Ui) highlight(s string) string {
	return colorize(s, u.theme.HighlightedForeground())
}

// Wraps s in the theme's duplicate-peer color, ending in whatever style the
// line started out in.
func (u *Ui) highlightDuplicate(s string) string {
	return colorize(s, u.theme.DuplicatePeer())
}

func colorize(s string, color twin.Color) string {
	colored := twin.StyleDefault.WithForeground(color)
	notColored := twin.StyleDefault

	// "24 bit" is fine here, if the terminal doesn't support it, the pager will
	// just down sample it as needed.
	prefix := colored.RenderUpdateFrom(notColored, twin.ColorCount24bit)
	suffix := notColored.RenderUpdateFrom(colored, twin.ColorCount24bit)

	return prefix + s + suffix
}
