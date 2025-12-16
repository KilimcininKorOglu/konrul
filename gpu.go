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
