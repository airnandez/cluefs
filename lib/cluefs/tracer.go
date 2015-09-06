package cluefs

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type FsOperTracer interface {
	String() string
	SetTimeEnd()
	MarshalCSV() []string
}

type Tracer interface {
	Trace(op FsOperTracer)
}

type CSVTracer struct {
	writer        *csv.Writer
	receptionChan chan FsOperTracer
}

func NewCSVTracer(filePath string) (*CSVTracer, error) {
	destFile, err := openTraceDestination(filePath)
	if err != nil {
		return nil, err
	}
	tracer := CSVTracer{
		writer:        csv.NewWriter(destFile),
		receptionChan: make(chan FsOperTracer, 1024),
	}
	// Start the event collector
	go func() {
		for op := range tracer.receptionChan {
			tracer.writer.Write(op.MarshalCSV())
			tracer.writer.Flush()
		}
	}()
	return &tracer, nil
}

func (t *CSVTracer) Trace(op FsOperTracer) {
	t.receptionChan <- op
}

type JSONTracer struct {
	file          *os.File
	receptionChan chan FsOperTracer
}

func NewJSONTracer(filePath string) (*JSONTracer, error) {
	destFile, err := openTraceDestination(filePath)
	if err != nil {
		return nil, err
	}
	tracer := JSONTracer{
		file:          destFile,
		receptionChan: make(chan FsOperTracer, 1024),
	}
	// Start the event collector
	go func() {
		crlf := []byte{'\n'}
		for op := range tracer.receptionChan {
			if m, err := json.Marshal(op); err == nil {
				tracer.file.Write(m)
				tracer.file.Write(crlf)
			}
		}
	}()
	return &tracer, nil
}

func (t *JSONTracer) Trace(op FsOperTracer) {
	t.receptionChan <- op
}

func NewTracer(kind, fileName string) (Tracer, error) {
	var (
		tracer Tracer
		err    error
	)
	switch kind {
	default:
		fallthrough
	case "csv":
		tracer, err = NewCSVTracer(fileName)
	case "json":
		tracer, err = NewJSONTracer(fileName)
	}
	return tracer, err
}

func openTraceDestination(filePath string) (*os.File, error) {
	destFile := os.Stdout
	if len(filePath) > 0 && filePath != "-" {
		abspath, err := filepath.Abs(filePath)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve absolute path for '%s' [%s]", filePath, err)
		}
		destFile, err = os.OpenFile(abspath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0640)
		if err != nil {
			return nil, fmt.Errorf("could not open file '%s' for writing [%s]", filePath, err)
		}
	}
	return destFile, nil
}
