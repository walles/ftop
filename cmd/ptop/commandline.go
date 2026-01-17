package main

import "fmt"

var CLI struct {
	Theme ThemeName `help:"auto, dark or light" default:"auto"`
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
