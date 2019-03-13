package utils

import (
	"log"
)

func CheckError(err error) {
	if err != nil {
		log.Fatalf("Fatal error: %s\n", err.Error())
	}
}
