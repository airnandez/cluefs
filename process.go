package main

import (
	"path/filepath"
	"sync"
)

var (
	// procStore is a map from process ids to executable paths
	procMutex sync.RWMutex
	procStore map[uint32]string
)

func init() {
	procStore = make(map[uint32]string, 1024)
}

// processPath returns the path of the executable file associated to the
// given process id. It caches the path information in a map in memory
// for satisfying subsequent requests.
// If the process name cannot be retrieved, it returns an empty string.
func processPath(pid uint32) string {
	procMutex.RLock()
	path, ok := procStore[pid]
	procMutex.RUnlock()
	if !ok {
		path = osProcessPath(pid)
		procMutex.Lock()
		procStore[pid] = path
		procMutex.Unlock()
	}
	return path
}

// processName returns the name of the executable file associated to the
// given process id.
func processName(pid uint32) string {
	return filepath.Base(processPath(pid))
}
