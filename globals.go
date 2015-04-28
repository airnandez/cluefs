package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const (
	version = "0.2"
)

var (
	programName string
	errlog      *log.Logger
)

func init() {
	programName = filepath.Base(os.Args[0])
	errlog = log.New(os.Stderr, fmt.Sprintf("%s: ", programName), 0)
}
