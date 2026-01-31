package main

import "fmt"

var CLI struct {
	Version bool      `help:"show version information"`
	Theme   ThemeName `help:"auto, dark or light" default:"auto"`
	Debug   bool      `help:"print debug logs after exit"`

	// Hidden options for development use
	Profile bool `help:"generate profile-*.out files before exiting" hidden:"true"`
	Panic   bool `help:"panic on purpose for testing crash handling" hidden:"true"`
}

type ThemeName string

func (t ThemeName) Validate() error {
	switch t {
	case "auto", "dark", "light":
		return nil
	default:
		return fmt.Errorf(`must be "auto", "dark" or "light": <%s>`, t)
	}
}

func (t ThemeName) String() string {
	return string(t)
}
