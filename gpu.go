// Konrul - GPU Monitoring Support
// NVIDIA GPU monitoring via nvidia-smi

package main

import (
	"os/exec"
	"strconv"
	"strings"
)

// GPUInfo holds GPU information
type GPUInfo struct {
	Available   bool
	Name        string
	Utilization int     // GPU utilization %
	MemoryUsed  uint64  // bytes
	MemoryTotal uint64  // bytes
	MemoryPercent float64
	Temperature int     // Celsius
	FanSpeed    int     // %
	PowerUsage  float64 // Watts
	PowerLimit  float64 // Watts
}

// GetGPUInfo returns NVIDIA GPU information using nvidia-smi
func GetGPUInfo() GPUInfo {
	info := GPUInfo{Available: false}

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

// FormatGPUInfo returns formatted GPU information string
func FormatGPUInfo() string {
	info := GetGPUInfo()

	if !info.Available {
		return "No NVIDIA GPU"
	}

	// Truncate GPU name if too long
	name := info.Name
	if len(name) > 20 {
		name = name[:17] + "..."
	}

	result := name + "\n"
	result += "GPU: " + strconv.Itoa(info.Utilization) + "%\n"
	result += "Mem: " + formatBytes(info.MemoryUsed) + "/" + formatBytes(info.MemoryTotal) + "\n"
	
	if info.Temperature > 0 {
		result += "Temp: " + strconv.Itoa(info.Temperature) + "°C"
	}

	return result
}
