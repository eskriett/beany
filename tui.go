package main

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type tui struct {
	app               *tview.Application
	server            *server
	maxTubeNameLength int
	mainList          *tview.List
	peekView          *tview.TextView
	outputView        *tview.TextView
	tubes             []string
	pages             *tview.Pages
}

func NewTui(server *server) *tui {
	app := tview.NewApplication()
	t := &tui{
		server:            server,
		maxTubeNameLength: 0,
		tubes:             make([]string, 0),
		app:               app,
	}

	list := tview.NewList()
	list.ShowSecondaryText(false)
	list.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		t.peekView.Clear()
	})
	t.mainList = list

	peekView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	t.peekView = peekView

	outputView := tview.NewTextView()
	outputView.SetDynamicColors(true)
	t.outputView = outputView

	rightPanel := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(peekView, 0, 1, false).
		AddItem(outputView, 1, 0, false)

	funcKeyMap := map[int]string{
		1:  "Help",
		2:  "Setup",
		3:  "DeleteBuried",
		4:  "DeleteDelayed",
		5:  "DeleteReady",
		6:  "KickJobs",
		7:  "KickAllTubes",
		8:  "Pause",
		9:  "Disconnect",
		10: "Exit",
	}

	var footerParts []string
	for i := 1; i <= 10; i++ {
		footerParts = append(footerParts, fmt.Sprintf("[white:darkgreen]F%d[-:-] %s", i, funcKeyMap[i]))
	}

	footerKeysText := " " + strings.Join(footerParts, " ")
	keysView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(footerKeysText)

	versionView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight).
		SetText("[::b]beany[-:-:-] --tui " + Version + " ")

	footer := tview.NewFlex().
		AddItem(keysView, tview.TaggedStringWidth(footerKeysText), 0, false).
		AddItem(versionView, 0, 1, false)

	grid := tview.NewGrid().
		SetColumns(0, 0).
		SetRows(0, 1).
		SetBorders(true).
		AddItem(t.mainList, 0, 0, 1, 1, 0, 0, true).
		AddItem(rightPanel, 0, 1, 1, 1, 0, 0, false).
		AddItem(footer, 1, 0, 1, 2, 0, 0, false)

	pages := tview.NewPages().
		AddPage("main", grid, true, true)
	t.pages = pages

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF1:
			t.HandleHelp()
		case tcell.KeyF2:
			t.HandleSetup()
		case tcell.KeyF3:
			t.HandleDeleteBuried()
		case tcell.KeyF4:
			t.HandleDeleteDelayed()
		case tcell.KeyF5:
			t.HandleDeleteReady()
		case tcell.KeyF6:
			t.HandleKickJobs()
		case tcell.KeyF7:
			t.HandleKickAllTubes()
		case tcell.KeyF8:
			t.HandlePause()
		case tcell.KeyF9:
			t.HandleDisconnect()
		case tcell.KeyF10:
			t.HandleExit()
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				t.HandleExit()
			case '?':
				t.HandleHelp()
			}
		}
		return event
	})

	app.SetRoot(pages, true)
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

func (t *tui) tubeCurrentlyActive(stats map[string]string) bool {
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
		// Add any new tubes to our list
		for tube, stats := range tubeStats {
			if t.tubeCurrentlyActive(stats) && !slices.Contains(t.tubes, tube) {
				t.tubes = append(t.tubes, tube)
			}
		}
		sort.Strings(t.tubes)

		// Calculate max tube name length
		newMaxLength := 0
		for _, tube := range t.tubes {
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
		for i, tube := range t.tubes {
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

		t.updatePeekView()
	})
}

func (t *tui) updatePeekView() {
	// Peek at the selected tube
	t.peekView.Clear()
	currentSelection := t.mainList.GetCurrentItem()
	if currentSelection >= 0 && currentSelection < len(t.tubes) {
		selectedTubeName := t.tubes[currentSelection]

		var peekContent strings.Builder
		var peekedSomething bool
		for _, state := range []string{"ready", "delayed", "buried"} {
			id, body, err := t.server.Peek(state, selectedTubeName)
			if err == nil {
				peekedSomething = true
				colour := getStateColourName(state)
				peekContent.WriteString(fmt.Sprintf("[white:%s:b]Peek %s[-:-:-]\n", colour, state))
				peekContent.WriteString(fmt.Sprintf("ID: %v\n", id))
				peekContent.WriteString(string(body) + "\n")

				stats, err := t.server.StatsJob(id)
				if err == nil {
					peekContent.WriteString(fmt.Sprintf("[white:%s:b]Stats %s[-:-:-]\n", colour, state))
					for _, key := range sortedMapKeys(stats) {
						if slices.Contains([]string{"age", "ttr", "time-left", "delay"}, key) {
							// Append human-readable duration string e.g. 12m30s to duration fields
							duration, err := time.ParseDuration(stats[key] + "s")
							if err == nil {
								ageHumanReadable := duration.String()
								peekContent.WriteString(fmt.Sprintf("%s: %s [darkgray:](%s)[-:-:-]\n", key, stats[key], ageHumanReadable))
							} else {
								peekContent.WriteString(fmt.Sprintf("%s: %s\n", key, stats[key]))
							}
						} else {
							peekContent.WriteString(fmt.Sprintf("%s: %s\n", key, stats[key]))
						}
					}
					peekContent.WriteString("\n")
				}
			}
		}

		if !peekedSomething {
			t.peekView.SetText(fmt.Sprintf("No jobs to peek in %s.", selectedTubeName))
		} else {
			t.peekView.SetText(peekContent.String())
		}
	} else {
		t.peekView.SetText("Select a tube to peek jobs.")
	}
}

func (t *tui) HandleHelp() {
	if t.pages.HasPage("help") {
		t.pages.RemovePage("help")
		return
	}

	modal := tview.NewModal().
		SetText("Placeholder help"). // TODO
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			t.pages.RemovePage("help")
		})
	t.pages.AddPage("help", modal, true, true)
}

func (t *tui) HandleSetup() {
	// TODO: Implement setup
}

func (t *tui) HandleDeleteBuried() {
	// TODO: Implement delete buried
}

func (t *tui) HandleDeleteDelayed() {
	// TODO: Implement delete delayed
}

func (t *tui) HandleDeleteReady() {
	// TODO: Implement delete ready
}

func (t *tui) HandleKickJobs() {
	currentSelection := t.mainList.GetCurrentItem()
	if currentSelection < 0 || currentSelection >= len(t.tubes) {
		t.OutputError(errors.New("invalid tube selection"))
		return
	}
	currentTube := t.tubes[currentSelection]

	stats, err := t.server.StatsTube(currentTube)
	if err != nil {
		t.OutputError(err)
		return
	}
	toKick, err := strconv.Atoi(stats["current-jobs-buried"])
	if err != nil {
		t.OutputError(err)
		return
	}

	if kicked, err := t.server.Kick(currentTube, toKick); err != nil {
		t.OutputError(err)
	} else {
		t.OutputInfo(fmt.Sprintf("Kicked %v jobs", kicked))
	}
}

func (t *tui) HandleKickAllTubes() {
	totalKickedJobs := 0
	kickedTubes := 0
	for _, tube := range t.tubes {
		stats, err := t.server.StatsTube(tube)
		if err != nil {
			t.OutputError(fmt.Errorf("error getting stats for tube %s: %w", tube, err))
			return
		}
		buriedCount, err := strconv.Atoi(stats["current-jobs-buried"])
		if err != nil {
			t.OutputError(fmt.Errorf("error parsing buried job count for tube %s: %w", tube, err))
			return
		}

		if buriedCount > 0 {
			kickedCount, err := t.server.Kick(tube, buriedCount)
			if err != nil {
				t.OutputError(fmt.Errorf("error kicking jobs in tube %s: %w", tube, err))
				return
			}
			if kickedCount > 0 {
				totalKickedJobs += kickedCount
				kickedTubes++
			}
		}
	}

	t.OutputInfo(fmt.Sprintf("Kicked %d jobs on %d tubes", totalKickedJobs, kickedTubes))
}

func (t *tui) HandlePause() {
	// TODO: Implement pause
}

func (t *tui) HandleDisconnect() {
	// TODO: Implement disconnect
}

func (t *tui) HandleExit() {
	t.app.Stop()
}

func (t *tui) OutputError(err error) {
	t.outputView.SetText(fmt.Sprintf("[red:]%s[-:-:-]", err.Error()))
}

func (t *tui) OutputInfo(info string) {
	t.outputView.SetText(info)
}
