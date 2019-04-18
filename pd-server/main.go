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

// Parse commands.
// There are two types of commands that can be parsed right now.
// 1. Normal GDB Commands
// 2. PDB-specific commands of the format pdb_<command> [r=g,g,g,g] where g
//    can be a single rank, or something like x..y, where x < y.
func parseInput(input string) (command string, ranks []int) {
	input = strings.TrimSpace(input)
	command = input
	ranks = nil

	// If it's a normal gdb command, return immediately.
	if !strings.HasPrefix(command, "pdb_") {
		return
	}

	// Split on any number of spaces or tabs, not just single " ".
	commandSegments := strings.Fields(input)

	// Get the last bit with r=g,g,g,g
	rankSpecString := strings.TrimSpace(commandSegments[len(commandSegments)-1])
	rankSpecString = strings.Trim(rankSpecString, "[]")

	// Get the <command> by stripping away pdb_ and [r=g,g,g,g]
	commandSegments = commandSegments[:len(commandSegments)-1]
	command = strings.Join(commandSegments, " ")
	command = strings.TrimPrefix(command, "pdb_")

	// Using r=g,g,g,g, come up with a list of ranks that have to be
	// sent the command. Note that existence of these ranks is not
	// guaranteed, we just make up the list of ranks based on input.
	rankListPrefix := strings.Split(rankSpecString, "=")[0]
	if rankListPrefix != "r" {
		log.Println("Rank list prefix other than r is not supported, so sending to all ranks silently.")
		return
	}

	rankGroups := strings.Split(strings.Split(rankSpecString, "=")[1], ",")
	rankSet := make(map[int]bool) // Go's approximation of a set.
	for _, rg := range rankGroups {
		// First check if g is one single rank.
		rank, err := strconv.Atoi(rg)
		if err != nil {
			rgEndpoints := strings.Split(rg, "..")
			if len(rgEndpoints) != 2 {
				continue
			}

			low, err := strconv.Atoi(rgEndpoints[0])
			if err != nil {
				continue
			}

			high, err := strconv.Atoi(rgEndpoints[1])
			if err != nil {
				continue
			}

			for i := low; i < high; i++ {
				rankSet[i] = true
			}

		} else {
			rankSet[rank] = true
		}
	}

	ranks = make([]int, len(rankSet))
	i := 0
	for k := range rankSet {
		ranks[i] = k
		i++
	}
	return
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
		input := scanner.Text()
		command, ranks := parseInput(input)
		sendCommandTo(command, ranks)
	}
}

// Send the `message` to ranks specified inside `ranks`.
// If `ranks` is nil, then send message to all the connected clients.
// In case we are trying to send a message to some non-existent client,
// ignore that silently.
func sendCommandTo(message string, ranks []int) {
	if ranks != nil {
		for _, rank := range ranks {
			c, rankExists := connections[rank]
			if !rankExists {
				continue
			}
			fmt.Fprintf(*c, "%s", fmt.Sprintf("RUN:%s\n", message))
		}
		return
	}

	for _, c := range connections {
		fmt.Fprintf(*c, "%s", fmt.Sprintf("RUN:%s\n", message))
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
