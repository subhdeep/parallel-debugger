package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/milindl/gdb"
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
		if len(notification) == 1 {
			<-breakpointHitNotification
		}
	case "stopped":
		breakpointHitNotification <- 1
	case "error":
		fmt.Println("TODO: Error handling")
	// default:
	// 	fmt.Printf("Class = %s\n", notification["class"])
	}
}

func SynchronizedSend(gdb *gdb.Gdb, operation string, arguments ...string) (map[string]interface{}) {
	result, err := gdb.Send(operation, arguments...)
	CheckError(err)
	handleNotifications(result)
	return result
}

// InitGDB initializes the GDB interpreter
func InitGDB(filename string) {
	// start a new instance and pipe the target output to stdout
	gdb, _ := gdb.New(handleNotifications)
	go io.Copy(os.Stdout, gdb)
	// go io.Copy(gdb, os.Stdin)


	SynchronizedSend(gdb, "file", filename)
	SynchronizedSend(gdb, "set exec-wrapper env \"LD_PRELOAD=./mpic.so\"")
	SynchronizedSend(gdb, "run")
	gdb.Exit()
}
