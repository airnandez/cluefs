package main

import (
	"os"
	"github.com/airnandez/cluefs/lib/cluefs"
)

func main() {
	// Parse command line arguments
	conf, err := cluefs.ParseArguments()
	if err != nil {
		os.Exit(1)
	}

	// Create the tracer
	tracer, err := cluefs.NewTracer(conf.GetOutputFormat(), conf.GetTraceDestination())
	if err != nil {
		cluefs.ErrlogMain.Printf("%s", err)
		os.Exit(2)
	}

	// Create the file system object
	cfs, err := cluefs.NewClueFS(conf.GetShadowDir(), tracer)
	if err != nil {
		cluefs.ErrlogMain.Printf("could not create file system [%s]", err)
		os.Exit(2)
	}

	// Mount and serve file system requests
	if err = cfs.MountAndServe(conf.GetMountPoint(), conf.GetReadOnly()); err != nil {
		cluefs.ErrlogMain.Printf("could not mount file system [%s]", err)
		os.Exit(3)
	}

	// We are done
	os.Exit(0)
}
