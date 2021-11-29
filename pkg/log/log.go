/*
2019 © Postgres.ai
*/

// Package log formats and prints log messages.
package log

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

var debugMode = true

var std = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)

const (
	calldepth = 3
)

func toString(i1 interface{}) string {
	if i1 == nil {
		return ""
	}

	switch i2 := i1.(type) {
	case bool:
		if i2 {
			return "true"
		}

		return "false"

	case string:
		return i2

	case *bool:
		if i2 == nil {
			return ""
		}

		if *i2 {
			return "true"
		}

		return "false"

	case *string:
		if i2 == nil {
			return ""
		}

		return *i2

	case *json.Number:
		return i2.String()

	case json.Number:
		return i2.String()

	default:
		return fmt.Sprint(i2)
	}
}

func prepareMessage(v ...interface{}) string {
	builder := strings.Builder{}

	for _, value := range v {
		builder.WriteString(" " + toString(value))
	}

	return builder.String()
}

func printLine(v ...interface{}) {
	_ = std.Output(calldepth, fmt.Sprintln(v...))
}

func printf(format string, v ...interface{}) {
	_ = std.Output(calldepth, fmt.Sprintf(format, v...))
}

// SetDebug enables debug logs.
func SetDebug(enable bool) {
	debugMode = enable

	if debugMode {
		std.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		std.SetFlags(log.LstdFlags)
	}
}

// Msg outputs message.
func Msg(v ...interface{}) {
	printLine("[INFO]  " + prepareMessage(v...))
}

// Warn outputs a warning message.
func Warn(v ...interface{}) {
	printLine("[WARNING]  " + prepareMessage(v...))
}

// Dbg outputs debug message.
func Dbg(v ...interface{}) {
	if debugMode {
		printLine("[DEBUG] " + prepareMessage(v...))
	}
}

// Err outputs error message.
func Err(v ...interface{}) {
	printLine("[ERROR] " + prepareMessage(v...))
}

// Errf outputs formatted log.
func Errf(format string, v ...interface{}) {
	printf("[ERROR] "+format, v...)
}

// Audit outputs messages for security audit.
func Audit(v ...interface{}) {
	printLine("[AUDIT] " + prepareMessage(v...))
}

// Fatal prints fatal message and exits.
func Fatal(v ...interface{}) {
	log.Fatal("[FATAL] " + prepareMessage(v...))
}

// Fatalf prints an error with a stack trace.
func Fatalf(err error) {
	log.Fatalf("[FATAL] %+v", err)
}
