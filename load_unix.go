//go:build !windows
// +build !windows

package main

import (
	"fmt"

	"github.com/shirou/gopsutil/v3/load"
)

// getLoadInfo returns load average for Unix systems
func getLoadInfo() string {
	avg, err := load.Avg()
	if err != nil {
		return "N/A"
	}
	return fmt.Sprintf("%.2f %.2f %.2f", avg.Load1, avg.Load5, avg.Load15)
}
