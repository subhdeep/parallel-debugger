package utils

import (
	"bufio"
	// "encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/milindl/gdb"
)

const MPI_COMM_WORLD int = 1140850688

type GdbInstance struct {
	breakpointHitNotification chan int
	pdFilename                string
	internal                  *gdb.Gdb
	hooks                     map[string]func(notification map[string]interface{}) bool
	cInfoChan                 chan CollectiveInfo
	trackedCollectives        map[string]bool
}

type CollectiveInfo struct {
	Rank         int
	LineInfo     string
	FunctionName string
}

// NewGdb creates a new GdbInstance struct.
func NewGdb(cInfoChan chan CollectiveInfo) (g *GdbInstance) {
	// start a new instance and pipe the target output to stdout
	g = new(GdbInstance)
	g.hooks = make(map[string]func(notification map[string]interface{}) bool)
	g.breakpointHitNotification = make(chan int)
	g.internal, _ = gdb.New(g.handleNotifications)
	g.cInfoChan = cInfoChan
	g.trackedCollectives = make(map[string]bool)
	return
}

// InitGdb does the following:
// 1. Mirror stdout of the target program to stdout of the go program as well as the server. (TODO: Echo output of target to server)
// 2. Add LD_PRELOAD with the shared library file.
// 3. Run the code uptill MPI_Init, and write data about rank, size to pdFilename.
func (g *GdbInstance) InitGdb(debugTarget string) (pdFilename string) {
	go io.Copy(os.Stdout, g.internal)

	fname := getSoFilepath()
	g.pdFilename = fmt.Sprintf("/tmp/pd_init_data_%d", os.Getpid())

	g.SynchronizedSend("set breakpoint pending on")
	g.SynchronizedSend("file", debugTarget)
	g.SynchronizedSend(fmt.Sprintf("set exec-wrapper env 'LD_PRELOAD=%s' 'FILENAME=%s'", fname, g.pdFilename))
	g.SynchronizedSend("break PMPI_Init")
	g.SynchronizedSend("run")
	g.SynchronizedSend("finish")
	g.SynchronizedSend("finish")
	g.SynchronizedSend("clear PMPI_Init")

	g.toggleCollectiveTracking("MPI_Barrier")
	g.toggleCollectiveTracking("MPI_Barrier")
	// g.SynchronizedSend("break internal_MPI_Barrier")
	// g.SynchronizedSend("break internal_MPI_Bcast")

	return g.pdFilename
}

// This will run indefinitely and process messages from some Reader.
// This reader will (usually) be the Conn of the server.
// For each message, it either runs it in the gdb instance (if the message prefix is RUN:)
// else it prints the message (if the prefix is COMMAND:)
func (g *GdbInstance) ProcessCommands(r io.Reader, processCommandsDone chan bool) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		CheckError(scanner.Err())
		line := scanner.Text()
		lineSplit := strings.SplitN(line, ":", 2)

		if len(lineSplit) != 2 {
			continue
		}

		if lineSplit[0] == "COMMAND" {
			fmt.Printf("Server message: %s\n", lineSplit[1])
		} else if lineSplit[0] == "RUN" {
			fmt.Printf("Running: %s\n", lineSplit[1])
			g.SynchronizedSend(lineSplit[1])
		} else if lineSplit[0] == "COLLECTIVE" {
			g.toggleCollectiveTracking(lineSplit[1])
		}
	}

	processCommandsDone <- true
}

func (g *GdbInstance) toggleCollectiveTracking(coll string) {
	curr_val, ok := g.trackedCollectives[coll]

	if !ok || !curr_val {
		g.trackedCollectives[coll] = true
		g.SynchronizedSend(fmt.Sprintf("break internal_%s", coll))
	} else {
		g.trackedCollectives[coll] = false
		g.SynchronizedSend(fmt.Sprintf("clear internal_%s", coll))
	}
}

// On receiving a notification from gdb, the supplied `hook` will be run.
// Each hook is supposed to take input a notification and return a boolean
// in case it fails. A hook can have a name, so that hooks can be removed/added.
func (g *GdbInstance) AddNotificationHook(hookName string, hook func(notification map[string]interface{}) bool) {
	g.hooks[hookName] = hook
}

func (g *GdbInstance) RemoveNotificationHook(hookName string) {
	delete(g.hooks, hookName)
}

func (g *GdbInstance) handleNotifications(notification map[string]interface{}) {
	if notification["class"] == "library-loaded" || notification["class"] == "library-unloaded" {
		return
	}

	// Handle Synchronization, in case we get a ^running, we should
	// wait for *stopped. Otherwise, we should carry on.
	switch notification["class"] {
	case "done":
	case "running":
		if len(notification) == 1 {
			<-g.breakpointHitNotification
		}

	case "stopped":
		g.breakpointHitNotification <- 1
		isBkpt, funcName, _ := analyzeStoppedProcess(notification["payload"].(map[string]interface{}))
		if isBkpt {
			go g.processBkpt(funcName)
		}
	}

	// Run any custom hooks
	for _, hook := range g.hooks {
		// We don't really use the bool returned anywhere yet.
		hook(notification)
	}
}

func (g *GdbInstance) processBkpt(funcName string) {
	if strings.HasPrefix(funcName, "internal_") {
		if tracking, exists := g.trackedCollectives[strings.TrimPrefix(funcName, "internal_")]; !exists || !tracking {
			return
		}
		g.SynchronizedSend("finish")
		result := g.SynchronizedSend("-stack-list-variables 1")
		comm_s, _ := extractVariableFromResult(result, "comm")
		comm, _ := strconv.Atoi(comm_s)
		rank_s, _ := extractVariableFromResult(result, "rank")
		rank, _ := strconv.Atoi(rank_s)
		if comm != MPI_COMM_WORLD {
			return
		}
		result = g.SynchronizedSend("-stack-list-frames")
		c := CollectiveInfo{
			rank,
			getFileAndLineFromResult(result),
			strings.TrimPrefix(funcName, "internal_"),
		}
		g.cInfoChan <- c
		g.SynchronizedSend("continue")
	}
}

// Send a command and wait for it to complete in gdb.
// This means that for an async command, we will wait till we
// get a stopped notification.
func (g *GdbInstance) SynchronizedSend(operation string, arguments ...string) map[string]interface{} {
	result, err := g.internal.Send(operation, arguments...)
	CheckError(err)
	g.handleNotifications(result)
	return result
}

// Helper/Utility.

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

func analyzeStoppedProcess(payload map[string]interface{}) (isBkpt bool, funcName string, bkptNo int) {
	reason_, ok := payload["reason"]

	if !ok {
		return false, "", -1
	}

	reason := reason_.(string)
	if reason != "breakpoint-hit" {
		return false, "", -1
	}

	frame := payload["frame"].(map[string]interface{})
	isBkpt = true
	funcName = frame["func"].(string)
	bkptNo, _ = strconv.Atoi(payload["bkptno"].(string))
	return
}

func extractVariableFromResult(result map[string]interface{}, varname string) (string, bool) {
	payload := result["payload"].(map[string]interface{})
	variables := payload["variables"].([]interface{})
	for _, variable_ := range variables {
		variable := variable_.(map[string]interface{})
		if variable["name"].(string) == varname {
			return variable["value"].(string), true
		}
	}
	return "nil", false
}

func getFileAndLineFromResult(result map[string]interface{}) string {
	payload := result["payload"].(map[string]interface{})
	stack := payload["stack"].([]interface{})
	// We will look only at the 0th frame!
	top := stack[1].(map[string]interface{})
	frame := top["frame"].(map[string]interface{})
	file := frame["file"].(string)
	line := frame["line"].(string)
	return fmt.Sprintf("%s:%s", file, line)
}
