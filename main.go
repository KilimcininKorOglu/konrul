// Konrul - Terminal Based System Monitor
// Named after the mythological Turkish phoenix-like creature
// A lightweight htop-like system monitor written in Go

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

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
	PID     int
	Name    string
	State   string
	CPU     float64
	Memory  float64
	User    string
	Command string
}

// CPUStats holds CPU timing information
type CPUStats struct {
	User   uint64
	Nice   uint64
	System uint64
	Idle   uint64
	Total  uint64
}

var prevCPUStats []CPUStats

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

	render := func() {
		// Update CPU
		cpuPercent := getCPUPercent()
		cpuGauge.Percent = int(cpuPercent)
		cpuGauge.Label = fmt.Sprintf("%.1f%%", cpuPercent)

		// Update CPU history
		cpuHistory = append(cpuHistory[1:], cpuPercent)
		cpuSparkline.Data = cpuHistory

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
		uptime := getUptime()
		loadAvg := getLoadAverage()
		hostname, _ := os.Hostname()
		sysInfo.Text = fmt.Sprintf(
			"Hostname: %s\nUptime: %s\nLoad: %s\nProcesses: %d",
			hostname, uptime, loadAvg, getProcessCount(),
		)

		// Update Process Table
		processes := getProcesses()
		sort.Slice(processes, func(i, j int) bool {
			return processes[i].CPU > processes[j].CPU
		})

		rows := [][]string{
			{"PID", "USER", "CPU%", "MEM%", "STATE", "COMMAND"},
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

		for _, p := range visibleProcesses {
			rows = append(rows, []string{
				strconv.Itoa(p.PID),
				truncateString(p.User, 8),
				fmt.Sprintf("%.1f", p.CPU),
				fmt.Sprintf("%.1f", p.Memory),
				p.State,
				truncateString(p.Command, 40),
			})
		}

		processTable.Rows = rows
		processTable.RowStyles[0] = ui.NewStyle(ui.ColorYellow, ui.ColorClear, ui.ModifierBold)

		if selectedRow > 0 && selectedRow < len(rows) {
			processTable.RowStyles[selectedRow] = ui.NewStyle(ui.ColorBlack, ui.ColorCyan)
		}

		termWidth, termHeight = ui.TerminalDimensions()
		grid.SetRect(0, 0, termWidth, termHeight)
		maxVisibleRows = termHeight/2 - 5

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
				render()
			case "<Down>", "j":
				processes := getProcesses()
				if selectedRow < maxVisibleRows && selectedRow < len(processes) {
					processTable.RowStyles[selectedRow] = ui.NewStyle(ui.ColorWhite)
					selectedRow++
				} else if scrollOffset+maxVisibleRows < len(processes) {
					scrollOffset++
				}
				render()
			case "<Up>", "k":
				if selectedRow > 1 {
					processTable.RowStyles[selectedRow] = ui.NewStyle(ui.ColorWhite)
					selectedRow--
				} else if scrollOffset > 0 {
					scrollOffset--
				}
				render()
			case "<Home>":
				processTable.RowStyles[selectedRow] = ui.NewStyle(ui.ColorWhite)
				selectedRow = 1
				scrollOffset = 0
				render()
			case "<End>":
				processes := getProcesses()
				scrollOffset = len(processes) - maxVisibleRows
				if scrollOffset < 0 {
					scrollOffset = 0
				}
				render()
			case "K", "<Delete>":
				processes := getProcesses()
				sort.Slice(processes, func(i, j int) bool {
					return processes[i].CPU > processes[j].CPU
				})
				idx := scrollOffset + selectedRow - 1
				if idx >= 0 && idx < len(processes) {
					pid := processes[idx].PID
					proc, err := os.FindProcess(pid)
					if err == nil {
						proc.Kill()
					}
				}
				render()
			}
		case <-ticker.C:
			render()
		}
	}
}

// getCPUPercent calculates total CPU usage percentage
func getCPUPercent() float64 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentStats []CPUStats

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				user, _ := strconv.ParseUint(fields[1], 10, 64)
				nice, _ := strconv.ParseUint(fields[2], 10, 64)
				system, _ := strconv.ParseUint(fields[3], 10, 64)
				idle, _ := strconv.ParseUint(fields[4], 10, 64)

				stats := CPUStats{
					User:   user,
					Nice:   nice,
					System: system,
					Idle:   idle,
					Total:  user + nice + system + idle,
				}
				currentStats = append(currentStats, stats)
			}
			break
		}
	}

	if len(prevCPUStats) == 0 {
		prevCPUStats = currentStats
		return 0
	}

	if len(currentStats) > 0 && len(prevCPUStats) > 0 {
		curr := currentStats[0]
		prev := prevCPUStats[0]

		totalDelta := float64(curr.Total - prev.Total)
		idleDelta := float64(curr.Idle - prev.Idle)

		if totalDelta > 0 {
			prevCPUStats = currentStats
			return ((totalDelta - idleDelta) / totalDelta) * 100
		}
	}

	prevCPUStats = currentStats
	return 0
}

// getMemoryInfo reads memory information from /proc/meminfo
func getMemoryInfo() (total, used uint64, percent float64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var memTotal, memAvailable uint64

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		value, _ := strconv.ParseUint(fields[1], 10, 64)
		value *= 1024

		switch fields[0] {
		case "MemTotal:":
			memTotal = value
		case "MemAvailable:":
			memAvailable = value
		}
	}

	used = memTotal - memAvailable
	if memTotal > 0 {
		percent = float64(used) / float64(memTotal) * 100
	}
	return memTotal, used, percent
}

// getSwapInfo reads swap information from /proc/meminfo
func getSwapInfo() (total, used uint64, percent float64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var swapTotal, swapFree uint64

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		value, _ := strconv.ParseUint(fields[1], 10, 64)
		value *= 1024

		switch fields[0] {
		case "SwapTotal:":
			swapTotal = value
		case "SwapFree:":
			swapFree = value
		}
	}

	used = swapTotal - swapFree
	if swapTotal > 0 {
		percent = float64(used) / float64(swapTotal) * 100
	}
	return swapTotal, used, percent
}

// getUptime returns system uptime in human-readable format
func getUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "N/A"
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return "N/A"
	}

	seconds, _ := strconv.ParseFloat(fields[0], 64)
	duration := time.Duration(seconds) * time.Second

	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// getLoadAverage returns system load average
func getLoadAverage() string {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return "N/A"
	}

	fields := strings.Fields(string(data))
	if len(fields) >= 3 {
		return fmt.Sprintf("%s %s %s", fields[0], fields[1], fields[2])
	}
	return "N/A"
}

// getProcessCount returns total number of processes
func getProcessCount() int {
	dirs, _ := filepath.Glob("/proc/[0-9]*")
	return len(dirs)
}

// getProcesses returns a list of all running processes
func getProcesses() []Process {
	var processes []Process

	dirs, err := filepath.Glob("/proc/[0-9]*")
	if err != nil {
		return processes
	}

	memTotal, _, _ := getMemoryInfo()

	for _, dir := range dirs {
		pid, err := strconv.Atoi(filepath.Base(dir))
		if err != nil {
			continue
		}

		proc := Process{PID: pid}

		// Read process name from comm
		if comm, err := os.ReadFile(filepath.Join(dir, "comm")); err == nil {
			proc.Name = strings.TrimSpace(string(comm))
			proc.Command = proc.Name
		}

		// Read full command line
		if cmdline, err := os.ReadFile(filepath.Join(dir, "cmdline")); err == nil {
			cmd := strings.ReplaceAll(string(cmdline), "\x00", " ")
			cmd = strings.TrimSpace(cmd)
			if cmd != "" {
				proc.Command = cmd
			}
		}

		// Read process status
		if status, err := os.Open(filepath.Join(dir, "status")); err == nil {
			scanner := bufio.NewScanner(status)
			for scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				if len(fields) < 2 {
					continue
				}
				switch fields[0] {
				case "State:":
					proc.State = fields[1]
				case "Uid:":
					proc.User = getUsername(fields[1])
				case "VmRSS:":
					if len(fields) >= 2 {
						rss, _ := strconv.ParseUint(fields[1], 10, 64)
						rss *= 1024
						if memTotal > 0 {
							proc.Memory = float64(rss) / float64(memTotal) * 100
						}
					}
				}
			}
			status.Close()
		}

		// Read CPU time from stat
		if stat, err := os.ReadFile(filepath.Join(dir, "stat")); err == nil {
			fields := strings.Fields(string(stat))
			if len(fields) >= 14 {
				utime, _ := strconv.ParseUint(fields[13], 10, 64)
				stime, _ := strconv.ParseUint(fields[14], 10, 64)
				proc.CPU = float64(utime+stime) / 100.0
			}
		}

		processes = append(processes, proc)
	}

	return processes
}

// getUsername converts UID to username
func getUsername(uid string) string {
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return uid
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) >= 3 && fields[2] == uid {
			return fields[0]
		}
	}
	return uid
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
	return s[:maxLen-3] + "..."
}
