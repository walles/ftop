package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"

	detectrace "github.com/jbenet/go-detect-race"

	"github.com/alecthomas/kong"
	"github.com/walles/ftop/internal/ftop"
	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

// Build a binary using build.sh to get a proper version string here. This
// pre-filled string will be used otherwise.
var versionString = "<build with ./build.sh to get a version number here>"

func main() {
	argsParser, err := kong.New(
		&CLI,
		kong.Description("Shows a top list of running processes.\n\nhttps://github.com/walles/ftop"),
	)
	if err != nil {
		panic(err)
	}

	if env, ok := os.LookupEnv("FTOP"); ok {
		log.Infof("FTOP=\"%s\"", env)
	} else {
		log.Infof("FTOP environment variable not set")
	}

	envArgs := strings.Fields(os.Getenv("FTOP"))
	_, err = argsParser.Parse(append(envArgs, os.Args[1:]...))
	if err != nil {
		if len(os.Getenv("FTOP")) > 0 {
			fmt.Fprintln(os.Stderr, "FTOP environment variable value: \""+os.Getenv("FTOP")+"\"")
			fmt.Fprintln(os.Stderr)
		}

		argsParser.FatalIfErrorf(err)
	}

	if CLI.Version {
		fmt.Println(versionString)
		os.Exit(0)
	}

	if CLI.Profile {
		if detectrace.WithRace() {
			fmt.Fprintln(os.Stderr, "ERROR: Profiling is not supported when built with --race")
			os.Exit(1)
		}

		os.Exit(profilingMainLoop(CLI.Panic))
	} else {
		os.Exit(mainLoop(CLI.Panic))
	}
}

// Generate files "profile-cpu.out" and "profile-heap.out" before exit.
//
//	go run ./cmd/ftop/ftop.go --profile
//
// Analyze the files like this:
//
//	go tool pprof -relative_percentages -web profile-cpu.out
//	go tool pprof -relative_percentages -web profile-heap.out
func profilingMainLoop(pleasePanic bool) int {
	//
	// Start CPU profiling
	//
	cpuFile, err := os.Create("profile-cpu.out")
	if err != nil {
		panic(err)
	}
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		panic(err)
	}

	//
	// Do the actual work
	//
	result := mainLoop(pleasePanic)

	// Write out CPU profile
	pprof.StopCPUProfile()
	err = cpuFile.Close()
	if err != nil {
		panic(err)
	}

	//
	// Write out heap profile
	//
	heapFile, err := os.Create("profile-heap.out")
	if err != nil {
		panic(err)
	}
	defer func() {
		err := heapFile.Close()
		if err != nil {
			panic(err)
		}
	}()

	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(heapFile); err != nil {
		panic(err)
	}

	fmt.Println("Profiles written, analyze them like this:")
	fmt.Println()
	fmt.Println("go tool pprof -relative_percentages -web profile-cpu.out")
	fmt.Println("go tool pprof -relative_percentages -web profile-heap.out")

	return result
}

func mainLoop(pleasePanic bool) int {
	screen, err := twin.NewScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating screen:", err)
		return 1
	}

	defer onExit(screen, CLI.Debug)

	defer func() {
		log.PanicHandler("main", recover(), debug.Stack())
	}()

	if pleasePanic {
		panic("panic requested by --panic command line option")
	}

	theme := themes.NewTheme(CLI.Theme.String(), screen.TerminalBackground())

	ui := ftop.NewUi(screen, theme)
	ui.MainLoop()

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

	if log.HasErrors() {
		fmt.Fprintln(os.Stderr, "vvv \033[1mftop crashed\033[0m vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Please post this text at \033[1mhttps://github.com/walles/ftop/issues\033[0m.")
		fmt.Fprintln(os.Stderr)
	}

	fmt.Fprintln(os.Stderr, "Version  :", versionString)
	fmt.Fprintln(os.Stderr, "GOOS     :", runtime.GOOS)
	fmt.Fprintln(os.Stderr, "GOARCH   :", runtime.GOARCH)
	fmt.Fprintln(os.Stderr, "GOVERSION:", runtime.Version())
	fmt.Fprintln(os.Stderr, "Compiler :", runtime.Compiler)
	fmt.Fprintln(os.Stderr, "NumCPU   :", runtime.NumCPU())

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, log.String(true))

	if log.HasErrors() {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Please post this text at \033[1mhttps://github.com/walles/ftop/issues\033[0m.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "^^^ \033[1mftop crashed\033[0m ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^")
	}

	if log.HasErrors() {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
