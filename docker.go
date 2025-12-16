// Konrul - Docker Container Monitoring Support
// Docker monitoring via docker CLI

package main

import (
	"encoding/json"
	"os/exec"
	"strings"
)

// DockerContainer holds container information
type DockerContainer struct {
	ID      string
	Name    string
	Image   string
	Status  string
	State   string
	CPUPerc string
	MemPerc string
	MemUsage string
}

// DockerInfo holds Docker daemon information
type DockerInfo struct {
	Available  bool
	Containers []DockerContainer
	Running    int
	Stopped    int
	Total      int
}

// dockerStatsJSON represents the JSON output from docker stats
type dockerStatsJSON struct {
	ID       string `json:"ID"`
	Name     string `json:"Name"`
	CPUPerc  string `json:"CPUPerc"`
	MemPerc  string `json:"MemPerc"`
	MemUsage string `json:"MemUsage"`
}

// dockerPsJSON represents the JSON output from docker ps
type dockerPsJSON struct {
	ID     string `json:"ID"`
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	Status string `json:"Status"`
	State  string `json:"State"`
}

// GetDockerInfo returns Docker container information
func GetDockerInfo() DockerInfo {
	info := DockerInfo{Available: false}

	// Check if Docker is available
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	if err := cmd.Run(); err != nil {
		return info
	}

	info.Available = true

	// Get container list with stats
	containers := getDockerContainers()
	info.Containers = containers

	// Count running/stopped
	for _, c := range containers {
		info.Total++
		if c.State == "running" {
			info.Running++
		} else {
			info.Stopped++
		}
	}

	return info
}

// getDockerContainers returns list of containers with their stats
func getDockerContainers() []DockerContainer {
	var containers []DockerContainer

	// Get container list
	cmd := exec.Command("docker", "ps", "-a", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		return containers
	}

	// Parse each line as JSON
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	containerMap := make(map[string]*DockerContainer)
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		var ps dockerPsJSON
		if err := json.Unmarshal([]byte(line), &ps); err != nil {
			continue
		}
		container := &DockerContainer{
			ID:     ps.ID[:12], // Short ID
			Name:   strings.TrimPrefix(ps.Names, "/"),
			Image:  ps.Image,
			Status: ps.Status,
			State:  ps.State,
		}
		containers = append(containers, *container)
		containerMap[container.ID] = container
	}

	// Get stats for running containers (non-blocking)
	statsCmd := exec.Command("docker", "stats", "--no-stream", "--format", "{{json .}}")
	statsOutput, err := statsCmd.Output()
	if err == nil {
		statsLines := strings.Split(strings.TrimSpace(string(statsOutput)), "\n")
		for _, line := range statsLines {
			if line == "" {
				continue
			}
			var stats dockerStatsJSON
			if err := json.Unmarshal([]byte(line), &stats); err != nil {
				continue
			}
			// Find container and update stats
			shortID := stats.ID
			if len(shortID) > 12 {
				shortID = shortID[:12]
			}
			for i := range containers {
				if containers[i].ID == shortID || strings.HasPrefix(containers[i].ID, shortID) {
					containers[i].CPUPerc = stats.CPUPerc
					containers[i].MemPerc = stats.MemPerc
					containers[i].MemUsage = stats.MemUsage
					break
				}
			}
		}
	}

	return containers
}

// FormatDockerInfo returns formatted Docker information string
func FormatDockerInfo() string {
	info := GetDockerInfo()

	if !info.Available {
		return "Docker N/A"
	}

	if info.Total == 0 {
		return "No containers"
	}

	result := ""
	result += "Run: " + intToStr(info.Running) + " Stop: " + intToStr(info.Stopped) + "\n"
	
	// Show top 3 running containers
	count := 0
	for _, c := range info.Containers {
		if c.State == "running" && count < 2 {
			name := c.Name
			if len(name) > 10 {
				name = name[:10]
			}
			cpu := c.CPUPerc
			if cpu == "" {
				cpu = "0%"
			}
			result += name + " " + cpu + "\n"
			count++
		}
	}

	return strings.TrimSuffix(result, "\n")
}

// intToStr converts int to string
func intToStr(i int) string {
	if i == 0 {
		return "0"
	}
	result := ""
	for i > 0 {
		result = string(rune('0'+i%10)) + result
		i /= 10
	}
	return result
}
