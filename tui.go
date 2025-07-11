package main

import (
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
	tubes             []string
	cli               *cli
	cliOutput         *tview.TextView
}

func NewTui(server *server, cli *cli) *tui {
	app := tview.NewApplication()
	t := &tui{
		server:            server,
		maxTubeNameLength: 0,
		tubes:             make([]string, 0),
		cli:               cli,
		app:               app,
	}

	list := tview.NewList()
	list.ShowSecondaryText(false)
	list.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		t.peekView.Clear()
	})
	t.mainList = list

	cliOutput := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	cliOutput.SetChangedFunc(func() {
		app.Draw()
	})
	t.cliOutput = cliOutput
	t.cli.shell.SetOut(cliOutput)

	peekView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	t.peekView = peekView

	rightPane := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(peekView, 0, 1, false).
		AddItem(cliOutput, 10, 1, false)

	funcKeyMap := map[int]string{
		1:  "Help",
		2:  "Setup",
		3:  "DeleteBuried",
		4:  "DeleteDelayed",
		5:  "DeleteReady",
		6:  "Kick",
		7:  "KickAll",
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
		AddItem(rightPane, 0, 1, 1, 1, 0, 0, false).
		AddItem(footer, 1, 0, 1, 2, 0, 0, false)

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
			t.HandleKick()
		case tcell.KeyF7:
			t.HandlePause()
		case tcell.KeyF11:
			t.HandleDisconnect()
		case tcell.KeyF12:
			t.HandleExit()
		}
		return event
	})

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
				peekContent.WriteString(string(body) + "\n\n")
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
	fmt.Fprintf(t.cliOutput, "called Help\n")
}

func (t *tui) HandleSetup() {
	fmt.Fprintf(t.cliOutput, "called Setup\n")
}

func (t *tui) HandleDeleteBuried() {
	fmt.Fprintf(t.cliOutput, "called DeleteBuried\n")
}

func (t *tui) HandleDeleteDelayed() {
	fmt.Fprintf(t.cliOutput, "called DeleteDelayed\n")
}

func (t *tui) HandleDeleteReady() {
	fmt.Fprintf(t.cliOutput, "called DeleteReady\n")
}

func (t *tui) HandleKick() {
	fmt.Fprintf(t.cliOutput, "called Kick\n")
}

func (t *tui) HandleKickAll() {
	fmt.Fprintf(t.cliOutput, "called KickAll\n")
}

func (t *tui) HandlePause() {
	fmt.Fprintf(t.cliOutput, "called Pause\n")
}

func (t *tui) HandleDisconnect() {
	fmt.Fprintf(t.cliOutput, "called Disconnect\n")
}

func (t *tui) HandleExit() {
	fmt.Fprintf(t.cliOutput, "called Exit\n")
}
