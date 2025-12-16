// Konrul - GPU Monitoring Support
// NVIDIA GPU monitoring via nvidia-smi
// AMD GPU monitoring via rocm-smi

package main

import (
	"os/exec"
	"strconv"
	"strings"
)

// GPUVendor represents the GPU vendor
type GPUVendor int

const (
	GPUVendorNone GPUVendor = iota
	GPUVendorNVIDIA
	GPUVendorAMD
)

// GPUInfo holds GPU information
type GPUInfo struct {
	Available     bool
	Vendor        GPUVendor
	Name          string
	Utilization   int     // GPU utilization %
	MemoryUsed    uint64  // bytes
	MemoryTotal   uint64  // bytes
	MemoryPercent float64
	Temperature   int     // Celsius
	FanSpeed      int     // %
	PowerUsage    float64 // Watts
	PowerLimit    float64 // Watts
}

// GPUProcess holds information about a process using GPU
type GPUProcess struct {
	PID       int32
	Name      string
	GPUMemory uint64  // bytes
	GPUPercent float64 // SM utilization % (NVIDIA only)
	Type      string  // C=Compute, G=Graphics, C+G=Both
	Vendor    GPUVendor
}

// GetGPUProcesses returns list of processes using GPU
func GetGPUProcesses() []GPUProcess {
	// Try NVIDIA first
	procs := getNVIDIAGPUProcesses()
	if len(procs) > 0 {
		return procs
	}

	// Try AMD
	procs = getAMDGPUProcesses()
	if len(procs) > 0 {
		return procs
	}

	return nil
}

// getNVIDIAGPUProcesses returns processes using NVIDIA GPU
func getNVIDIAGPUProcesses() []GPUProcess {
	var processes []GPUProcess

	// Method 1: nvidia-smi --query-compute-apps (gets compute processes)
	cmd := exec.Command("nvidia-smi",
		"--query-compute-apps=pid,process_name,used_memory",
		"--format=csv,noheader,nounits")

	output, err := cmd.Output()
	if err != nil {
		return processes
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ", ")
		if len(parts) < 3 {
			parts = strings.Split(line, ",")
		}
		if len(parts) < 3 {
			continue
		}

		pid, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 32)
		if err != nil {
			continue
		}

		name := strings.TrimSpace(parts[1])
		
		memMB, _ := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		memBytes := uint64(memMB * 1024 * 1024)

		processes = append(processes, GPUProcess{
			PID:       int32(pid),
			Name:      name,
			GPUMemory: memBytes,
			Type:      "G", // Graphics (most Windows apps use GPU for rendering)
			Vendor:    GPUVendorNVIDIA,
		})
	}

	// Method 2: nvidia-smi pmon for GPU utilization (optional enhancement)
	// This gives per-process GPU % but is more complex to parse
	pmonCmd := exec.Command("nvidia-smi", "pmon", "-c", "1", "-s", "u")
	pmonOutput, err := pmonCmd.Output()
	if err == nil {
		pmonLines := strings.Split(string(pmonOutput), "\n")
		for _, line := range pmonLines {
			// Skip header lines (start with #)
			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}
			
			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}
			
			// Fields: gpu, pid, type, sm%, mem%, enc, dec, jpg, ofa, command
			pid, err := strconv.ParseInt(fields[1], 10, 32)
			if err != nil || pid == 0 {
				continue
			}
			
			// Find matching process and update GPU%
			smPercent := 0.0
			if len(fields) > 3 && fields[3] != "-" {
				smPercent, _ = strconv.ParseFloat(fields[3], 64)
			}
			
			procType := "C"
			if len(fields) > 2 {
				procType = fields[2] // C, G, or C+G
			}
			
			// Update existing or add new
			found := false
			for i := range processes {
				if processes[i].PID == int32(pid) {
					processes[i].GPUPercent = smPercent
					processes[i].Type = procType
					found = true
					break
				}
			}
			
			if !found && smPercent > 0 {
				// Get process name from command field
				name := "unknown"
				if len(fields) > 9 {
					name = fields[9]
				}
				processes = append(processes, GPUProcess{
					PID:        int32(pid),
					Name:       name,
					GPUPercent: smPercent,
					Type:       procType,
					Vendor:     GPUVendorNVIDIA,
				})
			}
		}
	}

	return processes
}

// getAMDGPUProcesses returns processes using AMD GPU
func getAMDGPUProcesses() []GPUProcess {
	var processes []GPUProcess

	// rocm-smi --showpidgpus
	cmd := exec.Command("rocm-smi", "--showpidgpus")
	output, err := cmd.Output()
	if err != nil {
		return processes
	}

	// Parse output format:
	// GPU[0] : PID 1234 is using 2048 bytes
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "PID") {
			continue
		}

		// Extract PID
		pidIdx := strings.Index(line, "PID")
		if pidIdx == -1 {
			continue
		}
		
		// Find the number after "PID "
		remaining := line[pidIdx+3:]
		remaining = strings.TrimSpace(remaining)
		
		fields := strings.Fields(remaining)
		if len(fields) < 1 {
			continue
		}
		
		pid, err := strconv.ParseInt(fields[0], 10, 32)
		if err != nil {
			continue
		}

		// Try to extract memory usage
		var memBytes uint64
		if strings.Contains(line, "using") {
			usingIdx := strings.Index(line, "using")
			if usingIdx != -1 {
				memPart := strings.TrimSpace(line[usingIdx+5:])
				memFields := strings.Fields(memPart)
				if len(memFields) >= 1 {
					mem, _ := strconv.ParseUint(memFields[0], 10, 64)
					// Check unit
					if len(memFields) >= 2 {
						unit := strings.ToLower(memFields[1])
						if strings.HasPrefix(unit, "mb") || strings.HasPrefix(unit, "mib") {
							mem = mem * 1024 * 1024
						} else if strings.HasPrefix(unit, "gb") || strings.HasPrefix(unit, "gib") {
							mem = mem * 1024 * 1024 * 1024
						} else if strings.HasPrefix(unit, "kb") || strings.HasPrefix(unit, "kib") {
							mem = mem * 1024
						}
						// else assume bytes
					}
					memBytes = mem
				}
			}
		}

		// Get process name
		name := getProcessName(int32(pid))

		processes = append(processes, GPUProcess{
			PID:       int32(pid),
			Name:      name,
			GPUMemory: memBytes,
			Vendor:    GPUVendorAMD,
		})
	}

	return processes
}

// getProcessName returns the name of a process by PID
func getProcessName(pid int32) string {
	// Try to get from /proc on Linux
	// For cross-platform, we could use gopsutil but keep it simple
	cmd := exec.Command("ps", "-p", strconv.Itoa(int(pid)), "-o", "comm=")
	output, err := cmd.Output()
	if err == nil {
		name := strings.TrimSpace(string(output))
		if name != "" {
			return name
		}
	}
	return "unknown"
}

// GetGPUInfo returns GPU information (tries NVIDIA first, then AMD)
func GetGPUInfo() GPUInfo {
	// Try NVIDIA first
	info := getNVIDIAGPUInfo()
	if info.Available {
		return info
	}

	// Try AMD
	info = getAMDGPUInfo()
	if info.Available {
		return info
	}

	return GPUInfo{Available: false}
}

// getNVIDIAGPUInfo returns NVIDIA GPU information using nvidia-smi
func getNVIDIAGPUInfo() GPUInfo {
	info := GPUInfo{Available: false, Vendor: GPUVendorNVIDIA}

	// Try to run nvidia-smi
	// Query: gpu_name, utilization.gpu, memory.used, memory.total, temperature.gpu, fan.speed, power.draw, power.limit
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=gpu_name,utilization.gpu,memory.used,memory.total,temperature.gpu,fan.speed,power.draw,power.limit",
		"--format=csv,noheader,nounits")

	output, err := cmd.Output()
	if err != nil {
		return info
	}

	// Parse output
	line := strings.TrimSpace(string(output))
	if line == "" {
		return info
	}

	// Split by comma
	parts := strings.Split(line, ", ")
	if len(parts) < 8 {
		// Try splitting with just comma (no space)
		parts = strings.Split(line, ",")
	}
	
	if len(parts) < 4 {
		return info
	}

	info.Available = true

	// GPU Name
	info.Name = strings.TrimSpace(parts[0])

	// GPU Utilization %
	if len(parts) > 1 {
		if val, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
			info.Utilization = val
		}
	}

	// Memory Used (MiB -> bytes)
	if len(parts) > 2 {
		if val, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64); err == nil {
			info.MemoryUsed = uint64(val * 1024 * 1024)
		}
	}

	// Memory Total (MiB -> bytes)
	if len(parts) > 3 {
		if val, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64); err == nil {
			info.MemoryTotal = uint64(val * 1024 * 1024)
		}
	}

	// Calculate memory percent
	if info.MemoryTotal > 0 {
		info.MemoryPercent = float64(info.MemoryUsed) / float64(info.MemoryTotal) * 100
	}

	// Temperature
	if len(parts) > 4 {
		if val, err := strconv.Atoi(strings.TrimSpace(parts[4])); err == nil {
			info.Temperature = val
		}
	}

	// Fan Speed
	if len(parts) > 5 {
		valStr := strings.TrimSpace(parts[5])
		// Fan speed might be "[N/A]" on some GPUs (laptops)
		if !strings.Contains(valStr, "N/A") {
			if val, err := strconv.Atoi(valStr); err == nil {
				info.FanSpeed = val
			}
		}
	}

	// Power Usage
	if len(parts) > 6 {
		valStr := strings.TrimSpace(parts[6])
		if !strings.Contains(valStr, "N/A") {
			if val, err := strconv.ParseFloat(valStr, 64); err == nil {
				info.PowerUsage = val
			}
		}
	}

	// Power Limit
	if len(parts) > 7 {
		valStr := strings.TrimSpace(parts[7])
		if !strings.Contains(valStr, "N/A") {
			if val, err := strconv.ParseFloat(valStr, 64); err == nil {
				info.PowerLimit = val
			}
		}
	}

	return info
}

// getAMDGPUInfo returns AMD GPU information using rocm-smi
func getAMDGPUInfo() GPUInfo {
	info := GPUInfo{Available: false, Vendor: GPUVendorAMD}

	// Try to run rocm-smi to check if AMD GPU is available
	// First, get GPU name
	cmdName := exec.Command("rocm-smi", "--showproductname")
	nameOutput, err := cmdName.Output()
	if err != nil {
		return info
	}

	// Parse GPU name from output
	// Output format: "GPU[0]		: Card series:		AMD Radeon RX 7900 XTX"
	nameLines := strings.Split(string(nameOutput), "\n")
	for _, line := range nameLines {
		if strings.Contains(line, "Card series") || strings.Contains(line, "Card model") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				info.Name = strings.TrimSpace(parts[len(parts)-1])
				break
			}
		}
	}

	// If no name found, try alternative format
	if info.Name == "" {
		for _, line := range nameLines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "GPU[") && strings.Contains(line, ":") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) >= 2 {
					info.Name = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}

	if info.Name == "" {
		info.Name = "AMD GPU"
	}

	info.Available = true

	// Get GPU utilization
	cmdUtil := exec.Command("rocm-smi", "--showuse")
	utilOutput, err := cmdUtil.Output()
	if err == nil {
		// Parse: "GPU[0]		: GPU use (%):			5"
		for _, line := range strings.Split(string(utilOutput), "\n") {
			if strings.Contains(line, "GPU use") || strings.Contains(line, "GPU activity") {
				// Extract the number at the end
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					valStr := strings.TrimSpace(parts[len(parts)-1])
					valStr = strings.TrimSuffix(valStr, "%")
					if val, err := strconv.Atoi(strings.TrimSpace(valStr)); err == nil {
						info.Utilization = val
						break
					}
				}
			}
		}
	}

	// Get memory info
	cmdMem := exec.Command("rocm-smi", "--showmeminfo", "vram")
	memOutput, err := cmdMem.Output()
	if err == nil {
		// Parse VRAM info
		// "GPU[0]		: VRAM Total Memory (B):	25753026560"
		// "GPU[0]		: VRAM Total Used Memory (B):	1048576"
		for _, line := range strings.Split(string(memOutput), "\n") {
			if strings.Contains(line, "Total Memory") && !strings.Contains(line, "Used") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					valStr := strings.TrimSpace(parts[len(parts)-1])
					if val, err := strconv.ParseUint(valStr, 10, 64); err == nil {
						info.MemoryTotal = val
					}
				}
			} else if strings.Contains(line, "Used Memory") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					valStr := strings.TrimSpace(parts[len(parts)-1])
					if val, err := strconv.ParseUint(valStr, 10, 64); err == nil {
						info.MemoryUsed = val
					}
				}
			}
		}
	}

	// Calculate memory percent
	if info.MemoryTotal > 0 {
		info.MemoryPercent = float64(info.MemoryUsed) / float64(info.MemoryTotal) * 100
	}

	// Get temperature
	cmdTemp := exec.Command("rocm-smi", "--showtemp")
	tempOutput, err := cmdTemp.Output()
	if err == nil {
		// Parse: "GPU[0]		: Temperature (Sensor edge) (C):	45.0"
		for _, line := range strings.Split(string(tempOutput), "\n") {
			if strings.Contains(line, "Temperature") && (strings.Contains(line, "edge") || strings.Contains(line, "junction")) {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					valStr := strings.TrimSpace(parts[len(parts)-1])
					if val, err := strconv.ParseFloat(valStr, 64); err == nil {
						info.Temperature = int(val)
						break
					}
				}
			}
		}
	}

	// Get fan speed
	cmdFan := exec.Command("rocm-smi", "--showfan")
	fanOutput, err := cmdFan.Output()
	if err == nil {
		// Parse: "GPU[0]		: Fan speed (%):		0"
		for _, line := range strings.Split(string(fanOutput), "\n") {
			if strings.Contains(line, "Fan") && strings.Contains(line, "%") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					valStr := strings.TrimSpace(parts[len(parts)-1])
					if val, err := strconv.Atoi(valStr); err == nil {
						info.FanSpeed = val
						break
					}
				}
			}
		}
	}

	// Get power usage
	cmdPower := exec.Command("rocm-smi", "--showpower")
	powerOutput, err := cmdPower.Output()
	if err == nil {
		// Parse: "GPU[0]		: Average Graphics Package Power (W):	25.0"
		for _, line := range strings.Split(string(powerOutput), "\n") {
			if strings.Contains(line, "Power") && strings.Contains(line, "W") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					valStr := strings.TrimSpace(parts[len(parts)-1])
					if val, err := strconv.ParseFloat(valStr, 64); err == nil {
						info.PowerUsage = val
						break
					}
				}
			}
		}
	}

	return info
}

// FormatGPUInfo returns formatted GPU information string
func FormatGPUInfo() string {
	info := GetGPUInfo()

	if !info.Available {
		return "No GPU detected"
	}

	// Truncate GPU name if too long
	name := info.Name
	if len(name) > 18 {
		name = name[:15] + "..."
	}

	// Add vendor indicator
	vendorStr := ""
	switch info.Vendor {
	case GPUVendorNVIDIA:
		vendorStr = "[NV] "
	case GPUVendorAMD:
		vendorStr = "[AMD] "
	}

	result := vendorStr + name + "\n"
	result += "GPU: " + strconv.Itoa(info.Utilization) + "%\n"
	result += "Mem: " + formatBytes(info.MemoryUsed) + "/" + formatBytes(info.MemoryTotal) + "\n"
	
	if info.Temperature > 0 {
		result += "Temp: " + strconv.Itoa(info.Temperature) + "°C"
	}

	return result
}
