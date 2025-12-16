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
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
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

// SortMode represents the process sorting mode
type SortMode int

const (
	SortByCPU SortMode = iota
	SortByMem
	SortByPID
	SortByName
)

func (s SortMode) String() string {
	switch s {
	case SortByCPU:
		return "CPU%"
	case SortByMem:
		return "MEM%"
	case SortByPID:
		return "PID"
	case SortByName:
		return "NAME"
	default:
		return "CPU%"
	}
}

// NetworkStats holds network I/O statistics
type NetworkStats struct {
	BytesRecv   uint64
	BytesSent   uint64
	LastRecv    uint64
	LastSent    uint64
	RecvPerSec  uint64
	SentPerSec  uint64
	LastUpdate  time.Time
}

// DiskStats holds disk I/O statistics
type DiskStats struct {
	ReadBytes    uint64
	WriteBytes   uint64
	LastRead     uint64
	LastWrite    uint64
	ReadPerSec   uint64
	WritePerSec  uint64
	LastUpdate   time.Time
}

var netStats NetworkStats
var diskStats DiskStats

func main() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	// CPU Gauge (Total)
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

	// Network I/O Info
	netInfo := widgets.NewParagraph()
	netInfo.Title = " Network "
	netInfo.BorderStyle.Fg = ui.ColorCyan

	// Disk I/O Info
	diskInfo := widgets.NewParagraph()
	diskInfo.Title = " Disk "
	diskInfo.BorderStyle.Fg = ui.ColorCyan

	// Per-core CPU BarChart
	cpuCores := widgets.NewBarChart()
	cpuCores.Title = " CPU Cores "
	cpuCores.BorderStyle.Fg = ui.ColorCyan
	cpuCores.BarColors = []ui.Color{ui.ColorGreen, ui.ColorYellow, ui.ColorRed, ui.ColorCyan, ui.ColorMagenta, ui.ColorBlue, ui.ColorWhite}
	cpuCores.NumStyles = []ui.Style{ui.NewStyle(ui.ColorBlack)}
	cpuCores.LabelStyles = []ui.Style{ui.NewStyle(ui.ColorCyan)}
	cpuCores.BarWidth = 3
	cpuCores.BarGap = 1

	// Get initial core count
	numCores := runtime.NumCPU()
	coreLabels := make([]string, numCores)
	for i := 0; i < numCores; i++ {
		coreLabels[i] = fmt.Sprintf("%d", i)
	}
	cpuCores.Labels = coreLabels

	// Layout
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	grid.Set(
		ui.NewRow(0.10,
			ui.NewCol(0.33, cpuGauge),
			ui.NewCol(0.33, memGauge),
			ui.NewCol(0.34, swapGauge),
		),
		ui.NewRow(0.15,
			ui.NewCol(0.40, cpuCores),
			ui.NewCol(0.20, netInfo),
			ui.NewCol(0.20, diskInfo),
			ui.NewCol(0.20, sysInfo),
		),
		ui.NewRow(0.75, processTable),
	)

	selectedRow := 1
	scrollOffset := 0
	maxVisibleRows := 15

	// Sorting options
	sortMode := SortByCPU
	sortReverse := false

	// Cache for processes
	var cachedProcesses []Process
	var lastProcessUpdate time.Time

	render := func() {
		// Update Total CPU
		cpuPercent := getCPUPercent()
		cpuGauge.Percent = int(cpuPercent)
		cpuGauge.Label = fmt.Sprintf("%.1f%%", cpuPercent)

		// Update Per-core CPU
		corePercents := getPerCoreCPU()
		cpuCores.Data = corePercents
		// Update labels if core count changed
		if len(corePercents) != len(cpuCores.Labels) {
			labels := make([]string, len(corePercents))
			for i := range corePercents {
				labels[i] = fmt.Sprintf("%d", i)
			}
			cpuCores.Labels = labels
		}

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

		// Update Network Info
		netInfo.Text = getNetworkInfo()

		// Update Disk Info
		diskInfo.Text = getDiskInfo()

		// Update Process Table (with caching)
		now := time.Now()
		if now.Sub(lastProcessUpdate) > 500*time.Millisecond || cachedProcesses == nil {
			cachedProcesses = getProcesses()
			lastProcessUpdate = now
		}

		processes := make([]Process, len(cachedProcesses))
		copy(processes, cachedProcesses)

		// Sort processes based on current sort mode
		sort.Slice(processes, func(i, j int) bool {
			var less bool
			switch sortMode {
			case SortByCPU:
				less = processes[i].CPU > processes[j].CPU
			case SortByMem:
				less = processes[i].Memory > processes[j].Memory
			case SortByPID:
				less = processes[i].PID < processes[j].PID
			case SortByName:
				less = processes[i].Name < processes[j].Name
			default:
				less = processes[i].CPU > processes[j].CPU
			}
			if sortReverse {
				return !less
			}
			return less
		})

		// Clear old row styles
		processTable.RowStyles = make(map[int]ui.Style)

		// Update process table title with sort info
		sortIndicator := ""
		if sortReverse {
			sortIndicator = " [R]"
		}
		processTable.Title = fmt.Sprintf(" Processes [c:CPU m:MEM p:PID n:NAME r:Rev] Sort:%s%s ", sortMode.String(), sortIndicator)

		rows := [][]string{
			{"PID", "USER", "CPU%", "MEM%", "STATE", "COMMAND"},
		}

		// Calculate visible rows based on terminal height
		// Process table takes 75% of screen height (0.75 in grid layout)
		// Subtract 3 for: top border (1) + header row (1) + bottom border (1)
		termWidth, termHeight = ui.TerminalDimensions()
		processTableHeight := int(float64(termHeight) * 0.75)
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
						var less bool
						switch sortMode {
						case SortByCPU:
							less = sortedProcesses[i].CPU > sortedProcesses[j].CPU
						case SortByMem:
							less = sortedProcesses[i].Memory > sortedProcesses[j].Memory
						case SortByPID:
							less = sortedProcesses[i].PID < sortedProcesses[j].PID
						case SortByName:
							less = sortedProcesses[i].Name < sortedProcesses[j].Name
						default:
							less = sortedProcesses[i].CPU > sortedProcesses[j].CPU
						}
						if sortReverse {
							return !less
						}
						return less
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
			// Sorting keys
			case "c":
				sortMode = SortByCPU
				render()
			case "m":
				sortMode = SortByMem
				render()
			case "p":
				sortMode = SortByPID
				render()
			case "n":
				sortMode = SortByName
				render()
			case "r":
				sortReverse = !sortReverse
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

// getPerCoreCPU returns CPU usage percentage for each core
func getPerCoreCPU() []float64 {
	percentages, err := cpu.Percent(0, true)
	if err != nil || len(percentages) == 0 {
		return []float64{}
	}
	return percentages
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

// getNetworkInfo returns formatted network I/O information
func getNetworkInfo() string {
	counters, err := net.IOCounters(false) // false = aggregate all interfaces
	if err != nil || len(counters) == 0 {
		return "RX: N/A\nTX: N/A"
	}

	now := time.Now()
	currentRecv := counters[0].BytesRecv
	currentSent := counters[0].BytesSent

	// Calculate per-second rates
	if !netStats.LastUpdate.IsZero() {
		elapsed := now.Sub(netStats.LastUpdate).Seconds()
		if elapsed > 0 {
			netStats.RecvPerSec = uint64(float64(currentRecv-netStats.LastRecv) / elapsed)
			netStats.SentPerSec = uint64(float64(currentSent-netStats.LastSent) / elapsed)
		}
	}

	// Update stats
	netStats.BytesRecv = currentRecv
	netStats.BytesSent = currentSent
	netStats.LastRecv = currentRecv
	netStats.LastSent = currentSent
	netStats.LastUpdate = now

	return fmt.Sprintf(
		"RX: %s\nTX: %s\nRX/s: %s\nTX/s: %s",
		formatBytes(netStats.BytesRecv),
		formatBytes(netStats.BytesSent),
		formatBytesPerSec(netStats.RecvPerSec),
		formatBytesPerSec(netStats.SentPerSec),
	)
}

// getDiskInfo returns formatted disk I/O and usage information
func getDiskInfo() string {
	// Get disk I/O stats
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return "Read: N/A\nWrite: N/A"
	}

	// Sum up all disk I/O
	var totalRead, totalWrite uint64
	for _, counter := range ioCounters {
		totalRead += counter.ReadBytes
		totalWrite += counter.WriteBytes
	}

	now := time.Now()

	// Calculate per-second rates
	if !diskStats.LastUpdate.IsZero() {
		elapsed := now.Sub(diskStats.LastUpdate).Seconds()
		if elapsed > 0 {
			diskStats.ReadPerSec = uint64(float64(totalRead-diskStats.LastRead) / elapsed)
			diskStats.WritePerSec = uint64(float64(totalWrite-diskStats.LastWrite) / elapsed)
		}
	}

	// Update stats
	diskStats.ReadBytes = totalRead
	diskStats.WriteBytes = totalWrite
	diskStats.LastRead = totalRead
	diskStats.LastWrite = totalWrite
	diskStats.LastUpdate = now

	// Get disk usage for root partition
	usageStr := ""
	if usage, err := disk.Usage("/"); err == nil {
		usageStr = fmt.Sprintf("Used: %.1f%%\n", usage.UsedPercent)
	} else if runtime.GOOS == "windows" {
		// Try C: drive on Windows
		if usage, err := disk.Usage("C:"); err == nil {
			usageStr = fmt.Sprintf("C: %.1f%%\n", usage.UsedPercent)
		}
	}

	return fmt.Sprintf(
		"%sR: %s/s\nW: %s/s",
		usageStr,
		formatBytesPerSec(diskStats.ReadPerSec),
		formatBytesPerSec(diskStats.WritePerSec),
	)
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

	// Get CPU core count
	numCores := runtime.NumCPU()

	// Get platform info
	platform := runtime.GOOS

	return fmt.Sprintf(
		"Hostname: %s\nUptime: %s\nLoad: %s\nCores: %d\nProcesses: %d\nPlatform: %s",
		hostname, uptimeStr, loadStr, numCores, procCount, platform,
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

// formatBytesPerSec converts bytes per second to human-readable format
func formatBytesPerSec(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fG/s", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fM/s", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fK/s", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB/s", bytes)
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
