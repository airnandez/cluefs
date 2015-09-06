package cluefs

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
	"path"
	"runtime"
	"strconv"

	// #include <stdio.h>
	// #include <unistd.h>
	// #include <string.h>
	"C"
)

// Create logger object
const (
	logFlags = log.Ldate | log.Ltime // | log.Lshortfile
)

var (
	debugLogger  = log.New(os.Stderr, "DEBUG ", logFlags)
	debugLevel   = 0
	prefixFormat = ""
	formatMap    = map[bool]string{
		true:  "\033[33mDEBUG L%d [%s:%d]\033[0m \t",
		false: "DEBUG L%d [%s:%d] \t",
	}
)

func init() {
	// Determine if output format to use according to the destination of the
	// debug messages. If stderr is a terminal we can use colors
	prefixFormat = formatMap[isTerminal(os.Stderr)]

	// Set the debug level from the environmental variable
	if env := os.Getenv("CLUEFS_DEBUG"); env != "" {
		if value, err := strconv.ParseInt(env, 10, 32); err == nil {
			debugLevel = clamp(int(value), 0, 5)
		}
	}
}

// Set the debug level
func SetLevel(level int) {
	debugLevel = level
}

func IsDebugActive() bool {
	return debugLevel > 0
}

// Show a debug message
func Debug(level int, format string, v ...interface{}) {
	if debugLevel > 0 && level <= debugLevel {
		_, file, line, _ := runtime.Caller(1)
		debugLogger.SetPrefix(fmt.Sprintf(prefixFormat, level, path.Base(file), line))
		debugLogger.Printf(format, v...)
	}
}

// Send debug message to syslog facility
func ToSyslog() {
	syslogger, err := syslog.NewLogger(syslog.LOG_ERR|syslog.LOG_USER, 0)
	if err == nil {
		debugLogger = syslogger
		prefixFormat = formatMap[false]
	}
}

func isTerminal(file *os.File) bool {
	if C.isatty(C.int(file.Fd())) == 0 {
		return false
	}
	return true
}

// Returns the minimum value among two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Returns the maximum value among two integers
func maxInt(a, b int) int {
	if a < b {
		return b
	}
	return a
}

// Keeps a given value within the specified interval
func clamp(val, min, max int) int {
	return minInt(maxInt(min, val), max)
}
