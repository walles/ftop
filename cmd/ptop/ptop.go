package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/io"
	"github.com/walles/ptop/internal/log"
	"github.com/walles/ptop/internal/processes"
	"github.com/walles/ptop/internal/ptop"
	"github.com/walles/ptop/internal/themes"
)

type processListUpdated struct{}

func main() {
	os.Exit(internalMain())
}

// Never call os.Exit() from inside of this function because that will cause us
// not to shut down the screen properly.
//
// Returns the program's exit code.
//
// Example:
//
//	func main() {
//	    os.Exit(internalMain())
//	}
func internalMain() int {
	argsParser, err := kong.New(&CLI)
	if err != nil {
		panic(err)
	}

	if env, ok := os.LookupEnv("PTOP"); ok {
		log.Infof("PTOP=\"%s\"", env)
	} else {
		log.Infof("PTOP environment variable not set")
	}

	envArgs := strings.Fields(os.Getenv("PTOP"))
	_, err = argsParser.Parse(append(envArgs, os.Args[1:]...))
	if err != nil {
		if len(os.Getenv("PTOP")) > 0 {
			fmt.Fprintln(os.Stderr, "PTOP environment variable value: \""+os.Getenv("PTOP")+"\"")
			fmt.Fprintln(os.Stderr)
		}

		argsParser.FatalIfErrorf(err)
	}

	screen, err := twin.NewScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating screen:", err)
		return 1
	}

	defer onExit(screen, CLI.Debug)

	defer func() {
		log.PanicHandler("main", recover(), debug.Stack())
	}()

	theme := themes.NewTheme(CLI.Theme.String(), screen.TerminalBackground())

	procsTracker := processes.NewTracker()
	ioTracker := io.NewTracker()

	events := make(chan twin.Event)
	go func() {
		defer func() {
			log.PanicHandler("main/screen events poller", recover(), debug.Stack())
		}()
		for event := range screen.Events() {
			events <- event
		}
	}()
	go func() {
		defer func() {
			log.PanicHandler("main/processes tracker poller", recover(), debug.Stack())
		}()
		for range procsTracker.OnUpdate {
			events <- processListUpdated{}
		}
	}()

	ui := ptop.NewUi(screen, theme)

	for {
		event := <-events

		if _, ok := event.(twin.EventResize); ok {
			allProcesses := procsTracker.Processes()
			ui.Render(allProcesses, ioTracker.Stats(), procsTracker.Launches())
		}

		if _, ok := event.(processListUpdated); ok {
			allProcesses := procsTracker.Processes()
			ui.Render(allProcesses, ioTracker.Stats(), procsTracker.Launches())
		}

		if event, ok := event.(twin.EventRune); ok {
			if event.Rune() == 'q' {
				break
			}
		}

		if event, ok := event.(twin.EventKeyCode); ok {
			if event.KeyCode() == twin.KeyEscape {
				break
			}
		}
	}

	return 0
}

func onExit(screen twin.Screen, forcePrintLogs bool) {
	screen.Close()

	if len(log.String(true)) == 0 {
		return
	}

	mustPrintLogs := log.HasErrors() || forcePrintLogs
	if !mustPrintLogs {
		return
	}

	fmt.Fprint(os.Stderr, log.String(true))

	if log.HasErrors() {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
