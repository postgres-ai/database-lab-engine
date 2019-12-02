/*
2019 © Postgres.ai
*/

package log

import (
	"encoding/json"
	"fmt"
	"log"
)

var DEBUG bool = true

const (
	WHITE  = "\x1b[1;37m"
	RED    = "\x1b[1;31m"
	GREEN  = "\x1b[1;32m"
	YELLOW = "\x1b[1;33m"
	END    = "\x1b[0m"
	OK     = GREEN + "OK" + END
	FAIL   = RED + "Fail" + END
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

// Output message.
func Msg(v ...interface{}) {
	log.Println("[INFO]  " + prepareMessage(v...))
}

// Output debug message.
func Dbg(v ...interface{}) {
	if DEBUG {
		log.Println("[DEBUG] " + prepareMessage(v...))
	}
}

// Output error message.
func Err(v ...interface{}) {
	log.Println("[ERROR] " + prepareMessage(v...))
}

// Messages for security audit.
func Audit(v ...interface{}) {
	log.Println("[AUDIT] " + prepareMessage(v...))
}

func Fatal(v ...interface{}) {
	log.Fatal("[FATAL] " + prepareMessage(v...))
}
