package tui

import (
	"fmt"
	"net"
	"os"
	"sort"

	tui "github.com/marcusolsson/tui-go"
)

type TUI struct {
	root         *tui.Box
	ui           tui.UI
	clients      map[int]*tui.Box
	Input        *tui.Entry
	clientParent *tui.Box
	conn         map[int]*net.Conn
	numOfClients int
	history      map[int][]string
	cmdHistory   []string
	histPtr      int
}

// NewTUI creates a new instance of a TUI
// it initializes the number of clients to be displayed to be as 2
func NewTUI(connections map[int]*net.Conn) (t *TUI) {
	t = new(TUI)
	t.clients = make(map[int]*tui.Box)
	t.clientParent = tui.NewHBox()
	t.Input = tui.NewEntry()
	t.Input.SetFocused(true)
	t.Input.SetSizePolicy(tui.Expanding, tui.Maximum)
	t.root = tui.NewVBox()
	t.conn = connections
	t.numOfClients = 2
	t.history = make(map[int][]string)
	t.histPtr = 0
	return
}

func (t *TUI) drawClient(title string, rank int) *tui.Box {
	box := tui.NewVBox()

	// if history exists
	for _, hist := range t.history[rank] {
		histBox := tui.NewHBox(
			tui.NewPadder(1, 0, tui.NewLabel(hist)),
			tui.NewSpacer(),
		)
		box.Append(histBox)
	}

	scroller := tui.NewScrollArea(box)
	scroller.SetAutoscrollToBottom(true)
	scrollerBox := tui.NewVBox(scroller)
	scrollerBox.SetBorder(true)
	scrollerBox.SetTitle(title)
	t.clients[rank] = box
	return scrollerBox
}

// drawInput draws the input textarea where the user
// can type their command which passes over to the server
func (t *TUI) drawInput() {
	inputBox := tui.NewHBox(t.Input)
	inputBox.SetBorder(true)
	inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)
	t.root.Append(t.Input)
}

// DrawUI paints the complete UI along with the clients and inputBox
func (t *TUI) DrawUI() {
	for i := 0; i < t.numOfClients; i++ {
		box := t.drawClient(fmt.Sprintf("rank-%d", i), i)
		t.clientParent.Append(box)
	}
	t.root.Append(t.clientParent)
	t.drawInput()
	var err error
	t.ui, err = tui.New(t.root)
	if err != nil {
		panic(err)
	}

	// TODO command History
	t.ui.SetKeybinding("Up", func() {
		if len(t.cmdHistory) == 0 {
			return
		}

		if t.histPtr < 0 {
			return
		}
		t.Input.SetText(t.cmdHistory[t.histPtr])
		if t.histPtr != 0 {
			t.histPtr--
		}
	})

	t.ui.SetKeybinding("Down", func() {
		if len(t.cmdHistory) == 0 || t.histPtr == 0 {
			return
		}
		if t.histPtr == len(t.cmdHistory) {
			t.Input.SetText("")
			t.histPtr = len(t.cmdHistory) - 1
			return
		}
		t.Input.SetText(t.cmdHistory[t.histPtr])
		fmt.Fprintln(os.Stderr, t.histPtr, "before")
		if t.histPtr != len(t.cmdHistory) {
			t.histPtr++
		}
		fmt.Fprintln(os.Stderr, t.histPtr, "after")
	})

	go func() {
		err := t.ui.Run()
		if err != nil {
			panic(err)
		}
	}()
}

func (t *TUI) AddToCmdHistory(hist string) {
	t.cmdHistory = append(t.cmdHistory, hist)
	t.histPtr = len(t.cmdHistory) - 1
}

func (t *TUI) reDraw(rank int, cat string) {

	var currClients []int
	for r := range t.clients {
		currClients = append(currClients, r)
	}
	t.clients = make(map[int]*tui.Box)
	for t.clientParent.Length() != 0 {
		t.clientParent.Remove(0)
	}
	if cat == "Add" {
		currClients = append(currClients, rank)
	} else if cat == "Remove" {
		i := -1
		for idx, r := range currClients {
			if r == rank {
				i = idx
				break
			}
		}
		if i != -1 {
			currClients = append(currClients[:i], currClients[i+1:]...)
		}
	}
	t.numOfClients = len(currClients)
	sort.Ints(currClients)
	for _, i := range currClients {
		box := t.drawClient(fmt.Sprintf("rank-%d", i), i)
		t.clientParent.Append(box)
	}

}

func (t *TUI) Add(rank int) {
	// if the box exists dont redraw
	if _, ok := t.clients[rank]; ok {
		return
	}
	t.reDraw(rank, "Add")
}

func (t *TUI) Remove(rank int) {
	// if the box dont exists dont redraw
	if _, ok := t.clients[rank]; !ok {
		return
	}
	t.reDraw(rank, "Remove")
}

func (t *TUI) Swap(newRank int, oldRank int) {
	t.Remove(oldRank)
	t.Add(newRank)
}

// ShowMessagesAll displays a particular message
// to all the clients in display
func (t *TUI) ShowMessagesAll(message string) {
	t.ui.Update(func() {
		command := tui.NewHBox(
			tui.NewPadder(1, 0, tui.NewLabel(message)),
			tui.NewSpacer(),
		)

		var rank int
		for rank = range t.conn {
			t.history[rank] = append(t.history[rank], message)
		}

		var b *tui.Box
		for rank, b = range t.clients {
			b.Append(command)
		}
	})
}

// ShowMessagesClient the messages of a particular client
// If the specific client is not in display
// the function return void silently
func (t *TUI) ShowMessagesClient(message string, rank int) {
	t.ui.Update(func() {
		command := tui.NewHBox(
			tui.NewPadder(1, 0, tui.NewLabel(message)),
			tui.NewSpacer(),
		)

		t.history[rank] = append(t.history[rank], message)

		var c *tui.Box
		var ok bool
		if c, ok = t.clients[rank]; !ok {
			return
		}
		c.Append(command)
	})

}

func (t *TUI) Quit() {
	t.ui.Quit()
	os.Exit(0)
}
