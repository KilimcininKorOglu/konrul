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
	ID        string
	Name      string
	Image     string
	Status    string
	State     string
	CPUPerc   string
	MemPerc   string
	MemUsage  string
	Ports     string // "0.0.0.0:8080->80/tcp, :::443->443/tcp"
	IPAddress string // Container IP (172.17.0.2)
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
	Ports  string `json:"Ports"`
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
		shortID := ps.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		container := &DockerContainer{
			ID:     shortID,
			Name:   strings.TrimPrefix(ps.Names, "/"),
			Image:  ps.Image,
			Status: ps.Status,
			State:  ps.State,
			Ports:  formatPorts(ps.Ports),
		}
		containers = append(containers, *container)
		containerMap[container.ID] = container
	}

	// Get IP addresses for running containers
	for i := range containers {
		if containers[i].State == "running" {
			containers[i].IPAddress = getContainerIP(containers[i].ID)
		}
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
	
	// Show top 2 running containers with port/IP info
	count := 0
	for _, c := range info.Containers {
		if c.State == "running" && count < 2 {
			name := c.Name
			if len(name) > 8 {
				name = name[:8]
			}
			
			// Format: name IP:port
			line := name
			
			// Add port info if available (compact)
			if c.Ports != "" {
				ports := c.Ports
				// Truncate if too long
				if len(ports) > 12 {
					ports = ports[:12]
				}
				line += " " + ports
			} else if c.IPAddress != "" {
				// Show IP if no ports
				line += " " + c.IPAddress
			}
			
			result += line + "\n"
			count++
		}
	}

	return strings.TrimSuffix(result, "\n")
}

// FormatDockerInfoDetailed returns detailed Docker info with ports and IPs
func FormatDockerInfoDetailed() string {
	info := GetDockerInfo()

	if !info.Available {
		return "Docker N/A"
	}

	if info.Total == 0 {
		return "No containers"
	}

	result := ""
	result += "Containers: " + intToStr(info.Running) + "/" + intToStr(info.Total) + "\n"
	
	// Show all running containers
	for _, c := range info.Containers {
		if c.State == "running" {
			name := c.Name
			if len(name) > 12 {
				name = name[:12]
			}
			
			line := name
			if c.IPAddress != "" {
				line += " [" + c.IPAddress + "]"
			}
			if c.Ports != "" {
				line += " " + c.Ports
			}
			
			result += line + "\n"
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

// formatPorts converts Docker port format to compact format
// Input: "0.0.0.0:8080->80/tcp, :::443->443/tcp"
// Output: "8080:80, 443:443"
func formatPorts(ports string) string {
	if ports == "" {
		return ""
	}

	var result []string
	// Split by comma
	portMappings := strings.Split(ports, ", ")
	
	for _, mapping := range portMappings {
		mapping = strings.TrimSpace(mapping)
		if mapping == "" {
			continue
		}
		
		// Parse format: "0.0.0.0:8080->80/tcp" or ":::443->443/tcp"
		// Extract host port and container port
		
		// Find "->" to split host and container
		arrowIdx := strings.Index(mapping, "->")
		if arrowIdx == -1 {
			continue
		}
		
		hostPart := mapping[:arrowIdx]
		containerPart := mapping[arrowIdx+2:]
		
		// Extract host port (last part after :)
		hostPort := ""
		lastColon := strings.LastIndex(hostPart, ":")
		if lastColon != -1 {
			hostPort = hostPart[lastColon+1:]
		}
		
		// Extract container port (before /)
		containerPort := containerPart
		slashIdx := strings.Index(containerPort, "/")
		if slashIdx != -1 {
			containerPort = containerPort[:slashIdx]
		}
		
		if hostPort != "" && containerPort != "" {
			result = append(result, hostPort+":"+containerPort)
		}
	}
	
	return strings.Join(result, ", ")
}

// getContainerIP returns the IP address of a container
func getContainerIP(containerID string) string {
	// Use docker inspect to get IP address
	cmd := exec.Command("docker", "inspect", 
		"--format", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", 
		containerID)
	
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	
	ip := strings.TrimSpace(string(output))
	
	// If multiple IPs (multiple networks), just return first one
	if strings.Contains(ip, "\n") {
		parts := strings.Split(ip, "\n")
		if len(parts) > 0 {
			ip = parts[0]
		}
	}
	
	return ip
}
