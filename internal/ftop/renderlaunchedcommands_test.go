package ftop

import (
	"reflect"
	"strings"
	"testing"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/ftop/internal/ui"
	"github.com/walles/moor/v2/twin"
)

func assertRenderLaunchedCommands(t *testing.T, root *processes.LaunchNode, expected []string) {
	t.Helper()

	width, height := 20, 10
	screen := twin.NewFakeScreen(width, height)
	screen.Clear()

	theme := themes.NewTheme("auto", nil)
	renderLaunchedCommands(screen, theme, root, 0, height-1)

	screenRows := []string{}
	for y := 1; y < height-1; y++ {
		row := ""
		for x := 1; x < width-1; x++ {
			row += string(screen.GetCell(x, y).Rune)
		}

		row = strings.TrimRight(row, " ")
		if row == "" {
			// No more content lines
			break
		}
		screenRows = append(screenRows, row)
	}

	if reflect.DeepEqual(screenRows, expected) {
		// We got what we wanted, now check the height calculation too
		assert.Equal(t, getLaunchedCommandsHeight(root), len(expected))
		return
	}

	// Failed, print diagnostics: print each slice element on its own line.
	// First "Expected" with rows below each other, then "Actual" the same way.
	join := func(s []string) string {
		if len(s) == 0 {
			return " (empty)"
		}

		out := ""
		for _, line := range s {
			out += "\n" + line
		}

		return out
	}

	t.Fatalf("\nExpected:%s\n\nActual:%s",
		join(expected), join(screenRows))
}

func TestRenderLaunchedCommands(t *testing.T) {
	nd := processes.LaunchNode{Command: "d"}
	nc := processes.LaunchNode{Command: "c", Children: []*processes.LaunchNode{&nd}}
	nb := processes.LaunchNode{Command: "b", Children: []*processes.LaunchNode{&nc}}
	ne := processes.LaunchNode{Command: "e"}
	ng := processes.LaunchNode{Command: "g"}
	ni := processes.LaunchNode{Command: "i"}
	nh := processes.LaunchNode{Command: "h", Children: []*processes.LaunchNode{&ni}}
	nf := processes.LaunchNode{Command: "f", Children: []*processes.LaunchNode{&ng, &nh}}
	nj := processes.LaunchNode{Command: "j"}
	na := processes.LaunchNode{
		Command:  "a",
		Children: []*processes.LaunchNode{&nb, &ne, &nf, &nj},
	}

	root := &na

	assertRenderLaunchedCommands(t, root, []string{
		"a┬─b──c──d",
		" ├─e",
		" ├─f┬─g",
		" │  └─h──i",
		" └─j",
	})
}

func TestRenderLaunchedCommands_sameOrderDifferentForks(t *testing.T) {
	nxd := processes.LaunchNode{Command: "d"}
	nxc := processes.LaunchNode{Command: "c", Children: []*processes.LaunchNode{&nxd}}

	nyd := processes.LaunchNode{Command: "d"}
	nyc := processes.LaunchNode{Command: "c", Children: []*processes.LaunchNode{&nyd}}

	nx := processes.LaunchNode{Command: "x", Children: []*processes.LaunchNode{&nxc}}
	ny := processes.LaunchNode{Command: "y", Children: []*processes.LaunchNode{&nyc}}
	na := processes.LaunchNode{
		Command:  "a",
		Children: []*processes.LaunchNode{&nx, &ny},
	}

	root := &na

	assertRenderLaunchedCommands(t, root, []string{
		"a┬─x──c──d",
		" └─y──c──d",
	})
}

func TestRenderLaunchedCommands_forkAfterMultipleParents(t *testing.T) {
	nd := processes.LaunchNode{Command: "d"}
	nc := processes.LaunchNode{Command: "c", Children: []*processes.LaunchNode{&nd}}
	ne := processes.LaunchNode{Command: "e"}
	nb := processes.LaunchNode{Command: "b", Children: []*processes.LaunchNode{&nc, &ne}}
	na := processes.LaunchNode{Command: "a", Children: []*processes.LaunchNode{&nb}}

	root := &na

	assertRenderLaunchedCommands(t, root, []string{
		"a──b┬─c──d",
		"    └─e",
	})
}

func TestRenderLaunchedCommands_somethingBetweenUs(t *testing.T) {
	nd := processes.LaunchNode{Command: "d", LaunchCount: 3}
	nc := processes.LaunchNode{Command: "c", Children: []*processes.LaunchNode{&nd}}
	ne := processes.LaunchNode{Command: "e", LaunchCount: 2}
	ng := processes.LaunchNode{Command: "g", LaunchCount: 1}
	nf := processes.LaunchNode{Command: "f", Children: []*processes.LaunchNode{&ng}}
	nb := processes.LaunchNode{Command: "b", Children: []*processes.LaunchNode{&nc, &ne, &nf}}
	na := processes.LaunchNode{Command: "a", Children: []*processes.LaunchNode{&nb}}

	root := &na

	assertRenderLaunchedCommands(t, root, []string{
		"a──b┬─c──d(3)",
		"    ├─e(2)",
		"    └─f──g(1)",
	})
}

func TestRenderLaunchedCommand_dontClipTooEarly(t *testing.T) {
	width, height := 20, 10
	screen := twin.NewFakeScreen(width, height)
	screen.Clear()

	nc := processes.LaunchNode{Command: "c"}
	nd := processes.LaunchNode{Command: "d"}
	nb := processes.LaunchNode{Command: "b", Children: []*processes.LaunchNode{&nc, &nd}}
	na := processes.LaunchNode{Command: "a", Children: []*processes.LaunchNode{&nb}}

	root := &na

	// Limit last y to 1. We should still get the whole graph, since it goes from y=0 to y=1.
	topBottomRamp := ui.NewColorRamp(0, 9, twin.NewColorHex(0xffffff), twin.NewColorHex(0x808080))
	renderLaunchedCommand(screen, "", root, 0, 0, width-1, 1, topBottomRamp)

	expected := []string{
		"a──b┬─c",
		"    └─d",
	}

	screenRows := []string{}
	for y := range height {
		row := ""
		for x := range width {
			row += string(screen.GetCell(x, y).Rune)
		}

		row = strings.TrimRight(row, " ")
		if row == "" {
			// No more content lines
			break
		}
		screenRows = append(screenRows, row)
	}

	if reflect.DeepEqual(screenRows, expected) {
		// We got what we wanted
		return
	}

	// Failed, print diagnostics: print each slice element on its own line.
	// First "Expected" with rows below each other, then "Actual" the same way.
	join := func(s []string) string {
		if len(s) == 0 {
			return " (empty)"
		}

		out := ""
		for _, line := range s {
			out += "\n" + line
		}

		return out
	}

	t.Fatalf("\nExpected:%s\n\nActual:%s",
		join(expected), join(screenRows))

}

func TestRenderLaunchedCommand_dontClipTooLate(t *testing.T) {
	width, height := 20, 10
	screen := twin.NewFakeScreen(width, height)
	screen.Clear()

	nc := processes.LaunchNode{Command: "c"}
	nd := processes.LaunchNode{Command: "d"}
	nb := processes.LaunchNode{Command: "b", Children: []*processes.LaunchNode{&nc, &nd}}
	na := processes.LaunchNode{Command: "a", Children: []*processes.LaunchNode{&nb}}

	root := &na

	// Limit last y to 0. We should get exactly one line of graph, since the second should be clipped out.
	topBottomRamp := ui.NewColorRamp(0, 9, twin.NewColorHex(0xffffff), twin.NewColorHex(0x808080))
	renderLaunchedCommand(screen, "", root, 0, 0, width-1, 0, topBottomRamp)

	expected := []string{
		"a──b┬─c",
	}

	screenRows := []string{}
	for y := range height {
		row := ""
		for x := range width {
			row += string(screen.GetCell(x, y).Rune)
		}

		row = strings.TrimRight(row, " ")
		if row == "" {
			// No more content lines
			break
		}
		screenRows = append(screenRows, row)
	}

	if reflect.DeepEqual(screenRows, expected) {
		// We got what we wanted
		return
	}

	// Failed, print diagnostics: print each slice element on its own line.
	// First "Expected" with rows below each other, then "Actual" the same way.
	join := func(s []string) string {
		if len(s) == 0 {
			return " (empty)"
		}

		out := ""
		for _, line := range s {
			out += "\n" + line
		}

		return out
	}

	t.Fatalf("\nExpected:%s\n\nActual:%s",
		join(expected), join(screenRows))

}

func TestRenderLaunchedCommands_sortByLaunchCount(t *testing.T) {
	nPlugin := processes.LaunchNode{Command: "Firefox/plugin-container", LaunchCount: 1}
	nFirefox := processes.LaunchNode{Command: "Firefox/firefox", Children: []*processes.LaunchNode{&nPlugin}}

	nMdworker := processes.LaunchNode{Command: "CoreServices/mdworker_shared", LaunchCount: 9}

	nLaunchd := processes.LaunchNode{Command: "launchd", Children: []*processes.LaunchNode{&nMdworker, &nFirefox}}
	root := &nLaunchd

	assertRenderLaunchedCommands(t, root, []string{
		"launchd┬─CoreServ",
		"       └─Firefox/",
	})
}
