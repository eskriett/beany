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
	updater           *time.Ticker
	keysView          *tview.TextView
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

	keysView := tview.NewTextView().
		SetDynamicColors(true)
	t.keysView = keysView
	t.setupKeysView()

	versionView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight).
		SetText("[::b]beany[-:-:-] --tui " + Version + " ") // TODO t.server.ConnectionStr() might be useful, would need to make the whole footer updateable

	footer := tview.NewFlex().
		AddItem(keysView, tview.TaggedStringWidth(t.keysView.GetText(true)), 0, false).
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
			t.handleHelp()
		case tcell.KeyF2:
			t.handleSetup()
		case tcell.KeyF3:
			t.handleDeleteBuried()
		case tcell.KeyF4:
			t.handleDeleteDelayed()
		case tcell.KeyF5:
			t.handleDeleteReady()
		case tcell.KeyF6:
			t.handleKickJobs()
		case tcell.KeyF7:
			t.handleKickAllTubes()
		case tcell.KeyF8:
			t.handlePause()
		case tcell.KeyF9:
			t.handleConnectDisconnect()
		case tcell.KeyF10:
			t.handleExit()
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				t.handleExit()
			case '?':
				t.handleHelp()
			}
		}
		return event
	})

	app.SetRoot(pages, true)
	t.app = app
	return t
}

func (t *tui) setupKeysView() {
	connectString := "Connect"
	if t.server.isConnected() {
		connectString = "Disconnect"
	}
	funcKeyMap := map[int]string{
		1:  "Help",
		2:  "Setup",
		3:  "DeleteBuried",
		4:  "DeleteDelayed",
		5:  "DeleteReady",
		6:  "KickJobs",
		7:  "KickAllTubes",
		8:  "Pause",
		9:  connectString,
		10: "Exit",
	}

	var footerParts []string
	for i := 1; i <= 10; i++ {
		footerParts = append(footerParts, fmt.Sprintf("[white:darkgreen]F%d[-:-] %s", i, funcKeyMap[i]))
	}

	footerKeysText := " " + strings.Join(footerParts, " ")
	t.keysView.SetText(footerKeysText)
}

func (t *tui) buildListRow(tube string, stats map[string]string) string {
	var rowBuilder strings.Builder

	// Add tube name cell
	namePadding := strings.Repeat(" ", t.maxTubeNameLength-len(tube))
	tubeNameColour := ""
	if !tubeCurrentlyActive(stats) {
		tubeNameColour = "darkgray"
	}
	if stats["pause-time-left"] != "" && stats["pause-time-left"] != "0" {
		tubeNameColour = "lightblue"
	}
	fmt.Fprintf(&rowBuilder, "[%s:]%s%s[-:-:-] ", tubeNameColour, tube, namePadding)

	// Add state cells
	for _, state := range []string{"ready", "delayed", "buried"} {
		value := stats[fmt.Sprintf("current-jobs-%s", state)]
		width := getCellWidth(value)
		if width > 0 {
			colourName := getStateColourName(state)
			// Right-align the value with padding
			paddedValue := fmt.Sprintf("%*s ", width, value)
			fmt.Fprintf(&rowBuilder, "[white:%s]%s[-:-:-]", colourName, paddedValue)
		}
	}

	rowBuilder.WriteString(" ")

	if stats["pause-time-left"] != "" && stats["pause-time-left"] != "0" {
		fmt.Fprintf(&rowBuilder, "[lightblue:]* PAUSED: %s *[-:-:-]", secondsToHumanReadable(stats["pause-time-left"]))
	}

	return rowBuilder.String()
}

func tubeCurrentlyActive(stats map[string]string) bool {
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

func getCellWidth(contents string) int {
	val, _ := strconv.Atoi(contents)
	if val == 0 {
		return 0
	}
	log := math.Log2(float64(val))
	return int(math.Max(float64(len(contents)), log)) + 1
}

func (t *tui) Run() {
	go func() {
		t.update()

		t.updater = time.NewTicker(1 * time.Second)
		defer t.updater.Stop()

		for range t.updater.C {
			t.update()
		}
	}()

	if err := t.app.Run(); err != nil {
		panic(err)
	}
}

func (t *tui) update() {
	if !t.server.isConnected() {
		t.outputError(errors.New("not connected to a beanstalk server"))
		return
	}

	tubeStats, err := t.server.GetTubeStats()
	if err != nil {
		panic(err)
	}

	t.app.QueueUpdateDraw(func() {
		// This check prevents a race condition where an update is queued
		// right before a disconnect, which would otherwise cause the list
		// to be repopulated after being cleared in HandleDisconnect.
		if !t.server.isConnected() {
			return
		}

		// Preserve selection by getting the name of the current tube.
		// This must be done before t.tubes is modified.
		selectedTubeName, _ := t.getCurrentTube()

		// Add any new tubes to our list
		for tube, stats := range tubeStats {
			if tubeCurrentlyActive(stats) && !slices.Contains(t.tubes, tube) {
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
			rowString := t.buildListRow(tube, stats)
			t.mainList.AddItem(rowString, "", 0, nil)
			if tube == selectedTubeName {
				newSelectedTubeIndex = i
			}
		}

		if newSelectedTubeIndex != -1 {
			t.mainList.SetCurrentItem(newSelectedTubeIndex)
		}

		t.updatePeekView()
	})
}

func (t *tui) updatePeekView() {
	// Peek at the selected tube
	t.peekView.Clear()
	currentTube, err := t.getCurrentTube()
	if err != nil {
		t.outputError(err)
		return
	}

	var peekContent strings.Builder
	var peekedSomething bool
	for _, state := range []string{"ready", "delayed", "buried"} {
		id, body, err := t.server.Peek(state, currentTube)
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
						peekContent.WriteString(fmt.Sprintf(
							"%s: %s [darkgray:](%s)[-:-:-]\n",
							key, stats[key], secondsToHumanReadable(stats[key]),
						))
					} else {
						peekContent.WriteString(fmt.Sprintf("%s: %s\n", key, stats[key]))
					}
				}
				peekContent.WriteString("\n")
			}
		}
	}

	if !peekedSomething {
		t.peekView.SetText(fmt.Sprintf("No jobs to peek in %s.", currentTube))
	} else {
		t.peekView.SetText(peekContent.String())
	}
}

func secondsToHumanReadable(seconds string) string {
	duration, err := time.ParseDuration(seconds + "s")
	if err == nil {
		return duration.String()
	} else {
		return "" // This error doesn't matter, just omit the human readable duration
	}
}

func (t *tui) getCurrentTube() (string, error) {
	currentSelection := t.mainList.GetCurrentItem()
	if currentSelection < 0 || currentSelection >= len(t.tubes) {
		return "", errors.New("no tube selected")
	}
	return t.tubes[currentSelection], nil
}

func (t *tui) handleHelp() {
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

func (t *tui) handleSetup() {
	// TODO: Implement setup
}

func (t *tui) handleDeleteBuried() {
	// TODO: Implement delete buried
	t.handleDeleteAll("buried")
}

func (t *tui) handleDeleteDelayed() {
	// TODO: Implement delete delayed
	t.handleDeleteAll("delayed")
}

func (t *tui) handleDeleteReady() {
	// TODO: Implement delete ready
	t.handleDeleteAll("ready") // TODO this works but shows an error "peek-ready: not found"
}

func (t *tui) handleDeleteAll(state string) {
	currentTube, err := t.getCurrentTube()
	if err != nil {
		t.outputError(err)
		return
	}

	stats, err := t.server.StatsTube(currentTube)
	if err != nil {
		t.outputError(err)
		return
	}
	expectedCount, err := strconv.Atoi(stats[fmt.Sprintf("current-jobs-%s", state)])
	if err != nil {
		t.outputError(err)
		return
	}

	if expectedCount == 0 {
		t.outputInfo(fmt.Sprintf("No %s jobs to delete", state))
		return
	}

	actualCount, err := t.server.DeleteAll(state, currentTube)
	if err != nil {
		t.outputError(err)
		return
	}

	t.outputInfo(fmt.Sprintf("Deleted %d %s jobs", actualCount, state))
}

func (t *tui) handleKickJobs() {
	currentTube, err := t.getCurrentTube()
	if err != nil {
		t.outputError(err)
		return
	}

	stats, err := t.server.StatsTube(currentTube)
	if err != nil {
		t.outputError(err)
		return
	}
	toKick, err := strconv.Atoi(stats["current-jobs-buried"])
	if err != nil {
		t.outputError(err)
		return
	}

	if toKick == 0 {
		toKick, err = strconv.Atoi(stats["current-jobs-delayed"])
		if err != nil {
			t.outputError(err)
			return
		}
	}

	if kicked, err := t.server.Kick(currentTube, toKick); err != nil {
		t.outputError(err)
	} else {
		t.outputInfo(fmt.Sprintf("Kicked %v jobs", kicked))
	}
}

func (t *tui) handleKickAllTubes() {
	totalKickedJobs := 0
	kickedTubes := 0
	for _, tube := range t.tubes {
		stats, err := t.server.StatsTube(tube)
		if err != nil {
			t.outputError(fmt.Errorf("error getting stats for tube %s: %w", tube, err))
			return
		}
		buriedCount, err := strconv.Atoi(stats["current-jobs-buried"])
		if err != nil {
			t.outputError(fmt.Errorf("error parsing buried job count for tube %s: %w", tube, err))
			return
		}

		if buriedCount > 0 {
			kickedCount, err := t.server.Kick(tube, buriedCount)
			if err != nil {
				t.outputError(fmt.Errorf("error kicking jobs in tube %s: %w", tube, err))
				return
			}
			if kickedCount > 0 {
				totalKickedJobs += kickedCount
				kickedTubes++
			}
		}
	}

	t.outputInfo(fmt.Sprintf("Kicked %d jobs on %d tubes", totalKickedJobs, kickedTubes))
}

func (t *tui) handlePause() {
	currentTube, err := t.getCurrentTube()
	if err != nil {
		t.outputError(err)
		return
	}

	if t.pages.HasPage("pause") {
		t.pages.RemovePage("pause")
		return
	}

	modal := tview.NewModal().
		SetText("Pause tube to prevent jobs from being reserved").
		AddButtons([]string{"Cancel", "Pause for 1 hour", "Unpause"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 1 {
				if err := t.server.Pause(currentTube, 3600*time.Second); err != nil {
					t.outputError(err)
					return
				}
				t.outputInfo(fmt.Sprintf("Paused tube %s for 1 hour", currentTube))
			} else if buttonIndex == 2 {
				if err := t.server.Unpause(currentTube); err != nil {
					t.outputError(err)
					return
				}
				t.outputInfo(fmt.Sprintf("Unpaused tube %s", currentTube))
			}
			t.pages.RemovePage("pause")
		})
	t.pages.AddPage("pause", modal, true, true)
}

func (t *tui) handleConnectDisconnect() {
	if t.pages.HasPage("connect") {
		t.pages.RemovePage("connect")
		return
	}

	if t.server.isConnected() {
		if err := t.server.Disconnect(); err != nil {
			t.outputError(err)
		} else {
			t.outputInfo("Disconnected from beanstalk server. Press F9 to connect.")
			t.updater.Stop()
			t.mainList.Clear()
			t.mainList.AddItem("Not connected to a beanstalk server. Press F9 to connect.", "", 0, nil)
			t.peekView.Clear()
			t.tubes = make([]string, 0)
			t.setupKeysView()
		}
	} else {
		form := tview.NewForm()
		form.AddInputField("Host (defaults to localhost)", "", 20, nil, nil).
			AddInputField("Port (defaults to 11300)", "", 6, nil, nil).
			AddButton("Connect", func() {
				t.pages.RemovePage("connect")
				t.outputInfo("Connecting...")

				host := form.GetFormItem(0).(*tview.InputField).GetText()
				if host == "" {
					host = "localhost"
				}
				portStr := form.GetFormItem(1).(*tview.InputField).GetText()
				if portStr == "" {
					portStr = "11300"
				}

				go func() {
					port, err := strconv.Atoi(portStr)
					if err != nil {
						t.app.QueueUpdateDraw(func() {
							t.outputError(fmt.Errorf("invalid port: %w", err))
						})
						return
					}

					if err := t.server.Connect(host, port); err != nil {
						t.app.QueueUpdateDraw(func() {
							t.outputError(err)
						})
						return
					}

					address := fmt.Sprintf("%s:%d", host, port)
					t.mainList.Clear()
					t.mainList.AddItem("Waiting for data from " + address + "...", "", 0, nil)

					t.app.QueueUpdateDraw(func() {
						t.outputInfo("Connected to " + address)
						t.updater.Reset(1 * time.Second)
						t.setupKeysView()
					})
				}()
			}).
			AddButton("Cancel", func() {
				t.pages.RemovePage("connect")
			})
		form.SetBorder(true).SetTitle("Connect to beanstalkd").SetTitleAlign(tview.AlignLeft)
		t.pages.AddPage("connect", form, true, true)
	}
}

func (t *tui) handleExit() {
	t.app.Stop()
}

func (t *tui) outputError(err error) {
	t.outputView.SetText(fmt.Sprintf("[red:]%s[-:-:-]", err.Error()))
}

func (t *tui) outputInfo(info string) {
	t.outputView.SetText(info)
}
