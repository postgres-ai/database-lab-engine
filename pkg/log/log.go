/*
2019 © Postgres.ai
*/

package log

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

var debugMode bool = true

var std = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)

const (
	WHITE  = "\x1b[1;37m"
	RED    = "\x1b[1;31m"
	GREEN  = "\x1b[1;32m"
	YELLOW = "\x1b[1;33m"
	END    = "\x1b[0m"
	OK     = GREEN + "OK" + END
	FAIL   = RED + "Fail" + END
)

const (
	calldepth = 3
)

func toString(i1 interface{}) string {
	if i1 == nil {
		return ""
	}
	switch i2 := i1.(type) {
	default:
		return fmt.Sprint(i2)
	case bool:
		if i2 {
			return "true"
		} else {
			return "false"
		}
	case string:
		return i2
	case *bool:
		if i2 == nil {
			return ""
		}
		if *i2 {
			return "true"
		} else {
			return "false"
		}
	case *string:
		if i2 == nil {
			return ""
		}
		return *i2
	case *json.Number:
		return i2.String()
	case json.Number:
		return i2.String()
	}
}

func prepareMessage(v ...interface{}) string {
	message := ""
	for _, value := range v {
		message = message + " " + toString(value)
	}
	return message
}

func println(v ...interface{}) {
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

// Output message.
func Msg(v ...interface{}) {
	println("[INFO]  " + prepareMessage(v...))
}

// Warn outputs a warning message.
func Warn(v ...interface{}) {
	println("[WARNING]  " + prepareMessage(v...))
}

// Output debug message.
func Dbg(v ...interface{}) {
	if debugMode {
		println("[DEBUG] " + prepareMessage(v...))
	}
}

// Output error message.
func Err(v ...interface{}) {
	println("[ERROR] " + prepareMessage(v...))
}

// Errf outputs formatted log.
func Errf(format string, v ...interface{}) {
	printf("[ERROR] "+format, v...)
}

// Messages for security audit.
func Audit(v ...interface{}) {
	println("[AUDIT] " + prepareMessage(v...))
}

func Fatal(v ...interface{}) {
	log.Fatal("[FATAL] " + prepareMessage(v...))
}

// Fatalf prints an error with a stack trace.
func Fatalf(err error) {
	log.Fatalf("[FATAL] %+v", err)
}
