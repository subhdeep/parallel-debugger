package tui

import (
	"fmt"
	"net"
	"os"

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
	inDisplay    []int
}

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
	return
}

func (t *TUI) drawClients() {
	var counter int
	for rank := range t.conn {
		if counter == t.numOfClients {
			break
		}
		t.inDisplay = append(t.inDisplay, rank)
		title := fmt.Sprintf("rank-%d", rank)
		box := tui.NewVBox()

		scroller := tui.NewScrollArea(box)
		scroller.SetAutoscrollToBottom(true)

		scrollerBox := tui.NewVBox(scroller)
		scrollerBox.SetBorder(true)
		scrollerBox.SetTitle(title)

		t.clients[rank] = box
		t.clientParent.Append(scrollerBox)
		counter++
	}

	t.root.Append(t.clientParent)
}

func (t *TUI) drawInput() {
	inputBox := tui.NewHBox(t.Input)
	inputBox.SetBorder(true)
	inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)
	t.root.Append(t.Input)
}

func (t *TUI) DrawUI() {
	t.drawClients()
	t.drawInput()
	var err error
	t.ui, err = tui.New(t.root)
	if err != nil {
		panic(err)
	}

	go func() {
		err := t.ui.Run()
		if err != nil {
			panic(err)
		}
	}()
}

func (t *TUI) ShowMessagesAll(message string) {
	t.ui.Update(func() {
		command := tui.NewHBox(
			tui.NewPadder(1, 0, tui.NewLabel(message)),
			tui.NewSpacer(),
		)

		for _, b := range t.inDisplay {
			t.clients[b].Append(command)
		}
	})
}

func (t *TUI) ShowMessagesClient(message string, rank int) {
	t.ui.Update(func() {
		command := tui.NewHBox(
			tui.NewPadder(1, 0, tui.NewLabel(message)),
			tui.NewSpacer(),
		)

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
