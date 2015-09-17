package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var (
	programName string
	errlog      *log.Logger

	// The two variables below are set at build time for official releases
	// (see Makefile)
	version   = "dev"
	buildTime = "unknown"
)

func init() {
	programName = filepath.Base(os.Args[0])
	errlog = log.New(os.Stderr, fmt.Sprintf("%s: ", programName), 0)
}
