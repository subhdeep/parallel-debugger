package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
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
		fallthrough
	default:
		// fmt.Printf("Class = %s\n", notification["class"])
	}
}

func SynchronizedSend(gdb *gdb.Gdb, operation string, arguments ...string) map[string]interface{} {
	result, err := gdb.Send(operation, arguments...)
	CheckError(err)
	handleNotifications(result)
	return result
}

func getSoFilepath() string {
	dir := os.Getenv("PD_FILE_DIR")
	if dir == "" {
		dir_, err := os.Getwd()
		CheckError(err)
		dir = dir_
	}

	fname := fmt.Sprintf("%s/mpic.so", dir)
	_, err := os.Stat(fname)
	if err != nil && os.IsNotExist(err) {
		// Path is wrong for the shared library, throw error.
		log.Fatalln("Shared object does not exist, set PD_FILE_DIR to directory containing mpic.so")
	}

	return fname
}

// InitGDB initializes the GDB interpreter
func InitGDB(filename string) (pdFilename string) {
	// start a new instance and pipe the target output to stdout
	gdb, _ := gdb.New(handleNotifications)
	go io.Copy(os.Stdout, gdb)
	// go io.Copy(gdb, os.Stdin)
	fname := getSoFilepath()
	pdFilename = fmt.Sprintf("/tmp/pd_init_data_%d", os.Getpid())
	SynchronizedSend(gdb, "set breakpoint pending on")
	SynchronizedSend(gdb, "file", filename)
	SynchronizedSend(gdb, fmt.Sprintf("set exec-wrapper env 'LD_PRELOAD=%s' 'FILENAME=%s'", fname, pdFilename))
	SynchronizedSend(gdb, "break PMPI_Init")
	SynchronizedSend(gdb, "run")
	SynchronizedSend(gdb, "finish")
	SynchronizedSend(gdb, "finish")
	SynchronizedSend(gdb, "continue")

	return
}
