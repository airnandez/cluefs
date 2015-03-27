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
	var tracer Tracer
	destFile := conf.GetTraceDestination()
	switch conf.GetOutputFormat() {
	case "json":
		tracer, err = NewJSONTracer(destFile)

	default:
		fallthrough
	case "csv":
		tracer, err = NewCSVTracer(destFile)
	}
	if err != nil {
		errlog.Printf("%s", err)
		os.Exit(2)
	}

	// Create the file system object
	tfs, err := NewClueFS(conf.GetShadowDir(), tracer)
	if err != nil {
		errlog.Printf("could not create file system [%s]", err)
		os.Exit(2)
	}

	// Mount and serve file system requests
	if err = tfs.MountAndServe(conf.GetMountPoint(), conf.GetReadOnly()); err != nil {
		errlog.Printf("could not mount file system [%s]", err)
		os.Exit(3)
	}

	// We are done
	os.Exit(0)
}
