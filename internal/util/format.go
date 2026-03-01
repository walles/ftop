package util

import (
	"fmt"
	"time"
)

func FormatDuration(duration time.Duration) string {
	totalSeconds := int(duration.Seconds())
	if totalSeconds < 10 {
		// FIXME: Don't show decimals here unless we know we're getting
		// sub-second precision from ps. macOS provides decimals, unsure about
		// Linux.
		return fmt.Sprintf("%.2fs", duration.Seconds())
	}

	if totalSeconds < 60 {
		// FIXME: Don't show decimals here unless we know we're getting
		// sub-second precision from ps. macOS provides decimals, unsure about
		// Linux.
		return fmt.Sprintf("%.1fs", duration.Seconds())
	}

	if totalSeconds < 3600 {
		minutes := totalSeconds / 60
		seconds := totalSeconds % 60
		return fmt.Sprintf("%dm%02ds", minutes, seconds)
	}

	if totalSeconds < 86400 {
		hours := totalSeconds / 3600
		minutes := (totalSeconds % 3600) / 60
		return fmt.Sprintf("%dh%02dm", hours, minutes)
	}

	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	return fmt.Sprintf("%dd%02dh", days, hours)
}

// Turns memory numbers into strings like "1.1G", "256.0M" or "512B". Note no
// trailing B, to conserve space in the UI.
func FormatMemory(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	var number float64
	var unit string
	switch {
	case bytes >= TB:
		number = float64(bytes) / float64(TB)
		unit = "T"
	case bytes >= GB:
		number = float64(bytes) / float64(GB)
		unit = "G"
	case bytes >= MB:
		number = float64(bytes) / float64(MB)
		unit = "M"
	case bytes >= KB:
		number = float64(bytes) / float64(KB)
		unit = "k"
	default:
		number = float64(bytes)
		unit = "B"
	}

	if number < 10 {
		// Add decimals to smaller numbers
		return fmt.Sprintf("%.1f%s", number, unit)
	}

	// No decimals for larger numbers
	return fmt.Sprintf("%.0f%s", number, unit)
}
