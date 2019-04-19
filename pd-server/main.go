package main

import (
	"bufio"
	"container/list"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"git.cse.iitk.ac.in/ssaha/parallel-debugger/pd-server/tui"
	"git.cse.iitk.ac.in/ssaha/parallel-debugger/utils"
	tuiGo "github.com/marcusolsson/tui-go"
)

var connections = make(map[int]*net.Conn)

type CollectiveCall struct {
	funcName string
	callers  map[int]*utils.CollectiveInfo
}

var collectiveCallList struct {
	calls *list.List
	mux   sync.Mutex
}

func main() {
	// Initialize some structs.
	collectiveCallList.mux.Lock()
	collectiveCallList.calls = list.New()
	collectiveCallList.mux.Unlock()

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
		t := tui.NewTUI(connections)
		// 	t.ShowMessagesAll(e.Text())
		// })
		t.DrawUI()
		t.ShowMessagesAll("You are connected")
		t.Input.OnSubmit(func(e *tuiGo.Entry) {
			if e.Text() == "quit" {
				t.Quit()
			}
			takeUserInput(e.Text())
			t.Input.SetText("")
		})

		processClient := make(chan bool)
		go processClientMessage(wSize, processClient, t)
		<-processClient
		connections = make(map[int]*net.Conn)
	}

}

func takeUserInput() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()
		if input == "pdb_listcoll" {
			calls := pendingCollectiveInfo()
			prettyPrintCollectiveInfo(calls)
		} else if strings.HasPrefix(input, "pdb_trackcoll") {
			toggleCollective(strings.Split(input)[1])
		} else {
			command, ranks := parseInput(input)
			sendCommandTo(command, ranks)
		}
	}
}

func sendCommandTo(message string, ranks []int) {
	sendMsgTo(message, ranks, "RUN")
}

func toggleCollective(coll string) {
	sendMsgTo(coll, nil, "COLLECTIVE")
}

// Send the `message` to ranks specified inside `ranks`.
// If `ranks` is nil, then send message to all the connected clients.
// In case we are trying to send a message to some non-existent client,
// ignore that silently.
// The sent message is of the form <prefix>:<message>
func sendMsgTo(message string, ranks []int, prefix string) {
	if ranks != nil {
		for _, rank := range ranks {
			c, rankExists := connections[rank]
			if !rankExists {
				continue
			}
			fmt.Fprintf(*c, "%s", fmt.Sprintf("%s:%s\n", prefix, message))
		}
		return
	}

	for _, c := range connections {
		fmt.Fprintf(*c, "%s", fmt.Sprintf("%s:%s\n", prefix, message))
	}
}

func processClientMessage(wSize int, processClientDone chan bool, t *tui.TUI) {
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
				lineSplit := strings.SplitN(line, ":", 2)
				handleClientMessage(lineSplit[0], lineSplit[1], r)
			}
		}(r, c)
	}

	waitGroup.Wait()
	processClientDone <- true
}

func handleClientMessage(cat string, msg string, rank int) {
	switch cat {
	case "CONSOLE":
		fmt.Printf("[rank %d] %s\n", rank, msg)
	case "ERROR":
		fmt.Printf("[rank %d] (!) %s\n", rank, msg)
	case "COLLECTIVE":
		var coll utils.CollectiveInfo
		_ = json.Unmarshal([]byte(msg), &coll)
		trackCollective(coll)
	}
}

func trackCollective(info utils.CollectiveInfo) {
	collectiveCallList.mux.Lock()
	defer collectiveCallList.mux.Unlock()
	cl := collectiveCallList.calls
	var c_ *list.Element
	var toRemove *list.Element = nil
	for c_ = cl.Front(); c_ != nil; c_ = c_.Next() {
		c := c_.Value.(*CollectiveCall)
		_, ok := c.callers[info.Rank]
		if c.funcName == info.FunctionName && !ok {
			c.callers[info.Rank] = &info
			if len(c.callers) == len(connections) {
				toRemove = c_
			}
			break
		}
	}

	if toRemove != nil {
		cl.Remove(toRemove)
	}

	if c_ == nil {
		clrs := make(map[int]*utils.CollectiveInfo)
		clrs[info.Rank] = &info
		c := &CollectiveCall{
			info.FunctionName,
			clrs,
		}
		cl.PushBack(c)
	}
}

func pendingCollectiveInfo() (calls []CollectiveCall) {
	collectiveCallList.mux.Lock()
	defer collectiveCallList.mux.Unlock()
	cl := collectiveCallList.calls
	for c_ := cl.Front(); c_ != nil; c_ = c_.Next() {
		c := c_.Value.(*CollectiveCall)
		calls = append(calls, *c)
	}
	return
}

func prettyPrintCollectiveInfo(calls []CollectiveCall) {
	for _, call := range calls {
		s := fmt.Sprintf("Collective function %s:\n", call.funcName)
		for i := 0; i < len(connections); i++ {
			info, ok := call.callers[i]
			if !ok {
				s += fmt.Sprintf("Rank %d: pending\n", i)
			} else {
				s += fmt.Sprintf("Rank %d: Called at %s\n", i, info.LineInfo)
			}
		}
		fmt.Println(s)
	}
}
