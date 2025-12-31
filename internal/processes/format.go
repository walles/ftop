package processes

import (
	"fmt"
	"time"
)

func formatDuration(duration time.Duration) string {
	totalSeconds := int(duration.Seconds())
	if totalSeconds < 60 {
		// FIXME: Don't show decimals here unless we know we're getting
		// sub-second precision from ps. macOS provides decimals, unsure about
		// Linux.
		return fmt.Sprintf("%.2fs", duration.Seconds())
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
func formatMemory(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= 2*TB:
		return fmt.Sprintf("%.1fT", float64(bytes)/float64(TB))
	case bytes >= 2*GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/float64(GB))
	case bytes >= 2*MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/float64(MB))
	case bytes >= 2*KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
