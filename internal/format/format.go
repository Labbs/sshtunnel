// Package format provides utility functions for formatting values for display.
package format

import (
	"fmt"
	"time"
)

// Bytes formats a byte count to a human-readable string (e.g., "1.5 KB", "3.2 MB").
func Bytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	// Bounds check to prevent panic on extremely large values
	units := "KMGTPE"
	if exp >= len(units) {
		exp = len(units) - 1
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), units[exp])
}

// Duration formats a duration for display in a compact form.
func Duration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours < 24 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	days := hours / 24
	hours = hours % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}
