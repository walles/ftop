package ftop

import (
	"strings"
	"testing"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

// The chain of launches that led to the picked process, oldest first and one
// indentation step per generation, with an arrow pointing out the process the
// page is about. Its children come below it, one step further in.
//
// Every line names its owner in a column of its own, and that column lines up
// across the whole tree however long the process names above and below it are.
func TestLaunchHierarchyForPaging(t *testing.T) {
	initProcess := &processes.Process{Pid: 1, Cmdline: "init", Username: "root"}
	bash := &processes.Process{Pid: 1234, Cmdline: "bash", Username: "alice"}
	picked := &processes.Process{Pid: 42, Cmdline: "picked", Username: "alice"}
	vim := &processes.Process{Pid: 4321, Cmdline: "vim", Username: "alice"}

	initProcess.AddChild(bash)
	bash.AddChild(picked)
	picked.AddChild(vim)

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.launchHierarchyForPaging(picked, &pt)

	expected := "" +
		"init(1)          root\n" +
		"  bash(1234)     alice\n" +
		"──▶ picked(42)   alice\n" +
		"      vim(4321)  alice\n"
	assert.Equal(t, sectionBody(page.String()), expected)

	assert.Equal(t, stringsContains(page.String(), "──Launch Hierarchy──"), true)
}

// Column widths are measured in terminal columns rather than in characters. A
// CJK name takes two columns per character, so a launcher wearing one pushes the
// owner column out for every line of the tree, not only for its own.
func TestLaunchHierarchyForPagingWideCharactersInTheChain(t *testing.T) {
	viewer := &processes.Process{Pid: 1234, Cmdline: "写真整理", Username: "alice"}
	picked := &processes.Process{Pid: 42, Cmdline: "picked", Username: "alice"}
	vim := &processes.Process{Pid: 4321, Cmdline: "vim", Username: "alice"}

	viewer.AddChild(picked)
	picked.AddChild(vim)

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.launchHierarchyForPaging(picked, &pt)

	expected := "" +
		"写真整理(1234)  alice\n" +
		"▶ picked(42)    alice\n" +
		"    vim(4321)   alice\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// The children of the picked process are measured the same way as the chain
// that launched it, so a CJK name among them decides the owner column too.
func TestLaunchHierarchyForPagingWideCharactersAmongTheChildren(t *testing.T) {
	bash := &processes.Process{Pid: 1234, Cmdline: "bash", Username: "alice"}
	picked := &processes.Process{Pid: 42, Cmdline: "picked", Username: "alice"}
	viewer := &processes.Process{Pid: 999, Cmdline: "写真整理", Username: "alice"}

	bash.AddChild(picked)
	picked.AddChild(viewer)

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.launchHierarchyForPaging(picked, &pt)

	expected := "" +
		"bash(1234)         alice\n" +
		"▶ picked(42)       alice\n" +
		"    写真整理(999)  alice\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}
