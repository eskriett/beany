package main

import (
	"github.com/rivo/tview"
)

type tui struct {
	app *tview.Application
	server *server
}

func NewTui(server *server) *tui {
	app := tview.NewApplication()
	pages := tview.NewPages()
	pages.AddPage("cli", tview.NewBox().SetBorder(true).SetTitle("beany cli"), true, true)
	pages.AddPage("table", tview.NewTable().SetBorder(true).SetTitle("beany"), true, true)
	app.SetRoot(pages, true)

	return &tui{
		app:    app,
		server: server,
	}
}

func (t *tui) Run() {
	if err := t.app.Run(); err != nil {
		panic(err)
	}
}
