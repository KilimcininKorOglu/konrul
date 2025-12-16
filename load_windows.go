//go:build windows
// +build windows

package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modPdh                       = windows.NewLazySystemDLL("pdh.dll")
	procPdhOpenQuery             = modPdh.NewProc("PdhOpenQueryW")
	procPdhAddCounterW           = modPdh.NewProc("PdhAddCounterW")
	procPdhCollectQueryData      = modPdh.NewProc("PdhCollectQueryData")
	procPdhGetFormattedCounterValue = modPdh.NewProc("PdhGetFormattedCounterValue")
	procPdhCloseQuery            = modPdh.NewProc("PdhCloseQuery")
)

const (
	PDH_FMT_LONG   = 0x00000100
	PDH_FMT_DOUBLE = 0x00000200
)

type PDH_FMT_COUNTERVALUE_DOUBLE struct {
	CStatus     uint32
	_           [4]byte
	DoubleValue float64
}

// getLoadInfo returns CPU Queue Length for Windows
// This is the Windows equivalent of Unix load average
func getLoadInfo() string {
	queueLen, err := getCPUQueueLength()
	if err != nil {
		// Fallback: calculate based on running processes and CPU count
		return "N/A (Queue)"
	}
	return fmt.Sprintf("Queue: %.0f", queueLen)
}

// getCPUQueueLength retrieves the Processor Queue Length from Windows Performance Counters
func getCPUQueueLength() (float64, error) {
	var queryHandle uintptr
	var counterHandle uintptr

	// Open query
	ret, _, _ := procPdhOpenQuery.Call(0, 0, uintptr(unsafe.Pointer(&queryHandle)))
	if ret != 0 {
		return 0, fmt.Errorf("PdhOpenQuery failed: %x", ret)
	}
	defer procPdhCloseQuery.Call(queryHandle)

	// Add counter for Processor Queue Length
	counterPath, _ := windows.UTF16PtrFromString("\\System\\Processor Queue Length")
	ret, _, _ = procPdhAddCounterW.Call(
		queryHandle,
		uintptr(unsafe.Pointer(counterPath)),
		0,
		uintptr(unsafe.Pointer(&counterHandle)),
	)
	if ret != 0 {
		return 0, fmt.Errorf("PdhAddCounterW failed: %x", ret)
	}

	// Collect data
	ret, _, _ = procPdhCollectQueryData.Call(queryHandle)
	if ret != 0 {
		return 0, fmt.Errorf("PdhCollectQueryData failed: %x", ret)
	}

	// Get formatted value
	var counterValue PDH_FMT_COUNTERVALUE_DOUBLE
	ret, _, _ = procPdhGetFormattedCounterValue.Call(
		counterHandle,
		PDH_FMT_DOUBLE,
		0,
		uintptr(unsafe.Pointer(&counterValue)),
	)
	if ret != 0 {
		return 0, fmt.Errorf("PdhGetFormattedCounterValue failed: %x", ret)
	}

	return counterValue.DoubleValue, nil
}
