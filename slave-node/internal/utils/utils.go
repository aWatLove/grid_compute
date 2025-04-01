package utils

import (
	"log"
	"runtime/debug"
)

func Recovery(service string) {
	if recoveryMessage := recover(); recoveryMessage != nil {
		log.Println("[%s][RECOVERY] Panic message: %s\n", service, recoveryMessage)
		log.Println("[%s][RECOVERY] Panic Stacktrace:\n%s\n", service, string(debug.Stack()))
	}
}
