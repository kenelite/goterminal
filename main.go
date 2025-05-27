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

// TerminalView encapsulates one shell session with output & input widgets.
type TerminalView struct {
	grid   *widget.TextGrid
	scroll *container.Scroll
	input  *widget.Entry
	ptmx   *os.File
}

// NewTerminalView starts a shell PTY and returns a TerminalView.
func NewTerminalView(shell string) *TerminalView {
	tv := &TerminalView{
		grid: widget.NewTextGrid(),
	}
	tv.grid.SetText(fmt.Sprintf("Starting %s...\n", shell))
	tv.scroll = container.NewVScroll(tv.grid)
	tv.scroll.SetMinSize(fyne.NewSize(800, 500))

	tv.input = widget.NewEntry()
	tv.input.SetPlaceHolder("Type a command and press Enter")

	// Launch the PTY
	cmd := exec.Command(shell)
	var err error
	tv.ptmx, err = pty.Start(cmd)
	if err != nil {
		log.Fatalf("Failed to start PTY: %v", err)
	}

	// Read loop: capture all shell output
	go func() {
		reader := bufio.NewReader(tv.ptmx)
		ansi := termenv.NewOutput(io.Discard)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					log.Println("PTY read error:", err)
				}
				break
			}
			plain := ansi.String(line).String() // strip ANSI for now
			fyne.Do(func() {
				tv.grid.SetText(tv.grid.Text() + plain)
				tv.scroll.ScrollToBottom()
			})
		}
	}()

	// Send input commands into the shell
	tv.input.OnSubmitted = func(cmdLine string) {
		// Show the command in the grid (like a prompt echo)
		fyne.Do(func() {
			tv.grid.SetText(tv.grid.Text() + "> " + cmdLine + "\n")
			tv.scroll.ScrollToBottom()
		})
		// Write to PTY
		_, _ = tv.ptmx.Write([]byte(cmdLine + "\n"))
		tv.input.SetText("")
	}

	return tv
}

// Widget returns a container with output scroll + input bar.
func (tv *TerminalView) Widget() fyne.CanvasObject {
	return container.NewBorder(nil, tv.input, nil, nil, tv.scroll)
}

func main() {
	myApp := app.New()
	w := myApp.NewWindow("goterminal")
	w.Resize(fyne.NewSize(1024, 768))

	// Tab container
	tabs := container.NewAppTabs()
	tabs.SetTabLocation(container.TabLocationTop)

	// Add initial shell tab
	tabs.Append(container.NewTabItem("Shell 1", NewTerminalView("/bin/zsh").Widget()))

	// Button to add more shells
	newTabBtn := widget.NewButton("+", func() {
		num := len(tabs.Items) + 1
		title := fmt.Sprintf("Shell %d", num)
		tabs.Append(container.NewTabItem(title, NewTerminalView("/bin/zsh").Widget()))
		tabs.SelectIndex(len(tabs.Items) - 1)
	})

	// Layout: tabs with “+” on left
	w.SetContent(container.NewBorder(nil, nil, newTabBtn, nil, tabs))
	w.ShowAndRun()
}
