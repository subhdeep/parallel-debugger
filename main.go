package main

import (
	"fmt"
	"github.com/cyrus-and/gdb"
	"io"
	"log"
	"os"
	"encoding/json"
)

var breakpointHitNotification = make(chan int)

func handleNotifications (notification map[string]interface{}) {
	if notification["class"] == "library-loaded" || notification["class"] == "library-unloaded" {
		return
	}
	jsonStr, _ := json.Marshal(notification)
	fmt.Println(string(jsonStr))

	if (notification["class"] == "stopped") {
		breakpointHitNotification <- 1
	}

	// if (notification["class"] == )
}

func main() {
	// start a new instance and pipe the target output to stdout
	gdb, _ := gdb.New(handleNotifications)
	go io.Copy(os.Stdout, gdb)
	go io.Copy(gdb, os.Stdin)

	if len(os.Args) < 2 {
		log.Fatal("Need to have filename to execute")
	}

	filename := os.Args[1]

	// load and run a program
	result, err := gdb.Send("file-exec-and-symbols", filename)

	if err != nil {
		log.Fatal(err)
	}

	handleNotifications(result)

	result, err = gdb.Send("break-insert", "4")
	if err != nil {
		log.Fatal(err)
	}

	handleNotifications(result)

	gdb.Send("exec-run")
	<-breakpointHitNotification

	fmt.Println("-------------------- Starting EXEC-NEXT")
	gdb.Send("exec-next")
	fmt.Println("-------------------- Finished EXEC-NEXT")
	<-breakpointHitNotification
	fmt.Println("-------------------- Got 'stop' notification")

	fmt.Println("-------------------- Starting DATA-EVAL-EXPRESSION")
	gdb.Send("data-evaluate-expression", "init_debugger()")
	fmt.Println("-------------------- Starting EXEC-RUN")
	gdb.Send("exec-run")
	fmt.Println("-------------------- Finished EXEC-RUN")
	<-breakpointHitNotification
	fmt.Println("-------------------- Got 'stop' notification")


	gdb.Exit()
}
