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
	t.SetupKeysView()

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
			t.HandleConnectDisconnect()
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

func (t *tui) SetupKeysView() {
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

func (t *tui) BuildListRow(tube string, stats map[string]string) string {
	var rowParts []string

	// Add tube name cell
	namePadding := strings.Repeat(" ", t.maxTubeNameLength-len(tube))
	tubeNameColour := ""
	if !tubeCurrentlyActive(stats) {
		tubeNameColour = "darkgray"
	}
	if stats["pause-time-left"] != "0" {
		tubeNameColour = "lightblue"
	}
	rowParts = append(rowParts, fmt.Sprintf("[%s:]%s%s[-:-:-] ", tubeNameColour, tube, namePadding))

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

	if stats["pause-time-left"] != "0" {
		rowParts = append(rowParts, fmt.Sprintf("[lightblue:]* PAUSED: %s *[-:-:-]", secondsToHumanReadable(stats["pause-time-left"])))
	}

	return strings.Join(rowParts, "")
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

		t.updater = time.NewTicker(1 * time.Second)
		defer t.updater.Stop()

		for range t.updater.C {
			t.Update()
		}
	}()

	if err := t.app.Run(); err != nil {
		panic(err)
	}
}

func (t *tui) Update() {
	if !t.server.isConnected() {
		t.OutputError(errors.New("not connected to a beanstalk server"))
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
		selectedTubeName, _ := t.GetCurrentTube()

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
			rowString := t.BuildListRow(tube, stats)
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
	currentTube, err := t.GetCurrentTube()
	if err != nil {
		t.OutputError(err)
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

func (t *tui) GetCurrentTube() (string, error) {
	currentSelection := t.mainList.GetCurrentItem()
	if currentSelection < 0 || currentSelection >= len(t.tubes) {
		return "", errors.New("no tube selected")
	}
	return t.tubes[currentSelection], nil
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
	currentTube, err := t.GetCurrentTube()
	if err != nil {
		t.OutputError(err)
		return
	}

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

	if toKick == 0 {
		toKick, err = strconv.Atoi(stats["current-jobs-delayed"])
		if err != nil {
			t.OutputError(err)
			return
		}
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

func (t *tui) HandlePause() { // TODO pause should be a modal with a duration input field, and an option to unpause
	currentTube, err := t.GetCurrentTube()
	if err != nil {
		t.OutputError(err)
		return
	}

	if err := t.server.Pause(currentTube, 3600*time.Second); err != nil {
		t.OutputError(err)
		return
	}

	t.OutputInfo(fmt.Sprintf("Paused tube %s for 1 hour", currentTube))
}

func (t *tui) HandleConnectDisconnect() {
	// TODO: Implement disconnect - it doesn't clear the main list properly, and needs a way of reconnecting afterwards
	if t.server.isConnected() {
		if err := t.server.Disconnect(); err != nil {
			t.OutputError(err)
		} else {
			t.OutputInfo("Disconnected from beanstalk server. Press F9 to connect.")
			t.updater.Stop()
			t.mainList.Clear()
			t.peekView.Clear()
			t.tubes = make([]string, 0)
			t.SetupKeysView()
		}
	} else {
		form := tview.NewForm()
		form.AddInputField("Host (defaults to localhost)", "", 20, nil, nil).
			AddInputField("Port (defaults to 11300)", "", 6, nil, nil).
			AddButton("Connect", func() {
				t.pages.RemovePage("connect")
				t.OutputInfo("Connecting...")

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
							t.OutputError(fmt.Errorf("invalid port: %w", err))
						})
						return
					}

					if err := t.server.Connect(host, port); err != nil {
						t.app.QueueUpdateDraw(func() {
							t.OutputError(err)
						})
						return
					}

					t.app.QueueUpdateDraw(func() {
						address := fmt.Sprintf("%s:%d", host, port)
						t.OutputInfo("Connected to " + address)
						t.updater.Reset(1 * time.Second)
						t.SetupKeysView()
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

func (t *tui) HandleExit() {
	t.app.Stop()
}

func (t *tui) OutputError(err error) {
	t.outputView.SetText(fmt.Sprintf("[red:]%s[-:-:-]", err.Error()))
}

func (t *tui) OutputInfo(info string) {
	t.outputView.SetText(info)
}
