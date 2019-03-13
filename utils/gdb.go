package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/cyrus-and/gdb"
)

var breakpointHitNotification = make(chan int)

func handleNotifications(notification map[string]interface{}) {
	if notification["class"] == "library-loaded" || notification["class"] == "library-unloaded" {
		return
	}
	jsonStr, _ := json.Marshal(notification)
	fmt.Println(string(jsonStr))

	switch notification["class"] {
	case "done":
	case "running":
		<-breakpointHitNotification
	case "stopped":
		breakpointHitNotification <- 1
	case "error":
		fmt.Println("TODO: Error handling")
	}
}

func SynchronizedSend(operation string, arguments ...string) (map[string]interface{}) {
	result, err := gdb.Send(operation, arguments)
	CheckError(err)
	handleNotifications(result)
	return result
}

// InitGDB initializes the GDB interpreter
func InitGDB(filename string) {
	// start a new instance and pipe the target output to stdout
	gdb, _ := gdb.New(handleNotifications)
	go io.Copy(os.Stdout, gdb)
	go io.Copy(gdb, os.Stdin)

	result := SynchronizedSend("file", filename)
	result = SynchronizedSend("set exec-wrapper env \"LD_PRELOAD=./mpic.so\"")
	result = SynchronizedSend("run")
	SynchronizedSend("quit")
}
