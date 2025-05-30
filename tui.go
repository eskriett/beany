package main

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/rivo/tview"
	"github.com/gdamore/tcell/v2"
)

type tui struct {
	app *tview.Application
	server *server
	maxTubeNameLength int
	cellRefs map[string]map[string]*tview.TextView
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
		flex.AddItem(t.BuildFlexRow(tube, stats), 1, 0, false)
	}

	t.app = app
	return t
}

func (t *tui) BuildFlexRow(tube string, stats map[string]string) *tview.Flex {
	if t.cellRefs == nil {
		t.cellRefs = make(map[string]map[string]*tview.TextView)
	}
	t.cellRefs[tube] = make(map[string]*tview.TextView)

	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexColumn)

	// Add tube name cell
	tubeCell := tview.NewTextView().SetTextAlign(tview.AlignRight).SetText(fmt.Sprintf("%s ", tube))
	t.cellRefs[tube]["name"] = tubeCell
	flex.AddItem(tubeCell, t.maxTubeNameLength, 0, false)

	// Add state cells
	for _, state := range []string{"ready", "delayed", "buried"} {
		cell := tview.NewTextView()
		cell.SetBackgroundColor(getStateColor(state))
		cell.SetText(fmt.Sprintf("%s", stats[fmt.Sprintf("current-jobs-%s", state)]))
		t.cellRefs[tube][state] = cell
		flex.AddItem(cell, GetCellWidth(stats[fmt.Sprintf("current-jobs-%s", state)]), 0, false)
	}

	return flex
}

func getStateColor(state string) tcell.Color {
	switch state {
	case "ready":
		return tcell.ColorDarkGreen
	case "delayed":
		return tcell.ColorDarkGoldenrod
	case "buried":
		return tcell.ColorDarkRed
	default:
		return tcell.ColorBlack
	}
}

func GetCellWidth(contents string) int {
	val, _ := strconv.Atoi(contents)
	if val == 0 {
		return 0
	}
	ln := math.Log(float64(val))
	return int(math.Max(float64(len(contents)), ln)) + 1
}

func (t *tui) Run() {
	t.StartPeriodicUpdates()
	if err := t.app.Run(); err != nil {
		panic(err)
	}
}

func (t *tui) StartPeriodicUpdates() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			t.app.QueueUpdateDraw(func() {
				tubeStats, err := t.server.GetTubeStats()
				if err != nil {
					panic(err)
				}

				for tube, _ := range tubeStats {
					if len(tube) > t.maxTubeNameLength {
						t.maxTubeNameLength = len(tube) + 1 // TODO if this increases, we don't have a way of increasing the width of the tube name cell
					}
				}

				for tube, stats := range tubeStats {
					t.UpdateRow(tube, stats)
				}
			})
		}
	}()
}

func (t *tui) UpdateRow(tube string, stats map[string]string) {
	for _, state := range []string{"ready", "delayed", "buried"} {
		tv := t.cellRefs[tube][state]
		tv.SetText(fmt.Sprintf("%s", stats[fmt.Sprintf("current-jobs-%s", state)]))
		tv.SetBackgroundColor(getStateColor(state))
		// TODO also update the width of the cell
	}
}
