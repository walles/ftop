package ftop

import (
	"testing"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/moor/v2/twin"
)

func TestTruncateToLength(t *testing.T) {
	u := Ui{}

	styledRunes := []twin.StyledRune{
		{Rune: 'a', Style: twin.StyleDefault},
		{Rune: 'b', Style: twin.StyleDefault},
		{Rune: 'c', Style: twin.StyleDefault},
	}

	assert.Equal(t, len(u.truncateToLength(styledRunes, 4)), 3)
	assert.Equal(t, len(u.truncateToLength(styledRunes, 3)), 3)
	assert.Equal(t, len(u.truncateToLength(styledRunes, 2)), 2)
	assert.Equal(t, len(u.truncateToLength(styledRunes, 1)), 1)
}
