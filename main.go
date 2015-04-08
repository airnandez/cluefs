package main

import (
	"os"
)

func main() {
	// Parse command line arguments
	conf, err := ParseArguments()
	if err != nil {
		os.Exit(1)
	}

	// Create the tracer
	tracer, err := NewTracer(conf.GetOutputFormat(), conf.GetTraceDestination())
	if err != nil {
		errlog.Printf("%s", err)
		os.Exit(2)
	}

	// Create the file system object
	cfs, err := NewClueFS(conf.GetShadowDir(), tracer)
	if err != nil {
		errlog.Printf("could not create file system [%s]", err)
		os.Exit(2)
	}

	// Mount and serve file system requests
	if err = cfs.MountAndServe(conf.GetMountPoint(), conf.GetReadOnly()); err != nil {
		errlog.Printf("could not mount file system [%s]", err)
		os.Exit(3)
	}

	// We are done
	os.Exit(0)
}
