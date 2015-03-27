package main

import (
	"fmt"
	"os"
	"strings"
)

func osProcessPath(pid uint32) string {
	// Try the /proc/<pid>/exe file
	path, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err == nil {
		// Check that this is actually a file path. On Linux v2.0 or
		// earlier, this file contains a string of the form "[device]:inode"
		if strings.HasPrefix(path, "/") {
			return path
		}
	}

	// Try the /proc/<pid>/cmdline file
	f, err := os.Open(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}
	defer f.Close()

	// Read the contents of the file and Scan until finding the first
	// delimiter, i.e. '\0', the marker of the end of a C string.
	// See 'man 5 proc' for details.
	buffer := make([]byte, 4096)
	n, err := f.Read(buffer)
	if err != nil || n == 0 {
		return ""
	}
	for pos, b := range buffer[0:n] {
		if b == byte(0) {
			return string(buffer[0:pos])
		}
	}
	// Could not find delimiter. Assume all the contents of the buffer
	// contains a truncated executable path.
	return string(buffer[0:n])
}
