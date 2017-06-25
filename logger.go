package main

import (
	"io"
	"log"
)

var (
	// Debug logger interface. Should be kept to development only.
	Debug *log.Logger
	// Info logger interface. Use when logging actual information.
	Info *log.Logger
	// Warning logger interface. Only warn when the problem isn't guaranteed.
	Warning *log.Logger
	// Error logger interface.
	Error *log.Logger
)

// InitLog initializes the logger interfaces. To mute any of the interfaces,
// pass in ioutil.Discard for that interface's output writer.
func InitLog(debugOut, infoOut, warningOut, errorOut io.Writer) {
	Debug = log.New(debugOut, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	Info = log.New(infoOut, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warning = log.New(warningOut, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(errorOut, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}