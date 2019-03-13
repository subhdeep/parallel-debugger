package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"git.cse.iitk.ac.in/ssaha/parallel-debugger/utils"
)

var connections = make(map[int]*net.Conn)

func main() {
	port := 8080
	host := "0.0.0.0"
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	utils.CheckError(err)
	log.Printf("Server running on port %d\n", port)
	for {
		conn, err := ln.Accept()
		utils.CheckError(err)
		handleConnection(conn)
	}
}

func handleConnection(c net.Conn) {
	status, err := bufio.NewReader(c).ReadString('\n')
	utils.CheckError(err)

	status = strings.TrimSpace(status)
	rank, err := strconv.Atoi(strings.Split(status, ",")[0])
	utils.CheckError(err)
	wSize, err := strconv.Atoi(strings.Split(status, ",")[1])
	utils.CheckError(err)
	log.Printf("Processing client with rank = %d, world size = %d\n", rank, wSize)

	connections[rank] = &c

	if wSize == len(connections) {
		fmt.Printf("All the clients are connected\n")
		for _, v := range connections {
			fmt.Fprintf(*v, "All clients, including you, are connected\n")
		}
	}
}
