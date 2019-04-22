package main

import (
	"bufio"
	"fmt"
	"log"
	"encoding/json"
	"net"
	"os"
	"strings"

	"pd-utils"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage %s <hostname>:<port> <filename>\n", os.Args[0])
		os.Exit(1)
	}
	conn, err := net.Dial("tcp", os.Args[1])
	utils.CheckError(err)
	filename := os.Args[2]

	cInfoChan := make(chan utils.CollectiveInfo)
	gdbInstance := utils.NewGdb(cInfoChan)

	pdFilename := gdbInstance.InitGdb(filename)

	f, err := os.Open(pdFilename)
	utils.CheckError(err)
	line, err := bufio.NewReader(f).ReadString('\n')
	utils.CheckError(err)
	fmt.Fprintf(conn, "%s", line)
	log.Printf("GDB Initialized\n")

	// Each time we get some communicator info, send to the server to process.
	go (func() {
		var c utils.CollectiveInfo
		for {
			c = <-cInfoChan
			out, err := json.Marshal(c)
			if err != nil {
				continue
			}
			fmt.Fprintf(conn, "COLLECTIVE:%s\n", out)
		}
	})()

	// Each output that the gdb instance gets from gdb mi must be processed.
	// One hook is added here, which will send all ~console messages to the server.
	gdbInstance.AddNotificationHook("ConsoleSendingHook", func(notification map[string]interface{}) bool {
		if notification["type"] == "console" {
			// On getting a console notification, relay it to the server.
			// Filter newlines though. They will be added by us at server side!
			payload := strings.TrimSpace(notification["payload"].(string))
			if payload == "" || payload == "\n" {
				return true
			}
			fmt.Fprintf(conn, "CONSOLE:%s\n", payload)
			fmt.Println(payload)
		}
		return true
	})

	gdbInstance.AddNotificationHook("ErrorSendingHook", func(notification map[string]interface{}) bool {
		if notification["class"] == "error" {
			// On getting a error notification, tell the server.
			payload := notification["payload"].(map[string]interface{})
			msg := payload["msg"].(string)
			msg = strings.TrimSpace(msg)
			fmt.Fprintf(conn, "ERROR:%s\n", msg)
			fmt.Println(msg)
		}
		return true
	})

	gdbInstance.AddNotificationHook("LoggingHook", func(notification map[string]interface{}) bool {
		jsonStr, _ := json.Marshal(notification)
		log.Println(string(jsonStr))
		return true
	})

	// Each message from the server needs to be processed using ProcessMessage.
	processCommandsDone := make(chan bool)
	go gdbInstance.ProcessCommands(conn, processCommandsDone)
	<-processCommandsDone
}
