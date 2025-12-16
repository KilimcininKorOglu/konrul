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
	"sync"
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
	User    uint64
	Nice    uint64
	System  uint64
	Idle    uint64
	IOWait  uint64
	IRQ     uint64
	SoftIRQ uint64
	Total   uint64
}

// ProcessCPUStats holds per-process CPU timing for delta calculation
type ProcessCPUStats struct {
	UTime     uint64
	STime     uint64
	StartTime uint64
	LastCheck time.Time
}

var (
	prevCPUStats     CPUStats
	prevCPUStatsInit bool
	processCPUStats  = make(map[int]ProcessCPUStats)
	processCPUMutex  sync.RWMutex
	usernameCache    = make(map[string]string)
	usernameMutex    sync.RWMutex
	systemBootTime   uint64
	clkTck           float64 = 100 // Usually 100 on Linux, from sysconf(_SC_CLK_TCK)
)

func main() {
	// Get system boot time for accurate CPU calculations
	systemBootTime = getSystemBootTime()

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

	// Cache for processes to avoid repeated filesystem reads
	var cachedProcesses []Process
	var lastProcessUpdate time.Time

	render := func() {
		// Update CPU
		cpuPercent := getCPUPercent()
		cpuGauge.Percent = int(cpuPercent)
		cpuGauge.Label = fmt.Sprintf("%.1f%%", cpuPercent)

		// Update CPU history (convert to int for sparkline)
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
		uptime := getUptime()
		loadAvg := getLoadAverage()
		hostname, _ := os.Hostname()
		sysInfo.Text = fmt.Sprintf(
			"Hostname: %s\nUptime: %s\nLoad: %s\nProcesses: %d",
			hostname, uptime, loadAvg, getProcessCount(),
		)

		// Update Process Table (with caching to reduce I/O)
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
		termWidth, termHeight = ui.TerminalDimensions()
		maxVisibleRows = termHeight/2 - 5
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
						if proc, err := os.FindProcess(pid); err == nil {
							proc.Signal(os.Kill)
						}
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

// getSystemBootTime reads boot time from /proc/stat
func getSystemBootTime() uint64 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "btime ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				btime, _ := strconv.ParseUint(fields[1], 10, 64)
				return btime
			}
		}
	}
	return 0
}

// getCPUPercent calculates total CPU usage percentage using delta method
func getCPUPercent() float64 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentStats CPUStats

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) >= 8 {
				currentStats.User, _ = strconv.ParseUint(fields[1], 10, 64)
				currentStats.Nice, _ = strconv.ParseUint(fields[2], 10, 64)
				currentStats.System, _ = strconv.ParseUint(fields[3], 10, 64)
				currentStats.Idle, _ = strconv.ParseUint(fields[4], 10, 64)
				currentStats.IOWait, _ = strconv.ParseUint(fields[5], 10, 64)
				currentStats.IRQ, _ = strconv.ParseUint(fields[6], 10, 64)
				currentStats.SoftIRQ, _ = strconv.ParseUint(fields[7], 10, 64)

				currentStats.Total = currentStats.User + currentStats.Nice +
					currentStats.System + currentStats.Idle +
					currentStats.IOWait + currentStats.IRQ + currentStats.SoftIRQ
			}
			break
		}
	}

	if !prevCPUStatsInit {
		prevCPUStats = currentStats
		prevCPUStatsInit = true
		return 0
	}

	totalDelta := float64(currentStats.Total - prevCPUStats.Total)
	idleDelta := float64(currentStats.Idle + currentStats.IOWait - prevCPUStats.Idle - prevCPUStats.IOWait)

	prevCPUStats = currentStats

	if totalDelta > 0 {
		return ((totalDelta - idleDelta) / totalDelta) * 100
	}
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
		value *= 1024 // Convert from kB to bytes

		switch fields[0] {
		case "MemTotal:":
			memTotal = value
		case "MemAvailable:":
			memAvailable = value
		}

		// Early exit if we have both values
		if memTotal > 0 && memAvailable > 0 {
			break
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
	foundTotal, foundFree := false, false

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		value, _ := strconv.ParseUint(fields[1], 10, 64)
		value *= 1024 // Convert from kB to bytes

		switch fields[0] {
		case "SwapTotal:":
			swapTotal = value
			foundTotal = true
		case "SwapFree:":
			swapFree = value
			foundFree = true
		}

		// Early exit if we have both values
		if foundTotal && foundFree {
			break
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

	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "N/A"
	}

	days := int(seconds) / 86400
	hours := (int(seconds) % 86400) / 3600
	minutes := (int(seconds) % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
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

// getProcesses returns a list of all running processes with CPU usage
func getProcesses() []Process {
	var processes []Process

	dirs, err := filepath.Glob("/proc/[0-9]*")
	if err != nil {
		return processes
	}

	memTotal, _, _ := getMemoryInfo()
	now := time.Now()

	// Get total CPU time for percentage calculation
	totalCPUTime := getTotalCPUTime()

	processCPUMutex.Lock()
	defer processCPUMutex.Unlock()

	// Track which PIDs we've seen for cleanup
	seenPIDs := make(map[int]bool)

	for _, dir := range dirs {
		pid, err := strconv.Atoi(filepath.Base(dir))
		if err != nil {
			continue
		}
		seenPIDs[pid] = true

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

		// Read process status for state, uid, and memory
		if status, err := os.Open(filepath.Join(dir, "status")); err == nil {
			scanner := bufio.NewScanner(status)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "State:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						proc.State = fields[1]
					}
				} else if strings.HasPrefix(line, "Uid:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						proc.User = getCachedUsername(fields[1])
					}
				} else if strings.HasPrefix(line, "VmRSS:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						rss, _ := strconv.ParseUint(fields[1], 10, 64)
						rss *= 1024 // Convert from kB to bytes
						if memTotal > 0 {
							proc.Memory = float64(rss) / float64(memTotal) * 100
						}
					}
				}
			}
			status.Close()
		}

		// Read CPU time from stat and calculate percentage
		if stat, err := os.ReadFile(filepath.Join(dir, "stat")); err == nil {
			// Parse stat file - format: pid (comm) state ppid ...
			// Fields 14 and 15 (1-indexed) are utime and stime
			statStr := string(stat)

			// Find the end of comm field (after the closing parenthesis)
			commEnd := strings.LastIndex(statStr, ")")
			if commEnd > 0 && commEnd+2 < len(statStr) {
				fields := strings.Fields(statStr[commEnd+2:])
				if len(fields) >= 13 {
					utime, _ := strconv.ParseUint(fields[11], 10, 64) // 14th field (0-indexed: 11 after comm)
					stime, _ := strconv.ParseUint(fields[12], 10, 64) // 15th field

					currentCPUTime := utime + stime

					// Calculate CPU percentage using delta
					if prevStats, exists := processCPUStats[pid]; exists {
						timeDelta := now.Sub(prevStats.LastCheck).Seconds()
						if timeDelta > 0 && totalCPUTime > 0 {
							cpuDelta := float64(currentCPUTime - prevStats.UTime - prevStats.STime)
							// CPU percentage = (process CPU ticks / total CPU ticks) * 100 * num_cpus
							// Simplified: (delta_ticks / clkTck) / time_delta * 100
							proc.CPU = (cpuDelta / clkTck) / timeDelta * 100
							if proc.CPU < 0 {
								proc.CPU = 0
							}
							if proc.CPU > 100 {
								proc.CPU = 100
							}
						}
					}

					// Update stats for next calculation
					processCPUStats[pid] = ProcessCPUStats{
						UTime:     utime,
						STime:     stime,
						LastCheck: now,
					}
				}
			}
		}

		processes = append(processes, proc)
	}

	// Clean up old process stats
	for pid := range processCPUStats {
		if !seenPIDs[pid] {
			delete(processCPUStats, pid)
		}
	}

	return processes
}

// getTotalCPUTime returns total CPU time for all CPUs
func getTotalCPUTime() uint64 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				var total uint64
				for i := 1; i < len(fields); i++ {
					val, _ := strconv.ParseUint(fields[i], 10, 64)
					total += val
				}
				return total
			}
		}
	}
	return 0
}

// getCachedUsername converts UID to username with caching
func getCachedUsername(uid string) string {
	usernameMutex.RLock()
	if username, exists := usernameCache[uid]; exists {
		usernameMutex.RUnlock()
		return username
	}
	usernameMutex.RUnlock()

	// Not in cache, look it up
	username := lookupUsername(uid)

	usernameMutex.Lock()
	usernameCache[uid] = username
	usernameMutex.Unlock()

	return username
}

// lookupUsername reads /etc/passwd to find username for UID
func lookupUsername(uid string) string {
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
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
