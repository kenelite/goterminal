package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/creack/pty"
	"github.com/muesli/termenv"
)

// TerminalView wraps a PTY-backed shell session.
type TerminalView struct {
	ptmx   *os.File
	grid   *widget.TextGrid
	scroll *container.Scroll
}

// NewTerminalView starts `shellCmd` in a pty and returns its view.
func NewTerminalView(shellCmd string) *TerminalView {
	tv := &TerminalView{
		grid: widget.NewTextGrid(),
	}
	tv.grid.SetText("Starting " + shellCmd + "...\n")
	tv.scroll = container.NewVScroll(tv.grid)

	cmd := exec.Command(shellCmd)
	var err error
	tv.ptmx, err = pty.Start(cmd)
	if err != nil {
		log.Fatalf("pty.Start failed: %v", err)
	}

	// Read loop
	go func() {
		parser := termenv.NewOutput(io.Discard) // strips ANSI
		r := bufio.NewReader(tv.ptmx)
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					log.Println("PTY read error:", err)
				}
				break
			}
			plain := parser.String(line).String()
			fyne.Do(func() {
				tv.grid.SetText(tv.grid.Text() + plain)
				tv.scroll.ScrollToBottom()
			})
		}
	}()

	return tv
}

// Widget returns the scrollable output area.
func (tv *TerminalView) Widget() fyne.CanvasObject {
	return tv.scroll
}

func main() {
	myApp := app.New()
	w := myApp.NewWindow("goterminal")
	w.Resize(fyne.NewSize(1024, 768))

	// Tabs and views
	tabs := container.NewAppTabs()
	tabs.SetTabLocation(container.TabLocationTop)
	var views []*TerminalView

	// Helper to add a new shell tab
	addShell := func() {
		idx := len(views) + 1
		tv := NewTerminalView("/bin/zsh")
		views = append(views, tv)
		tabs.Append(container.NewTabItem(fmt.Sprintf("Shell %d", idx), tv.Widget()))
		tabs.SelectIndex(idx - 1)
	}

	// Start first tab
	addShell()

	// Capture all key/rune on the window and forward to active PTY
	w.Canvas().SetOnTypedRune(func(r rune) {
		tv := views[tabs.CurrentTabIndex()]
		tv.ptmx.Write([]byte(string(r)))
	})
	w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		tv := views[tabs.CurrentTabIndex()]
		switch ev.Name {
		case fyne.KeyEnter:
			tv.ptmx.Write([]byte{'\r'})
		case fyne.KeyBackspace:
			tv.ptmx.Write([]byte{0x7f})
		}
	})

	// “+” button to spawn new tabs
	newTab := widget.NewButton("+", addShell)

	// Layout
	w.SetContent(container.NewBorder(nil, nil, newTab, nil, tabs))
	w.ShowAndRun()
}
