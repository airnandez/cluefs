package cluefs

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var (
	programName string
	errlog      *log.Logger
	// Log destination for errors in main()
	ErrlogMain  *log.Logger

	// The two variables below are set at build time for official releases
	// (see Makefile)
	version   = "unknown"
	buildTime = "unknown"
)

func init() {
	programName = filepath.Base(os.Args[0])
	errlog = log.New(os.Stderr, fmt.Sprintf("%s: ", programName), 0)
	ErrlogMain = errlog
}
