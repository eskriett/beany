package main

import (
	"fmt"
	"math"
	"strconv"

	"github.com/rivo/tview"
	// "github.com/gdamore/tcell/v2"
)

type tui struct {
	app *tview.Application
	server *server
	maxTubeNameLength int
}

func NewTui(server *server) *tui {
	t := &tui{
		server: server,
		maxTubeNameLength: 0,
	}
	app := tview.NewApplication()
	pages := tview.NewPages()
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	flex.SetBorder(true)
	flex.SetTitle("beany")
	pages.AddPage("table", flex, true, true)
	app.SetRoot(pages, true)

	tubeStats, err := server.GetTubeStats()
	if err != nil {
		panic(err)
	}

	for tube, _ := range tubeStats {
		if len(tube) > t.maxTubeNameLength {
			t.maxTubeNameLength = len(tube) + 1
		}
	}

	for tube, stats := range tubeStats {
		flex.AddItem(t.GetFlexRow(tube, stats), 1, 0, false)
	}

	t.app = app
	return t
}

func (t *tui) GetFlexRow(tube string, stats map[string]string) *tview.Flex {
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexColumn)
	flex.AddItem(tview.NewTextView().SetText(fmt.Sprintf("%s", tube)), t.maxTubeNameLength, 0, false)
	flex.AddItem(tview.NewTextView().SetText(fmt.Sprintf("%s", stats["current-jobs-ready"])), GetCellWidth(stats["current-jobs-ready"]), 0, false)
	flex.AddItem(tview.NewTextView().SetText(fmt.Sprintf("%s", stats["current-jobs-delayed"])), GetCellWidth(stats["current-jobs-delayed"]), 0, false)
	flex.AddItem(tview.NewTextView().SetText(fmt.Sprintf("%s", stats["current-jobs-buried"])), GetCellWidth(stats["current-jobs-buried"]), 0, false)
	return flex
}

func GetCellWidth(contents string) int {
	val, _ := strconv.Atoi(contents)
	ln := math.Log(float64(val))
	return int(math.Max(float64(len(contents)), ln)) + 1
}

func (t *tui) Run() {
	if err := t.app.Run(); err != nil {
		panic(err)
	}
}
