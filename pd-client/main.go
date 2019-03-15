package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"

	"git.cse.iitk.ac.in/ssaha/parallel-debugger/utils"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage %s <hostname>:<port> <filename>\n", os.Args[0])
		os.Exit(1)
	}
	conn, err := net.Dial("tcp", os.Args[1])
	utils.CheckError(err)
	filename := os.Args[2]

	gdbInstance := utils.NewGdb()

	pdFilename := gdbInstance.InitGdb(filename, conn)

	f, err := os.Open(pdFilename)
	utils.CheckError(err)
	line, err := bufio.NewReader(f).ReadString('\n')
	utils.CheckError(err)
	fmt.Fprintf(conn, "%s", line)
	log.Printf("GDB Initialized\n")

	// Each output that the gdb instance gets from gdb mi must be processed.
	// One hook is added here, which will send all ~console messages to the server.
	gdbInstance.AddNotificationHook("ConsoleSendingHook", func(notification map[string]interface{}) bool {
		if notification["type"] == "console" {
			// On getting a console notification, relay it to the server.
			fmt.Fprintf(conn, "CONSOLE:%s", notification["payload"])
		}
		return true
	})

	gdbInstance.AddNotificationHook("ErrorSendingHook", func(notification map[string]interface{}) bool {
		if notification["class"] == "error" {
			// On getting a error notification, tell the server.
			payload := notification["payload"].(map[string]interface{})
			fmt.Fprintf(conn, "CONSOLE:%s\n", payload["msg"])
		}
		return true
	})

	// Each message from the server needs to be processed using ProcessMessage.
	processCommandsDone := make(chan bool)
	go gdbInstance.ProcessCommands(conn, processCommandsDone)
	<-processCommandsDone
}
