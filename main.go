// Konrul - Terminal Based System Monitor
// Named after the mythological Turkish phoenix-like creature
// A lightweight htop-like system monitor written in Go
// Cross-platform support: Linux, macOS, Windows, FreeBSD

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

// Version information (set by ldflags during build)
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// Process represents a system process
type Process struct {
	PID     int32
	Name    string
	State   string
	CPU     float64
	Memory  float32
	User    string
	Command string
}

func main() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	// CPU Gauge
	cpuGauge := widgets.NewGauge()
	cpuGauge.Title = " CPU "
	cpuGauge.BarColor = ui.ColorGreen
	cpuGauge.BorderStyle.Fg = ui.ColorCyan

	// Memory Gauge
	memGauge := widgets.NewGauge()
	memGauge.Title = " Memory "
	memGauge.BarColor = ui.ColorYellow
	memGauge.BorderStyle.Fg = ui.ColorCyan

	// Swap Gauge
	swapGauge := widgets.NewGauge()
	swapGauge.Title = " Swap "
	swapGauge.BarColor = ui.ColorMagenta
	swapGauge.BorderStyle.Fg = ui.ColorCyan

	// Process Table
	processTable := widgets.NewTable()
	processTable.Title = " Processes (Up/Down: scroll, q: quit, K: kill) "
	processTable.TextStyle = ui.NewStyle(ui.ColorWhite)
	processTable.RowSeparator = false
	processTable.BorderStyle.Fg = ui.ColorCyan
	processTable.TextAlignment = ui.AlignLeft
	// Column widths: PID(7), USER(9), CPU%(6), MEM%(6), STATE(6), COMMAND(remaining)
	processTable.ColumnWidths = []int{7, 9, 6, 6, 6, -1}

	// System Info
	sysInfo := widgets.NewParagraph()
	sysInfo.Title = " System "
	sysInfo.BorderStyle.Fg = ui.ColorCyan

	// CPU Sparkline
	cpuSparkline := widgets.NewSparkline()
	cpuSparkline.LineColor = ui.ColorGreen

	cpuSparklineGroup := widgets.NewSparklineGroup(cpuSparkline)
	cpuSparklineGroup.Title = " CPU History "
	cpuSparklineGroup.BorderStyle.Fg = ui.ColorCyan

	cpuHistory := make([]float64, 50)

	// Layout
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	grid.Set(
		ui.NewRow(0.12,
			ui.NewCol(0.33, cpuGauge),
			ui.NewCol(0.33, memGauge),
			ui.NewCol(0.34, swapGauge),
		),
		ui.NewRow(0.15,
			ui.NewCol(0.6, cpuSparklineGroup),
			ui.NewCol(0.4, sysInfo),
		),
		ui.NewRow(0.73, processTable),
	)

	selectedRow := 1
	scrollOffset := 0
	maxVisibleRows := 15

	// Cache for processes
	var cachedProcesses []Process
	var lastProcessUpdate time.Time

	render := func() {
		// Update CPU
		cpuPercent := getCPUPercent()
		cpuGauge.Percent = int(cpuPercent)
		cpuGauge.Label = fmt.Sprintf("%.1f%%", cpuPercent)

		// Update CPU history
		cpuHistory = append(cpuHistory[1:], cpuPercent)
		sparklineData := make([]float64, len(cpuHistory))
		copy(sparklineData, cpuHistory)
		cpuSparkline.Data = sparklineData

		// Update Memory
		memTotal, memUsed, memPercent := getMemoryInfo()
		memGauge.Percent = int(memPercent)
		memGauge.Label = fmt.Sprintf("%s / %s (%.1f%%)",
			formatBytes(memUsed), formatBytes(memTotal), memPercent)

		// Update Swap
		swapTotal, swapUsed, swapPercent := getSwapInfo()
		swapGauge.Percent = int(swapPercent)
		if swapTotal > 0 {
			swapGauge.Label = fmt.Sprintf("%s / %s (%.1f%%)",
				formatBytes(swapUsed), formatBytes(swapTotal), swapPercent)
		} else {
			swapGauge.Label = "No Swap"
		}

		// Update System Info
		sysInfo.Text = getSystemInfo()

		// Update Process Table (with caching)
		now := time.Now()
		if now.Sub(lastProcessUpdate) > 500*time.Millisecond || cachedProcesses == nil {
			cachedProcesses = getProcesses()
			lastProcessUpdate = now
		}

		processes := make([]Process, len(cachedProcesses))
		copy(processes, cachedProcesses)

		sort.Slice(processes, func(i, j int) bool {
			return processes[i].CPU > processes[j].CPU
		})

		// Clear old row styles
		processTable.RowStyles = make(map[int]ui.Style)

		rows := [][]string{
			{"PID", "USER", "CPU%", "MEM%", "STATE", "COMMAND"},
		}

		// Calculate visible rows based on terminal height
		// Process table takes 73% of screen height (0.73 in grid layout)
		// Subtract 3 for: top border (1) + header row (1) + bottom border (1)
		termWidth, termHeight = ui.TerminalDimensions()
		processTableHeight := int(float64(termHeight) * 0.73)
		maxVisibleRows = processTableHeight - 3
		if maxVisibleRows < 5 {
			maxVisibleRows = 5
		}

		// Adjust scroll offset if needed
		if scrollOffset > len(processes)-maxVisibleRows {
			scrollOffset = len(processes) - maxVisibleRows
		}
		if scrollOffset < 0 {
			scrollOffset = 0
		}

		visibleProcesses := processes
		if len(processes) > maxVisibleRows {
			end := scrollOffset + maxVisibleRows
			if end > len(processes) {
				end = len(processes)
			}
			if scrollOffset < len(processes) {
				visibleProcesses = processes[scrollOffset:end]
			}
		}

		// Calculate command column width dynamically
		// Total fixed columns: PID(7) + USER(9) + CPU%(6) + MEM%(6) + STATE(6) = 34
		// Plus borders and padding: ~4
		// Command gets the rest
		commandWidth := termWidth - 34 - 4
		if commandWidth < 20 {
			commandWidth = 20
		}

		for _, p := range visibleProcesses {
			rows = append(rows, []string{
				strconv.Itoa(int(p.PID)),
				truncateString(p.User, 8),
				fmt.Sprintf("%.1f", p.CPU),
				fmt.Sprintf("%.1f", p.Memory),
				p.State,
				truncateString(p.Command, commandWidth),
			})
		}

		processTable.Rows = rows
		processTable.RowStyles[0] = ui.NewStyle(ui.ColorYellow, ui.ColorClear, ui.ModifierBold)

		// Adjust selectedRow if it's out of bounds
		if selectedRow >= len(rows) {
			selectedRow = len(rows) - 1
		}
		if selectedRow < 1 {
			selectedRow = 1
		}

		if selectedRow > 0 && selectedRow < len(rows) {
			processTable.RowStyles[selectedRow] = ui.NewStyle(ui.ColorBlack, ui.ColorCyan)
		}

		grid.SetRect(0, 0, termWidth, termHeight)
		ui.Render(grid)
	}

	render()

	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				grid.SetRect(0, 0, payload.Width, payload.Height)
				render()
			case "<Down>", "j":
				totalProcesses := len(cachedProcesses)
				if selectedRow < maxVisibleRows && selectedRow < totalProcesses {
					selectedRow++
				} else if scrollOffset+maxVisibleRows < totalProcesses {
					scrollOffset++
				}
				render()
			case "<Up>", "k":
				if selectedRow > 1 {
					selectedRow--
				} else if scrollOffset > 0 {
					scrollOffset--
				}
				render()
			case "<Home>":
				selectedRow = 1
				scrollOffset = 0
				render()
			case "<End>":
				totalProcesses := len(cachedProcesses)
				if totalProcesses > maxVisibleRows {
					scrollOffset = totalProcesses - maxVisibleRows
					selectedRow = maxVisibleRows
				} else {
					selectedRow = totalProcesses
				}
				render()
			case "K", "<Delete>":
				if len(cachedProcesses) > 0 {
					// Sort to match display order
					sortedProcesses := make([]Process, len(cachedProcesses))
					copy(sortedProcesses, cachedProcesses)
					sort.Slice(sortedProcesses, func(i, j int) bool {
						return sortedProcesses[i].CPU > sortedProcesses[j].CPU
					})

					idx := scrollOffset + selectedRow - 1
					if idx >= 0 && idx < len(sortedProcesses) {
						pid := sortedProcesses[idx].PID
						killProcess(pid)
					}
				}
				// Force refresh after kill
				cachedProcesses = nil
				render()
			}
		case <-ticker.C:
			// Force process list refresh on each tick
			cachedProcesses = nil
			render()
		}
	}
}

// getCPUPercent returns total CPU usage percentage
func getCPUPercent() float64 {
	percentages, err := cpu.Percent(0, false)
	if err != nil || len(percentages) == 0 {
		return 0
	}
	return percentages[0]
}

// getMemoryInfo returns memory information
func getMemoryInfo() (total, used uint64, percent float64) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0, 0
	}
	return v.Total, v.Used, v.UsedPercent
}

// getSwapInfo returns swap information
func getSwapInfo() (total, used uint64, percent float64) {
	v, err := mem.SwapMemory()
	if err != nil {
		return 0, 0, 0
	}
	return v.Total, v.Used, v.UsedPercent
}

// getSystemInfo returns formatted system information
func getSystemInfo() string {
	hostname, _ := os.Hostname()

	// Get uptime
	uptimeStr := "N/A"
	if uptime, err := host.Uptime(); err == nil {
		days := uptime / 86400
		hours := (uptime % 86400) / 3600
		minutes := (uptime % 3600) / 60
		if days > 0 {
			uptimeStr = fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
		} else if hours > 0 {
			uptimeStr = fmt.Sprintf("%dh %dm", hours, minutes)
		} else {
			uptimeStr = fmt.Sprintf("%dm", minutes)
		}
	}

	// Get load average or CPU queue length (Windows)
	loadStr := getLoadInfo()

	// Get process count
	procCount := 0
	if procs, err := process.Pids(); err == nil {
		procCount = len(procs)
	}

	// Get platform info
	platform := runtime.GOOS

	return fmt.Sprintf(
		"Hostname: %s\nUptime: %s\nLoad: %s\nProcesses: %d\nPlatform: %s",
		hostname, uptimeStr, loadStr, procCount, platform,
	)
}

// getProcesses returns a list of all running processes
func getProcesses() []Process {
	var processes []Process

	ctx := context.Background()
	pids, err := process.Pids()
	if err != nil {
		return processes
	}

	for _, pid := range pids {
		proc, err := process.NewProcess(pid)
		if err != nil {
			continue
		}

		p := Process{PID: pid}

		// Get process name
		if name, err := proc.NameWithContext(ctx); err == nil {
			p.Name = name
			p.Command = name
		}

		// Get command line
		if cmdline, err := proc.CmdlineWithContext(ctx); err == nil && cmdline != "" {
			p.Command = cmdline
		}

		// Get username
		if username, err := proc.UsernameWithContext(ctx); err == nil {
			p.User = username
		} else {
			p.User = "-"
		}

		// Get CPU percent
		if cpuPercent, err := proc.CPUPercentWithContext(ctx); err == nil {
			p.CPU = cpuPercent
		}

		// Get memory percent
		if memPercent, err := proc.MemoryPercentWithContext(ctx); err == nil {
			p.Memory = memPercent
		}

		// Get status
		if status, err := proc.StatusWithContext(ctx); err == nil && len(status) > 0 {
			p.State = status[0]
		} else {
			p.State = "-"
		}

		processes = append(processes, p)
	}

	return processes
}

// killProcess terminates a process by PID
func killProcess(pid int32) {
	proc, err := process.NewProcess(pid)
	if err != nil {
		return
	}
	proc.Kill()
}

// formatBytes converts bytes to human-readable format
func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
