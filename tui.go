package main

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rivo/tview"
)

type tui struct {
	app               *tview.Application
	server            *server
	maxTubeNameLength int
	mainList          *tview.List
	visibleTubes      map[string]bool
	cli               *cli
}

func NewTui(server *server, cli *cli) *tui {
	app := tview.NewApplication()
	t := &tui{
		server:            server,
		maxTubeNameLength: 0,
		visibleTubes:      make(map[string]bool),
		cli:               cli,
		app:               app,
	}

	list := tview.NewList()
	list.ShowSecondaryText(false)
	t.mainList = list

	cliOutput := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	cliOutput.SetChangedFunc(func() {
		app.Draw()
	})
	t.cli.shell.SetOut(cliOutput)

	grid := tview.NewGrid().
		SetColumns(0, 0).
		SetRows(0).
		SetBorders(true).
		AddItem(t.mainList, 0, 0, 1, 1, 0, 0, true).
		AddItem(cliOutput, 0, 1, 1, 1, 0, 0, false)

	app.SetRoot(grid, true)
	t.app = app
	return t
}

func (t *tui) BuildListRow(tube string, stats map[string]string) string {
	var rowParts []string

	// Add tube name cell
	namePadding := strings.Repeat(" ", t.maxTubeNameLength-len(tube))
	rowParts = append(rowParts, fmt.Sprintf("%s%s ", tube, namePadding))

	// Add state cells
	for _, state := range []string{"ready", "delayed", "buried"} {
		value := stats[fmt.Sprintf("current-jobs-%s", state)]
		width := GetCellWidth(value)
		if width > 0 {
			colourName := getStateColourName(state)
			// Right-align the value with padding
			paddedValue := fmt.Sprintf("%*s ", width, value)
			rowParts = append(rowParts, fmt.Sprintf("[white:%s]%s[-:-:-]", colourName, paddedValue))
		}
	}

	rowParts = append(rowParts, " ")
	return strings.Join(rowParts, "")
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

func getStateColourName(state string) string {
	switch state {
	case "ready":
		return "darkgreen"
	case "delayed":
		return "darkgoldenrod"
	case "buried":
		return "darkred"
	default:
		return "white"
	}
}

func GetCellWidth(contents string) int {
	val, _ := strconv.Atoi(contents)
	if val == 0 {
		return 0
	}
	log := math.Log2(float64(val))
	return int(math.Max(float64(len(contents)), log)) + 1
}

func (t *tui) Run() {
	go func() {
		t.Update()

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			t.Update()
		}
	}()

	if err := t.app.Run(); err != nil {
		panic(err)
	}
}

func (t *tui) Update() {
	tubeStats, err := t.server.GetTubeStats()
	if err != nil {
		panic(err)
	}

	t.app.QueueUpdateDraw(func() {
		// Add tubes with jobs to the set of visible tubes
		for tube, stats := range tubeStats {
			if t.shouldBeVisible(stats) {
				t.visibleTubes[tube] = true
			}
		}

		// Create a sorted list of all tubes that have ever been visible
		var tubesToDisplay []string
		for tube := range t.visibleTubes {
			tubesToDisplay = append(tubesToDisplay, tube)
		}
		sort.Strings(tubesToDisplay)

		// Calculate max tube name length
		newMaxLength := 0
		for _, tube := range tubesToDisplay {
			if len(tube) > newMaxLength {
				newMaxLength = len(tube)
			}
		}
		t.maxTubeNameLength = newMaxLength

		// Preserve selection
		selectedTubeIndex := t.mainList.GetCurrentItem()
		var selectedTubeText string
		if selectedTubeIndex >= 0 && t.mainList.GetItemCount() > 0 && selectedTubeIndex < t.mainList.GetItemCount() {
			selectedTubeName, _ := t.mainList.GetItemText(selectedTubeIndex)
			selectedTubeText = selectedTubeName
		}

		t.mainList.Clear()

		newSelectedTubeIndex := -1
		for i, tube := range tubesToDisplay {
			stats, ok := tubeStats[tube]
			if !ok {
				// Tube is now empty, but we should still show it to prevent later list items from jumping around
				stats = map[string]string{
					"current-jobs-ready":   "0",
					"current-jobs-delayed": "0",
					"current-jobs-buried":  "0",
				}
			}
			rowString := t.BuildListRow(tube, stats)
			t.mainList.AddItem(rowString, "", 0, nil)
			if rowString == selectedTubeText {
				newSelectedTubeIndex = i
			}
		}

		if newSelectedTubeIndex != -1 {
			t.mainList.SetCurrentItem(newSelectedTubeIndex)
		} else if selectedTubeIndex < t.mainList.GetItemCount() {
			t.mainList.SetCurrentItem(selectedTubeIndex)
		}
	})
}
