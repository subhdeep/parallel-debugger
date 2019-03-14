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

	pdFilename := gdbInstance.InitGdb(filename)

	f, err := os.Open(pdFilename)
	utils.CheckError(err)
	line, err := bufio.NewReader(f).ReadString('\n')
	utils.CheckError(err)
	fmt.Fprintf(conn, "%s", line)
	log.Printf("GDB Initialized\n")

	// From now on, each command that this receives will be run inside gdb.
	// And each output that it gets will be handled by the server.
	gdbInstance.AddNotificationHook("ConsoleSendingHook", func(notification map[string]interface{}) bool {
		if notification["type"] == "console" {
			// On getting a console notification, relay it to the server.
			fmt.Fprintf(conn, "CONSOLE:%s", notification["payload"])
		}
		return true
	})

	processCommandsDone := make(chan bool)
	go gdbInstance.ProcessCommands(conn, processCommandsDone)
	<-processCommandsDone
}
