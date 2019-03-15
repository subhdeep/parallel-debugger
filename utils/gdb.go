package utils

import (
	"bufio"
	// "encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/milindl/gdb"
)

type GdbInstance struct {
	breakpointHitNotification chan int
	pdFilename                string
	internal                  *gdb.Gdb
	hooks                     map[string]func(notification map[string]interface{}) bool
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
	// case "error":
	// 	jsonStr, _ := json.Marshal(notification)
	// 	fmt.Println(string(jsonStr))
	}

	// Run any custom hooks
	for _, hook := range g.hooks {
		// We don't really use the bool returned anywhere yet.
		hook(notification)
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

// NewGdb creates a new GdbInstance struct.
func NewGdb() (g *GdbInstance) {
	// start a new instance and pipe the target output to stdout
	g = new(GdbInstance)
	g.hooks = make(map[string]func(notification map[string]interface{}) bool)
	g.breakpointHitNotification = make(chan int)
	g.internal, _ = gdb.New(g.handleNotifications)

	return
}

// InitGdb does the following:
// 1. Mirror stdout of the target program to stdout of the go program as well as the server.
// 2. Add LD_PRELOAD with the shared library file.
// 3. Run the code uptill MPI_Init, and write data about rank, size to pdFilename.
func (g *GdbInstance) InitGdb(debugTarget string, outputDest io.Writer) (pdFilename string) {
	go io.Copy(os.Stdout, g.internal)
	go io.Copy(outputDest, g.internal)

	fname := getSoFilepath()
	g.pdFilename = fmt.Sprintf("/tmp/pd_init_data_%d", os.Getpid())

	g.SynchronizedSend("set breakpoint pending on")
	g.SynchronizedSend("file", debugTarget)
	g.SynchronizedSend(fmt.Sprintf("set exec-wrapper env 'LD_PRELOAD=%s' 'FILENAME=%s'", fname, g.pdFilename))
	g.SynchronizedSend("break PMPI_Init")
	g.SynchronizedSend("run")
	g.SynchronizedSend("finish")
	g.SynchronizedSend("finish")
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
		}
	}

	processCommandsDone <- true
}
