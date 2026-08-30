package themes

import "testing"

// The duplicate-peer color is derived from each theme's foreground and
// highlight colors, see pickDuplicatePeerColor(). These are golden values: if a
// deliberate change to a theme's colors changes this on purpose, update the
// expectation to match.
func TestDuplicatePeerColor(t *testing.T) {
	testCases := []struct {
		theme    string
		expected string
	}{
		{theme: "dark", expected: "#ebbdea"},
		{theme: "light", expected: "#900090"},
	}

	for _, testCase := range testCases {
		theme := NewTheme(testCase.theme, nil)
		actual := theme.DuplicatePeer().String()
		if actual != testCase.expected {
			t.Errorf("%s theme: got %s, want %s", testCase.theme, actual, testCase.expected)
		}
	}
}
