package main

import (
	"github.com/op/go-logging"
	"os"
)

// Example format string. Everything except the message has a custom color
// which is dependent on the log level. Many fields have a custom output
// formatting too, eg. the time returns the hour down to the milli second.
//     %{id}        Sequence number for log message (uint64).
//     %{pid}       Process id (int)
//     %{time}      Time when log occurred (time.Time)
//     %{level}     Log level (Level)
//     %{module}    Module (string)
//     %{program}   Basename of os.Args[0] (string)
//     %{message}   Message (string)
//     %{longfile}  Full file name and line number: /a/b/c/d.go:23
//     %{shortfile} Final file name element and line number: d.go:23
//     %{color}     ANSI color based on log level
//     %{longpkg}   Full package path, eg. github.com/go-logging
//     %{shortpkg}  Base package path, eg. go-logging
//     %{longfunc}  Full function name, eg. littleEndian.PutUint32
//     %{shortfunc} Base function name, eg. PutUint32
var stdFormat = logging.MustStringFormatter(
	"%{color}%{time:15:04:05.000} %{shortfile} >%{level:.5s}%{color:reset} - %{message}",
)

var fileFormat = logging.MustStringFormatter(
	"%{time:15:04:05.000} >%{level:.5s} - %{message}",
)

func InitLogger(logPath string, level logging.Level) error {
	fp, err := os.OpenFile(logPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 06660)
	if err != nil {
		return err
	}

	stdBackend := logging.NewLogBackend(os.Stdout, "", 1)
	fileBackend := logging.NewLogBackend(fp, "", 1)

	stdFormatter := logging.NewBackendFormatter(stdBackend, stdFormat)
	fileFormatter := logging.NewBackendFormatter(fileBackend, fileFormat)

	stdB := logging.AddModuleLevel(stdFormatter)
	stdB.SetLevel(level, "")

	fileB := logging.AddModuleLevel(fileFormatter)
	fileB.SetLevel(level, "")

	// Set the backends to be used.
	logging.SetBackend(stdB, fileB)

	return nil
}
