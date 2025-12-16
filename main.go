// Konrul - Terminal Based System Monitor
// Named after the mythological Turkish phoenix-like creature
// A lightweight htop-like system monitor written in Go
// Cross-platform support: Linux, macOS, Windows, FreeBSD

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
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
	PPID    int32
	Name    string
	State   string
	CPU     float64
	Memory  float32
	User    string
	Command string
	Depth   int // Tree depth for indentation
}

// SortMode represents the process sorting mode
type SortMode int

const (
	SortByCPU SortMode = iota
	SortByMem
	SortByPID
	SortByName
	SortByGPUMem  // For GPU view
	SortByGPUPerc // For GPU view
)

// ViewMode represents the current view mode
type ViewMode int

const (
	ViewModeNormal ViewMode = iota // All processes
	ViewModeGPU                     // GPU processes only
	ViewModeDocker                  // Docker containers
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
	case SortByGPUMem:
		return "GPU_MEM"
	case SortByGPUPerc:
		return "GPU%"
	default:
		return "CPU%"
	}
}

func (v ViewMode) String() string {
	switch v {
	case ViewModeGPU:
		return "GPU"
	case ViewModeDocker:
		return "Docker"
	default:
		return "Normal"
	}
}

// getSortModeFromString converts a string to SortMode
func getSortModeFromString(s string) SortMode {
	switch strings.ToLower(s) {
	case "cpu":
		return SortByCPU
	case "mem", "memory":
		return SortByMem
	case "pid":
		return SortByPID
	case "name":
		return SortByName
	default:
		return SortByCPU
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

// Command-line flags
var (
	showVersion     bool
	refreshInterval int
	defaultSort     string
	startTreeView   bool
	configFile      string
)

func init() {
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showVersion, "v", false, "Show version information (shorthand)")
	flag.IntVar(&refreshInterval, "interval", 0, "Refresh interval in seconds (1-10)")
	flag.IntVar(&refreshInterval, "i", 0, "Refresh interval in seconds (shorthand)")
	flag.StringVar(&defaultSort, "sort", "", "Default sort mode (cpu, mem, pid, name)")
	flag.StringVar(&defaultSort, "s", "", "Default sort mode (shorthand)")
	flag.BoolVar(&startTreeView, "tree", false, "Start in tree view mode")
	flag.BoolVar(&startTreeView, "t", false, "Start in tree view mode (shorthand)")
	flag.StringVar(&configFile, "config", "", "Path to config file")
	flag.StringVar(&configFile, "c", "", "Path to config file (shorthand)")
}

func main() {
	flag.Parse()

	// Handle version flag
	if showVersion {
		fmt.Printf("Konrul %s\n", version)
		fmt.Printf("  Commit: %s\n", commit)
		fmt.Printf("  Built:  %s\n", date)
		fmt.Printf("  Go:     %s\n", runtime.Version())
		fmt.Printf("  OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	// Load config file
	config := LoadConfig()

	// Command-line flags override config file
	if refreshInterval == 0 {
		refreshInterval = config.RefreshInterval
	}
	if defaultSort == "" {
		defaultSort = config.DefaultSort
	}
	if !startTreeView {
		startTreeView = config.TreeView
	}

	// Validate refresh interval
	if refreshInterval < 1 {
		refreshInterval = 1
	} else if refreshInterval > 10 {
		refreshInterval = 10
	}

	// Initialize CPU metrics before starting UI (first call can be slow)
	cpu.Percent(100*time.Millisecond, false)
	cpu.Percent(100*time.Millisecond, true)

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	// Initialize theme from config
	SetTheme(config.Theme)
	theme := GetCurrentTheme()

	// CPU Gauge (Total)
	cpuGauge := widgets.NewGauge()
	cpuGauge.Title = " CPU "
	cpuGauge.BarColor = theme.CPUColor
	cpuGauge.BorderStyle.Fg = theme.BorderColor

	// Memory Gauge
	memGauge := widgets.NewGauge()
	memGauge.Title = " Memory "
	memGauge.BarColor = theme.MemoryColor
	memGauge.BorderStyle.Fg = theme.BorderColor

	// Swap Gauge
	swapGauge := widgets.NewGauge()
	swapGauge.Title = " Swap "
	swapGauge.BarColor = theme.SwapColor
	swapGauge.BorderStyle.Fg = theme.BorderColor

	// Process Table
	processTable := widgets.NewTable()
	processTable.Title = " Processes (Up/Down: scroll, q: quit, K: kill) "
	processTable.TextStyle = ui.NewStyle(theme.TextColor)
	processTable.RowSeparator = false
	processTable.BorderStyle.Fg = theme.BorderColor
	processTable.TextAlignment = ui.AlignLeft
	// Column widths: PID(7), USER(9), CPU%(6), MEM%(6), STATE(6), COMMAND(remaining)
	processTable.ColumnWidths = []int{7, 9, 6, 6, 6, -1}

	// System Info
	sysInfo := widgets.NewParagraph()
	sysInfo.Title = " System "
	sysInfo.BorderStyle.Fg = theme.BorderColor

	// Network I/O Info
	netInfo := widgets.NewParagraph()
	netInfo.Title = " Network "
	netInfo.BorderStyle.Fg = theme.BorderColor

	// Disk I/O Info
	diskInfo := widgets.NewParagraph()
	diskInfo.Title = " Disk "
	diskInfo.BorderStyle.Fg = theme.BorderColor

	// GPU/Docker Info (shared panel, toggle with 'd')
	gpuInfo := widgets.NewParagraph()
	gpuInfo.Title = " GPU "
	gpuInfo.BorderStyle.Fg = theme.BorderColor

	// Docker Info
	dockerInfo := widgets.NewParagraph()
	dockerInfo.Title = " Docker "
	dockerInfo.BorderStyle.Fg = theme.BorderColor

	// Status Bar (bottom)
	statusBar := widgets.NewParagraph()
	statusBar.Border = false
	statusBar.Text = " F1:Help F8:Sort F9:Kill F10:Quit | /:Search t:Tree d:Docker T:Theme "
	statusBar.TextStyle = ui.NewStyle(ui.ColorBlack, ui.ColorWhite)

	// Per-core CPU BarChart
	cpuCores := widgets.NewBarChart()
	cpuCores.Title = " CPU Cores "
	cpuCores.BorderStyle.Fg = theme.BorderColor
	cpuCores.BarColors = theme.BarColors
	cpuCores.NumStyles = []ui.Style{ui.NewStyle(ui.ColorBlack)}
	cpuCores.LabelStyles = []ui.Style{ui.NewStyle(theme.LabelColor)}
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

	selectedRow := 1
	scrollOffset := 0
	maxVisibleRows := 15

	// Sorting options (apply command-line defaults)
	sortMode := getSortModeFromString(defaultSort)
	sortReverse := false
	treeView := startTreeView

	// Search/filter options
	searchMode := false
	searchQuery := ""
	showHelp := false
	showDocker := false // Toggle between GPU and Docker panel
	viewMode := ViewModeNormal // Current view mode (Normal, GPU, Docker)

	// Cache for GPU processes
	var cachedGPUProcesses []GPUProcess

	// Function to update grid layout based on showDocker toggle
	updateGridLayout := func() {
		var infoPanel *widgets.Paragraph
		if showDocker {
			infoPanel = dockerInfo
		} else {
			infoPanel = gpuInfo
		}
		grid.Set(
			ui.NewRow(0.10,
				ui.NewCol(0.33, cpuGauge),
				ui.NewCol(0.33, memGauge),
				ui.NewCol(0.34, swapGauge),
			),
			ui.NewRow(0.14,
				ui.NewCol(0.35, cpuCores),
				ui.NewCol(0.15, netInfo),
				ui.NewCol(0.15, diskInfo),
				ui.NewCol(0.15, infoPanel),
				ui.NewCol(0.20, sysInfo),
			),
			ui.NewRow(0.73, processTable),
			ui.NewRow(0.03, statusBar),
		)
	}
	updateGridLayout() // Initial layout

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

		// Update GPU/Docker Info based on toggle
		if showDocker {
			dockerInfo.Text = FormatDockerInfo()
		} else {
			gpuInfo.Text = FormatGPUInfo()
		}

		// Update Process Table (with caching)
		now := time.Now()
		if now.Sub(lastProcessUpdate) > 500*time.Millisecond || cachedProcesses == nil {
			cachedProcesses = getProcesses()
			lastProcessUpdate = now
		}

		processes := make([]Process, len(cachedProcesses))
		copy(processes, cachedProcesses)

		// Apply search filter
		if searchQuery != "" {
			processes = filterProcesses(processes, searchQuery)
		}

		// Apply tree view or flat sorting
		if treeView {
			processes = buildProcessTree(processes)
		} else {
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
		}

		// Clear old row styles
		processTable.RowStyles = make(map[int]ui.Style)

		// Update process table title with sort info
		sortIndicator := ""
		if sortReverse {
			sortIndicator = " [R]"
		}
		treeIndicator := ""
		if treeView && viewMode == ViewModeNormal {
			treeIndicator = " [Tree]"
		}

		// Set table title based on view mode
		var rows [][]string
		switch viewMode {
		case ViewModeGPU:
			if searchMode {
				processTable.Title = fmt.Sprintf(" Search: %s_ (Enter:confirm, Esc:cancel) ", searchQuery)
			} else {
				processTable.Title = fmt.Sprintf(" GPU Processes [g:GPU%% G:GPU_MEM d:Normal] Sort:%s%s ", sortMode.String(), sortIndicator)
			}
			rows = [][]string{
				{"PID", "USER", "GPU%", "GPU_MEM", "TYPE", "COMMAND"},
			}
		case ViewModeDocker:
			if searchMode {
				processTable.Title = fmt.Sprintf(" Search: %s_ (Enter:confirm, Esc:cancel) ", searchQuery)
			} else {
				processTable.Title = fmt.Sprintf(" Docker Containers [d:Normal] Sort:%s%s ", sortMode.String(), sortIndicator)
			}
			rows = [][]string{
				{"CONTAINER", "IMAGE", "CPU%", "MEM", "IP", "PORTS"},
			}
		default:
			if searchMode {
				processTable.Title = fmt.Sprintf(" Search: %s_ (Enter:confirm, Esc:cancel) ", searchQuery)
			} else if searchQuery != "" {
				processTable.Title = fmt.Sprintf(" Processes [/:search Esc:clear] Filter:\"%s\" Sort:%s%s%s ", searchQuery, sortMode.String(), sortIndicator, treeIndicator)
			} else {
				processTable.Title = fmt.Sprintf(" Processes [/:search c:CPU m:MEM p:PID n:NAME r:Rev t:Tree d:GPU/Docker] Sort:%s%s%s ", sortMode.String(), sortIndicator, treeIndicator)
			}
			rows = [][]string{
				{"PID", "USER", "CPU%", "MEM%", "STATE", "COMMAND"},
			}
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

		// Render rows based on view mode
		switch viewMode {
		case ViewModeGPU:
			// GPU Process view
			gpuProcs := cachedGPUProcesses
			if gpuProcs == nil {
				gpuProcs = GetGPUProcesses()
				cachedGPUProcesses = gpuProcs
			}

			// Sort GPU processes
			sort.Slice(gpuProcs, func(i, j int) bool {
				var less bool
				switch sortMode {
				case SortByGPUMem:
					less = gpuProcs[i].GPUMemory > gpuProcs[j].GPUMemory
				case SortByGPUPerc:
					less = gpuProcs[i].GPUPercent > gpuProcs[j].GPUPercent
				case SortByPID:
					less = gpuProcs[i].PID < gpuProcs[j].PID
				case SortByName:
					less = gpuProcs[i].Name < gpuProcs[j].Name
				default:
					less = gpuProcs[i].GPUMemory > gpuProcs[j].GPUMemory
				}
				if sortReverse {
					return !less
				}
				return less
			})

			// Adjust scroll
			if scrollOffset > len(gpuProcs)-maxVisibleRows {
				scrollOffset = len(gpuProcs) - maxVisibleRows
			}
			if scrollOffset < 0 {
				scrollOffset = 0
			}

			visibleGPUProcs := gpuProcs
			if len(gpuProcs) > maxVisibleRows {
				end := scrollOffset + maxVisibleRows
				if end > len(gpuProcs) {
					end = len(gpuProcs)
				}
				if scrollOffset < len(gpuProcs) {
					visibleGPUProcs = gpuProcs[scrollOffset:end]
				}
			}

			commandWidth := termWidth - 40 - 4
			if commandWidth < 15 {
				commandWidth = 15
			}

			if len(gpuProcs) == 0 {
				rows = append(rows, []string{"", "", "No GPU", "processes", "", "found"})
			} else {
				for _, gp := range visibleGPUProcs {
					rows = append(rows, []string{
						strconv.Itoa(int(gp.PID)),
						truncateString(getProcessUser(gp.PID), 8),
						fmt.Sprintf("%.0f%%", gp.GPUPercent),
						formatBytes(gp.GPUMemory),
						gp.Type,
						truncateString(gp.Name, commandWidth),
					})
				}
			}

		case ViewModeDocker:
			// Docker Container view
			dockerContainers := GetDockerInfo().Containers

			// Sort containers
			sort.Slice(dockerContainers, func(i, j int) bool {
				var less bool
				switch sortMode {
				case SortByCPU:
					cpuI := parsePercent(dockerContainers[i].CPUPerc)
					cpuJ := parsePercent(dockerContainers[j].CPUPerc)
					less = cpuI > cpuJ
				case SortByMem:
					less = dockerContainers[i].MemUsage > dockerContainers[j].MemUsage
				case SortByName:
					less = dockerContainers[i].Name < dockerContainers[j].Name
				default:
					cpuI := parsePercent(dockerContainers[i].CPUPerc)
					cpuJ := parsePercent(dockerContainers[j].CPUPerc)
					less = cpuI > cpuJ
				}
				if sortReverse {
					return !less
				}
				return less
			})

			// Adjust scroll
			if scrollOffset > len(dockerContainers)-maxVisibleRows {
				scrollOffset = len(dockerContainers) - maxVisibleRows
			}
			if scrollOffset < 0 {
				scrollOffset = 0
			}

			visibleContainers := dockerContainers
			if len(dockerContainers) > maxVisibleRows {
				end := scrollOffset + maxVisibleRows
				if end > len(dockerContainers) {
					end = len(dockerContainers)
				}
				if scrollOffset < len(dockerContainers) {
					visibleContainers = dockerContainers[scrollOffset:end]
				}
			}

			if len(dockerContainers) == 0 {
				rows = append(rows, []string{"", "", "No", "containers", "", "found"})
			} else {
				for _, c := range visibleContainers {
					cpu := c.CPUPerc
					if cpu == "" {
						cpu = "-"
					}
					mem := c.MemUsage
					if mem == "" {
						mem = "-"
					}
					ip := c.IPAddress
					if ip == "" {
						ip = "-"
					}
					ports := c.Ports
					if ports == "" {
						ports = "-"
					}
					// Truncate ports if too long
					if len(ports) > 20 {
						ports = ports[:17] + "..."
					}
					rows = append(rows, []string{
						truncateString(c.Name, 12),
						truncateString(c.Image, 15),
						cpu,
						truncateString(mem, 10),
						ip,
						ports,
					})
				}
			}

		default:
			// Normal Process view
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
				// Add tree indentation if in tree view
				command := p.Command
				if treeView && p.Depth > 0 {
					indent := ""
					for i := 0; i < p.Depth-1; i++ {
						indent += "  "
					}
					indent += "├─"
					command = indent + command
				}
				rows = append(rows, []string{
					strconv.Itoa(int(p.PID)),
					truncateString(p.User, 8),
					fmt.Sprintf("%.1f", p.CPU),
					fmt.Sprintf("%.1f", p.Memory),
					p.State,
					truncateString(command, commandWidth),
				})
			}
		}

		processTable.Rows = rows
		processTable.RowStyles[0] = ui.NewStyle(theme.HeaderFg, theme.HeaderBg, ui.ModifierBold)

		// Adjust selectedRow if it's out of bounds
		if selectedRow >= len(rows) {
			selectedRow = len(rows) - 1
		}
		if selectedRow < 1 {
			selectedRow = 1
		}

		if selectedRow > 0 && selectedRow < len(rows) {
			processTable.RowStyles[selectedRow] = ui.NewStyle(theme.SelectionFg, theme.SelectionBg)
		}

		// Update status bar based on view mode
		switch viewMode {
		case ViewModeGPU:
			statusBar.Text = " F1:Help F8:Sort F9:Kill F10:Quit | g:GPU% G:GPU_MEM | d:Docker/Normal | GPU Processes "
		case ViewModeDocker:
			statusBar.Text = " F1:Help F8:Sort F9:Stop F10:Quit | c:CPU m:MEM n:NAME | d:GPU/Normal | Docker Containers "
		default:
			statusBar.Text = " F1:Help F8:Sort F9:Kill F10:Quit | /:Search t:Tree d:GPU/Docker T:Theme "
		}

		grid.SetRect(0, 0, termWidth, termHeight)
		ui.Render(grid)

		// Render help overlay if active
		if showHelp {
			renderHelp(termWidth, termHeight)
		}
	}

	render()

	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Duration(refreshInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case e := <-uiEvents:
			// Handle help mode - any key closes help
			if showHelp {
				showHelp = false
				render()
				continue
			}

			// Handle search mode input
			if searchMode {
				switch e.ID {
				case "<Enter>":
					searchMode = false
					render()
				case "<Escape>":
					searchMode = false
					searchQuery = ""
					render()
				case "<Backspace>":
					if len(searchQuery) > 0 {
						searchQuery = searchQuery[:len(searchQuery)-1]
					}
					render()
				case "<Space>":
					searchQuery += " "
					render()
				default:
					// Add printable characters
					if len(e.ID) == 1 {
						searchQuery += e.ID
						render()
					}
				}
				continue
			}

			switch e.ID {
			case "q", "<C-c>", "<F10>":
				return
			case "<F1>", "?", "h":
				showHelp = true
				render()
			case "<F8>":
				// Cycle through sort modes
				switch sortMode {
				case SortByCPU:
					sortMode = SortByMem
				case SortByMem:
					sortMode = SortByPID
				case SortByPID:
					sortMode = SortByName
				case SortByName:
					sortMode = SortByCPU
				}
				render()
			case "<F9>":
				// Kill selected process (same as K)
				if len(cachedProcesses) > 0 {
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
				cachedProcesses = nil
				render()
			case "/":
				searchMode = true
				searchQuery = ""
				render()
			case "<Escape>":
				if searchQuery != "" {
					searchQuery = ""
					render()
				}
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
			case "t":
				treeView = !treeView
				render()
			case "T":
				// Cycle through themes
				NextTheme()
				theme = GetCurrentTheme()
				// Update widget colors
				cpuGauge.BarColor = theme.CPUColor
				cpuGauge.BorderStyle.Fg = theme.BorderColor
				memGauge.BarColor = theme.MemoryColor
				memGauge.BorderStyle.Fg = theme.BorderColor
				swapGauge.BarColor = theme.SwapColor
				swapGauge.BorderStyle.Fg = theme.BorderColor
				processTable.TextStyle = ui.NewStyle(theme.TextColor)
				processTable.BorderStyle.Fg = theme.BorderColor
				sysInfo.BorderStyle.Fg = theme.BorderColor
				netInfo.BorderStyle.Fg = theme.BorderColor
				diskInfo.BorderStyle.Fg = theme.BorderColor
				gpuInfo.BorderStyle.Fg = theme.BorderColor
				dockerInfo.BorderStyle.Fg = theme.BorderColor
				cpuCores.BorderStyle.Fg = theme.BorderColor
				cpuCores.BarColors = theme.BarColors
				cpuCores.LabelStyles = []ui.Style{ui.NewStyle(theme.LabelColor)}
				render()
			case "d":
				// Cycle through view modes: Normal → GPU → Docker → Normal
				switch viewMode {
				case ViewModeNormal:
					// Try GPU view first
					gpuInfoData := GetGPUInfo()
					if gpuInfoData.Available {
						viewMode = ViewModeGPU
						showDocker = false
						cachedGPUProcesses = GetGPUProcesses()
					} else {
						// Skip to Docker if no GPU
						viewMode = ViewModeDocker
						showDocker = true
					}
				case ViewModeGPU:
					// Go to Docker view
					viewMode = ViewModeDocker
					showDocker = true
					// Reset GPU-specific sort modes
					if sortMode == SortByGPUMem || sortMode == SortByGPUPerc {
						sortMode = SortByCPU
					}
				case ViewModeDocker:
					// Go back to Normal view
					viewMode = ViewModeNormal
					showDocker = false
					// Reset GPU-specific sort modes
					if sortMode == SortByGPUMem || sortMode == SortByGPUPerc {
						sortMode = SortByCPU
					}
				}
				// Reset scroll position when switching views
				scrollOffset = 0
				selectedRow = 1
				updateGridLayout()
				render()
			case "g":
				// Sort by GPU% (only in GPU view)
				if viewMode == ViewModeGPU {
					sortMode = SortByGPUPerc
					render()
				}
			case "G":
				// Sort by GPU Memory (only in GPU view)
				if viewMode == ViewModeGPU {
					sortMode = SortByGPUMem
					render()
				}
			}
		case <-ticker.C:
			// Force process list refresh on each tick
			cachedProcesses = nil
			if viewMode == ViewModeGPU {
				cachedGPUProcesses = GetGPUProcesses()
			}
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

// getSystemInfo returns formatted system information (compact version)
func getSystemInfo() string {
	// Get uptime
	uptimeStr := "N/A"
	if uptime, err := host.Uptime(); err == nil {
		days := uptime / 86400
		hours := (uptime % 86400) / 3600
		minutes := (uptime % 3600) / 60
		if days > 0 {
			uptimeStr = fmt.Sprintf("%dd%dh", days, hours)
		} else if hours > 0 {
			uptimeStr = fmt.Sprintf("%dh%dm", hours, minutes)
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

	return fmt.Sprintf(
		"Up: %s\nLoad: %s\nProcs: %d",
		uptimeStr, loadStr, procCount,
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

		// Get parent PID
		if ppid, err := proc.PpidWithContext(ctx); err == nil {
			p.PPID = ppid
		}

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

// renderHelp displays a help overlay with keyboard shortcuts
func renderHelp(termWidth, termHeight int) {
	helpText := ` Konrul - Help

 Function Keys:
   F1         Show this help
   F8         Cycle sort mode
   F9         Kill process / Stop container
   F10        Quit application

 Navigation:
   ↑/k        Scroll up
   ↓/j        Scroll down
   Home       Jump to top
   End        Jump to bottom

 Process Management:
   K/Delete   Kill selected process

 Sorting (Normal View):
   c          Sort by CPU usage
   m          Sort by Memory usage
   p          Sort by PID
   n          Sort by Name
   r          Reverse sort order

 Sorting (GPU View):
   g          Sort by GPU%
   G          Sort by GPU Memory

 Views:
   d          Toggle: Normal → GPU → Docker
   t          Toggle tree view (Normal only)
   T          Cycle themes
   /          Search/filter processes
   Esc        Clear search filter

 Other:
   ?/h        Show this help
   q/Ctrl+C   Quit

 Press any key to close this help`

	// Create help paragraph
	helpPara := widgets.NewParagraph()
	helpPara.Title = " Help "
	helpPara.Text = helpText
	helpPara.BorderStyle.Fg = ui.ColorYellow
	helpPara.TitleStyle.Fg = ui.ColorYellow

	// Calculate centered position
	helpWidth := 50
	helpHeight := 44
	x := (termWidth - helpWidth) / 2
	y := (termHeight - helpHeight) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	helpPara.SetRect(x, y, x+helpWidth, y+helpHeight)
	ui.Render(helpPara)
}

// filterProcesses filters processes by search query (case-insensitive)
func filterProcesses(processes []Process, query string) []Process {
	query = strings.ToLower(query)
	var filtered []Process
	for _, p := range processes {
		// Search in Name, Command, User, and PID
		pidStr := strconv.Itoa(int(p.PID))
		if strings.Contains(strings.ToLower(p.Name), query) ||
			strings.Contains(strings.ToLower(p.Command), query) ||
			strings.Contains(strings.ToLower(p.User), query) ||
			strings.Contains(pidStr, query) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// buildProcessTree builds a tree-structured list of processes
func buildProcessTree(processes []Process) []Process {
	// Create a map for quick lookup
	processMap := make(map[int32]*Process)
	for i := range processes {
		processMap[processes[i].PID] = &processes[i]
	}

	// Find children for each process
	children := make(map[int32][]int32)
	var roots []int32
	for _, p := range processes {
		if p.PPID == 0 || processMap[p.PPID] == nil {
			roots = append(roots, p.PID)
		} else {
			children[p.PPID] = append(children[p.PPID], p.PID)
		}
	}

	// Sort roots by PID
	sort.Slice(roots, func(i, j int) bool {
		return roots[i] < roots[j]
	})

	// Build the tree recursively
	var result []Process
	var buildTree func(pid int32, depth int)
	buildTree = func(pid int32, depth int) {
		if p, ok := processMap[pid]; ok {
			p.Depth = depth
			result = append(result, *p)
			// Sort children by PID
			childPids := children[pid]
			sort.Slice(childPids, func(i, j int) bool {
				return childPids[i] < childPids[j]
			})
			for _, childPid := range childPids {
				buildTree(childPid, depth+1)
			}
		}
	}

	for _, rootPid := range roots {
		buildTree(rootPid, 0)
	}

	return result
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

// getProcessUser returns the username for a given PID
func getProcessUser(pid int32) string {
	ctx := context.Background()
	p, err := process.NewProcess(pid)
	if err != nil {
		return "?"
	}
	user, err := p.UsernameWithContext(ctx)
	if err != nil {
		return "?"
	}
	// On Windows, username might be "DOMAIN\\user", extract just the user part
	if idx := strings.LastIndex(user, "\\"); idx != -1 {
		user = user[idx+1:]
	}
	return user
}

// parsePercent parses a percentage string like "2.5%" to float64
func parsePercent(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	val, _ := strconv.ParseFloat(s, 64)
	return val
}
