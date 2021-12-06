package serial

import (
	"log"
)

func EnableTrace(enable bool) {
	traceEnabled = enable
}

var traceEnabled = false

func trace(args ...interface{}) {
	if traceEnabled {
		log.Println(args...)
	}
}
