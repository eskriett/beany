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
	app               *tview.Application
	server            *server
	maxTubeNameLength int
	cellRefs          map[string]map[string]*tview.TextView
	rowFlexes         map[string]*tview.Flex
	mainFlex          *tview.Flex
	visibleTubes      map[string]bool
}

func NewTui(server *server) *tui {
	t := &tui{
		server:            server,
		maxTubeNameLength: 0,
		rowFlexes:         make(map[string]*tview.Flex),
		visibleTubes:      make(map[string]bool),
	}
	app := tview.NewApplication()
	pages := tview.NewPages()
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	flex.SetBorder(true)
	flex.SetTitle("beany")
	t.mainFlex = flex
	pages.AddPage("table", flex, true, true)
	app.SetRoot(pages, true)

	tubeStats, err := server.GetTubeStats()
	if err != nil {
		panic(err)
	}

	for tube := range tubeStats {
		if len(tube) > t.maxTubeNameLength {
			t.maxTubeNameLength = len(tube) + 1
		}
	}

	for tube, stats := range tubeStats {
		rowFlex := t.BuildFlexRow(tube, stats)
		t.rowFlexes[tube] = rowFlex
		if t.shouldBeVisible(stats) {
			t.mainFlex.AddItem(rowFlex, 1, 0, false)
			t.visibleTubes[tube] = true
		}
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

func (t *tui) shouldBeVisible(stats map[string]string) bool {
	for _, state := range []string{"ready", "delayed", "buried"} {
		val, _ := strconv.Atoi(stats[fmt.Sprintf("current-jobs-%s", state)])
		if val > 0 {
			return true
		}
	}
	return false
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

				// This part handles resizing the name column if a new, longer tube name appears.
				newMaxLength := t.maxTubeNameLength
				for tube := range tubeStats {
					if len(tube)+1 > newMaxLength {
						newMaxLength = len(tube) + 1
					}
				}

				if newMaxLength > t.maxTubeNameLength {
					t.maxTubeNameLength = newMaxLength
					for tube, rowFlex := range t.rowFlexes {
						nameCell := t.cellRefs[tube]["name"]
						rowFlex.ResizeItem(nameCell, newMaxLength, 0)
					}
				}

				for tube, stats := range tubeStats {
					// Handle new tubes that weren't present at startup
					if _, exists := t.rowFlexes[tube]; !exists {
						rowFlex := t.BuildFlexRow(tube, stats)
						t.rowFlexes[tube] = rowFlex
					}

					// Make tubes visible if they become active
					if !t.visibleTubes[tube] && t.shouldBeVisible(stats) {
						t.mainFlex.AddItem(t.rowFlexes[tube], 1, 0, false)
						t.visibleTubes[tube] = true
					}

					t.UpdateRow(tube, stats)
				}
			})
		}
	}()
}

func (t *tui) UpdateRow(tube string, stats map[string]string) {
	rowFlex := t.rowFlexes[tube]
	for _, state := range []string{"ready", "delayed", "buried"} {
		tv := t.cellRefs[tube][state]
		value := stats[fmt.Sprintf("current-jobs-%s", state)]
		tv.SetText(fmt.Sprintf("%s", value))
		tv.SetBackgroundColor(getStateColor(state))
		rowFlex.ResizeItem(tv, GetCellWidth(value), 0)
	}
}
