package utils

import (
	"bufio"
	// "encoding/json"
	"fmt"
	"strings"
	"io"
	"log"
	"os"

	"github.com/milindl/gdb"
)

type GdbInstance struct {
	breakpointHitNotification chan int
	pdFilename string
	internal *gdb.Gdb
	hooks map[string]func(notification map[string]interface{}) bool
}


func (g *GdbInstance) AddNotificationHook(hookName string, hook func(notification map[string]interface{}) bool) {
	g.hooks[hookName] = hook
}

func (g* GdbInstance) RemoveNotificationHook(hookName string) {
	delete(g.hooks, hookName)
}

func (g *GdbInstance) handleNotifications(notification map[string]interface{}) {
	if notification["class"] == "library-loaded" || notification["class"] == "library-unloaded" {
		return
	}
	// jsonStr, _ := json.Marshal(notification)
	// fmt.Println(string(jsonStr))
	switch notification["class"] {
	case "done":
	case "running":
		if len(notification) == 1 {
			<-g.breakpointHitNotification
		}
	case "stopped":
		g.breakpointHitNotification <- 1
	case "error":
		fmt.Println("TODO: Error handling")
	}

	// Run any custom hooks
	for _, hook := range g.hooks {
		result := hook(notification)
		if result == false {
			break
		}
	}
}

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

// InitGDB initializes the GDB interpreter
func NewGdb() (g *GdbInstance) {
	// start a new instance and pipe the target output to stdout
	g = new(GdbInstance)
	g.hooks = make(map[string]func(notification map[string]interface{}) bool)
	g.breakpointHitNotification = make(chan int)
	g.internal, _ = gdb.New(g.handleNotifications)

	return
}

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
	// g.SynchronizedSend("continue")
	return g.pdFilename
}

func (g* GdbInstance) ProcessCommands(r io.Reader, processCommandsDone chan bool) {
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
			g.SynchronizedSend(lineSplit[1])
		}
	}

	processCommandsDone <- true
}
