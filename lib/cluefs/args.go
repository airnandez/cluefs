package cluefs

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
	"text/template"
)

type HelpType uint32

const (
	HelpShort HelpType = iota
	HelpLong
)

func helpRequested() (bool, HelpType) {
	if len(os.Args) == 1 {
		return true, HelpShort
	}
	helpKeywords := map[string]bool{
		"--help": true,
		"-help":  true,
		"-h":     true,
		"help":   true,
	}
	return helpKeywords[strings.ToLower(os.Args[1])], HelpLong
}

func versionInfoRequested() bool {
	if len(os.Args) == 1 {
		return false
	}
	versionKeywords := map[string]bool{
		"--version": true,
		"-version":  true,
		"-v":        true,
		"version":   true,
	}
	return versionKeywords[strings.ToLower(os.Args[1])]
}

func ParseArguments() (*Config, error) {
	// Did the user ask for help?
	parseErr := fmt.Errorf("")
	if ok, helpFlavor := helpRequested(); ok {
		printUsage(os.Stderr, helpFlavor)
		return nil, parseErr
	}

	// Did the user ask for version details?
	if versionInfoRequested() {
		printVersionInfo(os.Stderr)
		return nil, parseErr
	}

	// Parse command line
	flag.Usage = func() {
		printUsage(os.Stderr, HelpShort)
	}
	var (
		mount    string
		shadow   string
		outFile  string
		readOnly bool
		json     bool
		csv      bool
	)
	flag.StringVar(&mount, "mount", "", "")
	flag.StringVar(&shadow, "shadow", "", "")
	flag.StringVar(&outFile, "out", "-", "")
	flag.BoolVar(&readOnly, "ro", false, "")
	flag.BoolVar(&json, "json", false, "")
	flag.BoolVar(&csv, "csv", false, "")
	flag.Parse()
	if !flag.Parsed() {
		return nil, parseErr
	}

	// Check required arguments
	var err error
	if len(mount) == 0 {
		err = fmt.Errorf("please specify mount point with --mount option")
	} else if len(shadow) == 0 {
		err = fmt.Errorf("please specify shadow directory with --shadow option")
	}
	if err != nil {
		errlog.Println(err)
		printUsage(os.Stderr, HelpShort)
		return nil, err
	}

	// Validate arguments and save configuration
	config, err := saveConfig(mount, shadow, outFile, csv, json, readOnly)
	if err != nil {
		errlog.Println(err)
		return nil, err
	}
	return config, nil
}

func saveConfig(mountDir, shadowDir, outFile string, csv, json, readonly bool) (*Config, error) {
	// Validate mount directory
	absMount, err := validateMountPoint(mountDir)
	if err != nil {
		return nil, fmt.Errorf("'%s' is not a valid mount point [%s]", mountDir, err)
	}
	conf := NewConfig()
	conf.SetMountPoint(absMount)

	// Validate target directory
	absShadow, err := validateShadowDir(shadowDir)
	if err != nil {
		return nil, fmt.Errorf("'%s' is not a valid directory [%s]", shadowDir, err)
	}
	conf.SetShadowDir(absShadow)

	// Make sure that target directory is not under mount directory
	if strings.HasPrefix(absMount, absShadow) {
		return nil, fmt.Errorf("mount point (%s) cannot be under target directory (%s)", absMount, absShadow)
	}

	// Validate output format
	f, err := validateFormat(outFile, csv, json)
	if err != nil {
		return nil, err
	}
	conf.SetOutputFormat(f)

	// Set read only
	conf.SetReadOnly(readonly)

	// Save trace destination
	conf.SetTraceDestination(outFile)

	return conf, nil
}

func validateMountPoint(path string) (string, error) {
	abspath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("'%s' is not a valid mount point [%s]", path, err)
	}

	if err := ensureIsDir(abspath); err != nil {
		return "", err
	}

	if err := ensureDirEmpty(abspath); err != nil {
		return "", err
	}
	return abspath, nil
}

func validateShadowDir(path string) (string, error) {
	abspath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("'%s' is not a valid target directory [%s]", path, err)
	}
	if err := ensureIsDir(abspath); err != nil {
		return "", err
	}
	return abspath, nil
}

func validateFormat(outFile string, csv, json bool) (string, error) {
	if csv && json {
		return "", fmt.Errorf("only one of 'csv' or 'json' options can be specified")
	}
	if !csv && !json {
		// Infer trace format from output file extension, if any
		switch strings.ToLower(filepath.Ext(outFile)) {
		case ".json":
			return "json", nil
		case ".csv":
			fallthrough
		default:
			return "csv", nil
		}
	}
	if json {
		return "json", nil
	}
	return "csv", nil
}

func ensureIsDir(abspath string) error {
	info, err := os.Stat(abspath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory '%s' does not exist", abspath)
		}
		return fmt.Errorf("'%s' is not a valid directory [%s]", abspath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("'%s' is not a directory", abspath)
	}
	return nil
}

func ensureDirEmpty(abspath string) error {
	d, err := os.Open(abspath)
	if err != nil {
		return fmt.Errorf("could not open directory '%s' [%s]", abspath, err)
	}
	defer d.Close()
	names, err := d.Readdirnames(1)
	if err != nil && err != io.EOF {
		return fmt.Errorf("could not read directory contents for '%s' [%s]", abspath, err)
	}
	if len(names) > 0 {
		return fmt.Errorf("'%s' is not empty", abspath)
	}
	return nil
}

// printUsage prints how to use this application
func printUsage(f *os.File, kind HelpType) {
	const usageTempl = `
USAGE:
{{.Sp3}}{{.AppName}} --mount=<directory>  --shadow=<directory>  [--out=<file>]
{{.Sp3}}{{.AppNameFiller}} [(--csv | --json)]  [--ro]
{{.Sp3}}{{.AppName}} --help
{{.Sp3}}{{.AppName}} --version
{{if eq .UsageVersion "short"}}
Use '{{.AppName}} --help' to get detailed information about options and
examples of usage.{{else}}

DESCRIPTION:
{{.Sp3}}{{.AppName}} mounts a synthesized file system which purpose is to generate
{{.Sp3}}trace events for each low level file I/O operation executed on any file
{{.Sp3}}or directory under its control. Examples of such operations are open(2),
{{.Sp3}}read(2), write(2), close(2), access(2), etc.

{{.Sp3}}{{.AppName}} exposes the contents of the directory specified by the
{{.Sp3}}option '--shadow' via the path specified by the option '--mount'. {{.AppName}}
{{.Sp3}}generates a trace event for each I/O operation and forwards the operation
{{.Sp3}}to the target file system, that is, the one which actually hosts the shadow
{{.Sp3}}directory. See the EXAMPLES section below.

{{.Sp3}}Individual trace events generated by {{.AppName}} are written to the specified
{{.Sp3}}output file (option --out) in the specified format.


OPTIONS:
{{.Sp3}}--mount=<directory>
{{.Tab1}}This is the top directory through which the files and directories residing
{{.Tab1}}under the shadow directory will be exposed. See the EXAMPLES section below.
{{.Tab1}}The specified directory must exist and must be empty.

{{.Sp3}}--shadow=<directory>
{{.Tab1}}This is a directory where the files and directories you want to trace
{{.Tab1}}actually reside.
{{.Tab1}}The specified directory must exist but may be empty.

{{.Sp3}}--out=<file>
{{.Tab1}}Path of the text file to write the trace events to. If this file
{{.Tab1}}does not exist it will be created, otherwise new events will be appended.
{{.Tab1}}Note that this file cannot be located under the shadow directory.
{{.Tab1}}Use '-' (dash) to write the trace events to the standard output.
{{.Tab1}}In addition, you can specify a file name with extension '.csv' or '.json'
{{.Tab1}}to instruct {{.AppName}} to emit records in the corresponding format,
{{.Tab1}}as if you had used the '--csv' or '--json' options (see below).
{{.Tab1}}Default: write trace records to standard output.

{{.Sp3}}--csv
{{.Tab1}}Format each individual trace event generated by {{.AppName}} as a set of
{{.Tab1}}comma-separated values in a single line.
{{.Tab1}}Note that not all events contain the same information since each
{{.Tab1}}low level I/O operation requires specific arguments. Please refer to
{{.Tab1}}the documentation at 'https://github.com/airnandez/{{.AppName}}' for
{{.Tab1}}details on the format of each event.
{{.Tab1}}CSV is the default output format unless the output file name (see option
{{.Tab1}}'--out' above) has a '.json' extension.

{{.Sp3}}--json
{{.Tab1}}Format each individual trace event generated by {{.AppName}} as
{{.Tab1}}a JSON object. Events in this format are self-described but not all
{{.Tab1}}events contain the same information since each low level I/O operation
{{.Tab1}}requires specific arguments. Please refer to the documentation
{{.Tab1}}at 'https://github.com/airnandez/{{.AppName}}' for details on the format
{{.Tab1}}of each event.

{{.Sp3}}--ro
{{.Tab1}}Expose the shadow file system as a read-only file system.
{{.Tab1}}Default: if this option is not specified, the file system is mounted in
{{.Tab1}}read-write mode.

{{.Sp3}}--help
{{.Tab1}}Show this help

{{.Sp3}}--version
{{.Tab1}}Show version information and source repository location

EXAMPLES:
{{.Sp3}}To trace file I/O operations on files under $HOME/data use:

{{.Tab1}}{{.AppName}} --mount=/tmp/trace --shadow=$HOME/data

{{.Sp3}}After a successfull mount, the contents under $HOME/data are also
{{.Sp3}}accessible by using the path /tmp/trace. For instance, if the
{{.Sp3}}file $HOME/data/hello.txt exists, {{.AppName}} traces all the file I/O
{{.Sp3}}operations induced by the command:

{{.Tab1}}cat /tmp/trace/hello.txt

{{.Sp3}}Trace events for each one of the low level operations induced by the
{{.Sp3}}'cat' command above will be written to the output file, the standard
{{.Sp3}}output in this particular example.

{{.Sp3}}You can also create new files under /tmp/trace. For instance, the file
{{.Sp3}}I/O operations induced by the shell command:

{{.Tab1}}echo "This is a new file" > /tmp/trace/newfile.txt

{{.Sp3}}will be traced and the file will actually be created in
{{.Sp3}}$HOME/data/newfile.txt. This file will persist even after unmounting
{{.Sp3}}{{.AppName}} (see below on how to unmount the synthetized file system).

{{.Sp3}}Please note that any destructive action, such as removing or modifying
{{.Sp3}}the contents of a file or directory using the path /tmp/trace will
{{.Sp3}}affect the corresponding file or directory under $HOME/data.
{{.Sp3}}For example, the command:

{{.Tab1}}rm /tmp/trace/notes.txt

{{.Sp3}}will have the same destructive effect as if you had executed

{{.Tab1}}rm $HOME/data/notes.txt

{{.Sp3}}To unmount the file system exposed by {{.AppName}} use:

{{.Tab1}}umount /tmp/trace

{{.Sp3}}Alternatively, on MacOS X you can also use the diskutil(8) command:

{{.Tab1}}/usr/sbin/diskutil unmount /tmp/trace
{{end}}
`

	fields := map[string]string{
		"AppName":       programName,
		"AppNameFiller": strings.Repeat(" ", len(programName)),
		"Sp2":           "  ",
		"Sp3":           "   ",
		"Sp4":           "    ",
		"Sp5":           "     ",
		"Sp6":           "      ",
		"Tab1":          "\t",
		"Tab2":          "\t\t",
		"Tab3":          "\t\t\t",
		"Tab4":          "\t\t\t\t",
		"Tab5":          "\t\t\t\t\t",
		"Tab6":          "\t\t\t\t\t\t",
		"UsageVersion":  "short",
	}
	if kind == HelpLong {
		fields["UsageVersion"] = "long"
	}
	minWidth, tabWidth, padding := 8, 4, 0
	tabwriter := tabwriter.NewWriter(f, minWidth, tabWidth, padding, byte(' '), 0)
	templ := template.Must(template.New("").Parse(usageTempl))
	templ.Execute(tabwriter, fields)
	tabwriter.Flush()
}

// printVersionInfo prints the version information about this application
func printVersionInfo(f *os.File) {
	const versionTempl = `
{{.AppName}} version {{.AppVersion}} ({{.Os}},{{.Arch}})

Built on:
{{.Sp3}}{{.BuildTime}}

Author:
{{.Sp3}}Fabio Hernandez
{{.Sp3}}IN2P3/CNRS computing center, Lyon (France)

Source code and documentation:
{{.Sp3}}https://github.com/airnandez/{{.AppName}}
`
	fields := map[string]string{
		"AppName":    programName,
		"AppVersion": version,
		"BuildTime":  buildTime,
		"Os":         runtime.GOOS,
		"Arch":       runtime.GOARCH,
		"Sp3":        "   ",
	}
	minWidth, tabWidth, padding := 8, 4, 0
	tabwriter := tabwriter.NewWriter(f, minWidth, tabWidth, padding, byte(' '), 0)
	templ := template.Must(template.New("").Parse(versionTempl))
	templ.Execute(tabwriter, fields)
	tabwriter.Flush()
}
