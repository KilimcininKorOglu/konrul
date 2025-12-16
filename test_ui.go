// +build ignore

package main

import (
	"log"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func main() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	p := widgets.NewParagraph()
	p.Text = "HELLO WORLD - Press q to quit"
	p.SetRect(0, 0, 50, 5)

	ui.Render(p)

	for e := range ui.PollEvents() {
		if e.ID == "q" || e.ID == "<C-c>" {
			return
		}
		if e.Type == ui.ResizeEvent {
			ui.Render(p)
		}
	}
}
