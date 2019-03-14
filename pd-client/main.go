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
	pdFilename := utils.InitGDB(filename)
	f, err := os.Open(pdFilename)
	utils.CheckError(err)
	line, err := bufio.NewReader(f).ReadString('\n')
	utils.CheckError(err)
	fmt.Fprintf(conn, "%s", line)
	log.Printf("GDB Initiaalized\n")
	for {
		status, err := bufio.NewReader(conn).ReadString('\n')
		fmt.Println(status)
		if err != nil {
			fmt.Println(err)
			break
		}
	}
}
