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
}

func NewTui(server *server) *tui {
	t := &tui{
		server:            server,
		maxTubeNameLength: 0,
	}
	app := tview.NewApplication()
	pages := tview.NewPages()
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle("beany")
	t.mainList = list
	pages.AddPage("table", list, true, true)
	app.SetRoot(pages, true)

	t.app = app
	return t
}

func (t *tui) BuildListRow(tube string, stats map[string]string) string {
	var rowParts []string

	// Add tube name cell
	namePadding := strings.Repeat(" ", t.maxTubeNameLength-len(tube))
	rowParts = append(rowParts, fmt.Sprintf("%s%s", tube, namePadding))

	// Add state cells
	for _, state := range []string{"ready", "delayed", "buried"} {
		value := stats[fmt.Sprintf("current-jobs-%s", state)]
		width := GetCellWidth(value)
		if width > 0 {
			colourName := getStateColourName(state)
			// Right-align the value with padding
			paddedValue := fmt.Sprintf("%*s", width, value)
			rowParts = append(rowParts, fmt.Sprintf("[:%s]%s[-:-:-]", colourName, paddedValue))
		}
	}

	return strings.Join(rowParts, " ")
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
		// Calculate the max length for the tube name column
		newMaxLength := 0
		for tube := range tubeStats {
			if len(tube) > newMaxLength {
				newMaxLength = len(tube)
			}
		}
		t.maxTubeNameLength = newMaxLength

		// Filter for visible tubes and sort them
		var visibleTubes []string
		for tube, stats := range tubeStats {
			if t.shouldBeVisible(stats) {
				visibleTubes = append(visibleTubes, tube)
			}
		}
		sort.Strings(visibleTubes)

		// Preserve selection
		selectedTubeIndex := t.mainList.GetCurrentItem()
		var selectedTubeText string
		if selectedTubeIndex >= 0 && t.mainList.GetItemCount() > 0 && selectedTubeIndex < t.mainList.GetItemCount() {
			main, _ := t.mainList.GetItemText(selectedTubeIndex)
			selectedTubeText = main
		}

		t.mainList.Clear()

		newSelectedTubeIndex := -1
		for i, tube := range visibleTubes {
			stats := tubeStats[tube]
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
