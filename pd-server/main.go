package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

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
			fmt.Fprintf(*v, "COMMAND:All clients, including you, are connected\n")
		}
		go takeUserInput()
		processClient := make(chan bool)
		go processClientMessage(wSize, processClient)
		<-processClient
		connections = make(map[int]*net.Conn)
	}
}

func takeUserInput() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		command, ranks := parseInput(input)
		sendCommandTo(command, ranks)
	}
}

func parseInput(input string) (command string, ranks []int) {
	command = input
	ranks = nil
	if !strings.HasPrefix(command, "pdb_") {
		return
	}

	commandSegments := strings.Split(input, "\t ")
	rankSpecString := strings.TrimSpace(commandSegments[len(commandSegments) - 1])

	commandSegments = commandSegments[:len(commandSegments)-1]
	command = strings.Join(commandSegments, " ")
	command = strings.TrimPrefix(command, "pdb_")

	// Right now, only support comma separated ranks.
	ranksString := strings.Split(strings.Split(rankSpecString, "=")[1], ",")
	ranks = make([]int, len(ranksString))
	for i, rs := range ranksString {
		ranks[i], _ = strconv.Atoi(rs)
	}
	fmt.Println(command, ranks)
	return
}

func sendCommandTo(message string, ranks []int) {
	if ranks != nil {
		for _, rank := range ranks {
			c := connections[rank]
			fmt.Fprintf(*c, fmt.Sprintf("RUN:%s\n", message))
		}
		return
	}

	for _, c := range connections {
		fmt.Fprintf(*c, fmt.Sprintf("RUN:%s\n", message))
	}
}

func processClientMessage(wSize int, processClientDone chan bool) {
	if wSize != len(connections) {
		return
	}

	var waitGroup sync.WaitGroup
	waitGroup.Add(wSize)

	for r, c := range connections {

		// handling output of every client in a separate go routine
		go func(r int, c *net.Conn) {
			defer waitGroup.Done()
			scanner := bufio.NewScanner(*c)
			for scanner.Scan() {
				utils.CheckError(scanner.Err())
				line := scanner.Text()
				fmt.Printf("rank %d: %s\n", r, line)
			}
		}(r, c)

	}
	waitGroup.Wait()
	processClientDone <- true
}
