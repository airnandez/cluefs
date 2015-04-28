package main

import (
	"path/filepath"
)

// processPath returns the full path of the executable file associated to the
// given process id. If the process name cannot be retrieved, it returns an
// empty string.
// Note that we purposedly don't cache the result, since
// in situations when a process does a fork/execv to create a child
// process (as for instance /bin/bash does) we would erroneously return the
// path of the parent process.
func processPath(pid uint32) string {
	return osProcessPath(pid)
}

// processName returns the name of the executable file associated to the
// given process id.
func processName(pid uint32) string {
	return filepath.Base(osProcessPath(pid))
}
