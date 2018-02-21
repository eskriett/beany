package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/fatih/color"
	"github.com/nsf/termbox-go"
	"github.com/olekukonko/tablewriter"
)

type cli struct {
	server *server
	shell  *ishell.Shell
}

func NewCli() *cli {
	shell := ishell.New()
	shell.SetPager("less", []string{"-R"})
	shell.SetHomeHistoryPath(".beany_history")

	server := server{}
	server.connect()

	cli := cli{&server, shell}

	cli.addConnectCmd()
	cli.addDeleteCmd()
	cli.addDisconnectCmd()
	cli.addInfoCmd()
	cli.addKickCmd()
	cli.addListTubesCmd()
	cli.addStatsCmd()
	cli.addStatsTubeCmd()
	cli.addUseTubeCmd()
	cli.addVersionCmd()

	for _, state := range []string{"buried", "delayed", "ready"} {
		cli.addPeekCmd(state)
		cli.addDeleteAllCmd(state)
	}

	if len(os.Args) == 1 {
		shell.Process("info")
	}
	cli.setPrompt()

	return &cli
}

func (c *cli) addConnectCmd() {
	c.shell.AddCmd(&ishell.Cmd{
		Name:     "connect",
		Help:     "connects to a beanstalk server",
		LongHelp: helpConnect,
		Func: func(i *ishell.Context) {
			var host string
			var port int

			if len(i.Args) == 0 {
				host = "127.0.0.1"
				port = 11300
			} else if len(i.Args) == 1 {
				host = i.Args[0]
			} else if len(i.Args) == 2 {
				host = i.Args[0]
				var err error
				if port, err = strconv.Atoi(i.Args[1]); err != nil {
					outputError(err, i)
					return
				}
			} else {
				outputError(errors.New("Too many arguments"), i)
				return
			}

			if err := c.server.Connect(host, port); err != nil {
				outputError(err, i)
			}

			outputConnectionInfo(c, i)
			c.setPrompt()
		},
	})
}

func (c *cli) addDeleteCmd() {
	c.shell.AddCmd(&ishell.Cmd{
		Name:     "delete",
		Aliases:  []string{"del", "dj"},
		Help:     "delete a job",
		LongHelp: helpDelete,
		Func: func(i *ishell.Context) {
			var toDeleteStr string
			if len(i.Args) == 1 {
				toDeleteStr = i.Args[0]
			} else {
				outputError(errors.New("Wrong number of arguments provided"), i)
				return
			}

			toDelete, err := strconv.ParseUint(toDeleteStr, 10, 64)
			if err != nil {
				outputError(err, i)
				return
			}

			msg := fmt.Sprintf("Are you sure you want to delete job #%v", toDelete)
			if !getConfirmation(msg, i) {
				return
			}

			if err := c.server.Delete(toDelete); err != nil {
				outputError(err, i)
			} else {
				outputInfo(fmt.Sprintf("Deleted job #%v", toDelete), i)
			}
		},
	})
}

func (c *cli) addDeleteAllCmd(state string) {
	c.shell.AddCmd(&ishell.Cmd{
		Name:     fmt.Sprintf("delete-%s", state),
		Aliases:  []string{fmt.Sprintf("d%c", state[0])},
		Help:     fmt.Sprintf("deletes all %s jobs on the current tube", state),
		LongHelp: fmt.Sprintf(helpDeleteAll, state, state, state[0]),
		Completer: func([]string) []string {
			return c.server.ListTubes()
		},
		Func: func(i *ishell.Context) {
			tube, err := getTubeFromArgs(c, i)
			if err != nil {
				outputError(err, i)
				return
			}

			msg := fmt.Sprintf("Are you sure you want to delete all %s jobs from the %s tube",
				state, tube)
			if !getConfirmation(msg, i) {
				return
			}

			if n, _ := c.server.DeleteAll(state, tube); n > 0 {
				outputInfo(fmt.Sprintf("Deleted %d %s jobs", n, state), i)
			} else if n == 0 {
				outputError(errors.New(fmt.Sprintf("No %s jobs deleted", state)), i)
			}
		},
	})
}

func (c *cli) addDisconnectCmd() {
	c.shell.AddCmd(&ishell.Cmd{
		Name:     "disconnect",
		Help:     "disconnects from the beanstalk server",
		LongHelp: helpDisconnect,
		Func: func(i *ishell.Context) {
			if err := c.server.Disconnect(); err != nil {
				outputError(err, i)
			}
			c.setPrompt()
		},
	})
}

func (c *cli) addInfoCmd() {
	c.shell.AddCmd(&ishell.Cmd{
		Name:     "info",
		Help:     "info about the current connection",
		LongHelp: helpInfo,
		Func: func(i *ishell.Context) {
			outputConnectionInfo(c, i)
		},
	})
}

func (c *cli) addKickCmd() {
	c.shell.AddCmd(&ishell.Cmd{
		Name:     "kick",
		Help:     "kick jobs from the current tube",
		LongHelp: helpKick,
		Func: func(i *ishell.Context) {
			tube := c.server.CurrentTubeName()

			var toKickStr string
			if len(i.Args) == 0 {
				if stats, err := c.server.StatsTube(tube); err != nil {
					outputError(err, i)
					return
				} else {
					toKickStr = stats["current-jobs-buried"]
				}
			} else if len(i.Args) == 1 {
				toKickStr = i.Args[0]
			} else {
				outputError(errors.New("Too many arguments provided"), i)
				return
			}

			toKick, err := strconv.Atoi(toKickStr)
			if err != nil {
				outputError(err, i)
				return
			}

			if kicked, err := c.server.Kick(tube, toKick); err != nil {
				outputError(err, i)
			} else {
				outputInfo(fmt.Sprintf("Kicked %v jobs", kicked), i)
			}
		},
	})
}

func (c *cli) addListTubesCmd() {
	c.shell.AddCmd(&ishell.Cmd{
		Name:     "list-tubes",
		Aliases:  []string{"lt", "list"},
		Help:     "lists tubes",
		LongHelp: helpListTubes,
		Func: func(i *ishell.Context) {
			tubes := c.server.GetTubeStats()

			var output bytes.Buffer
			table := tablewriter.NewWriter(&output)
			table.SetHeader([]string{"Tube", "Ready", "Delayed", "Buried"})
			table.SetBorder(false)
			cyan := color.New(color.FgCyan, color.Bold).SprintFunc()

			for _, tube := range sortedMapKeys(tubes) {
				stats := tubes[tube]
				table.Append([]string{
					cyan(tube),
					color.GreenString(stats["current-jobs-ready"]),
					color.YellowString(stats["current-jobs-delayed"]),
					color.RedString(stats["current-jobs-buried"]),
				})
			}

			table.Render()
			outputPaged(output.String(), i)
		},
	})
}

func (c *cli) addPeekCmd(state string) {
	c.shell.AddCmd(&ishell.Cmd{
		Name:     fmt.Sprintf("peek-%s", state),
		Aliases:  []string{fmt.Sprintf("p%c", state[0])},
		Help:     fmt.Sprintf("peek at %s jobs", state),
		LongHelp: fmt.Sprintf(helpPeek, state, state, state[0]),
		Completer: func([]string) []string {
			return c.server.ListTubes()
		},
		Func: func(i *ishell.Context) {
			tube, err := getTubeFromArgs(c, i)
			if err != nil {
				outputError(err, i)
				return
			}

			if id, body, err := c.server.Peek(state, tube); err != nil {
				outputError(err, i)
			} else {
				cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
				details := fmt.Sprintf("%s\n%s",
					cyan(fmt.Sprintf("Job #%v", id)),
					string(body))
				outputPaged(details, i)
			}
		},
	})
}

func (c *cli) addStatsCmd() {
	c.shell.AddCmd(&ishell.Cmd{
		Name:     "stats",
		Help:     "display server statistics",
		LongHelp: helpStats,
		Func: func(i *ishell.Context) {
			stats, err := c.server.Stats()
			if err != nil {
				outputError(err, i)
				return
			}

			cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
			var sb strings.Builder
			for _, key := range sortedMapKeys(stats) {
				sb.WriteString(fmt.Sprintf("%s: %s\n", cyan(key), stats[key]))
			}
			outputPaged(sb.String(), i)
		},
	})
}

func (c *cli) addStatsTubeCmd() {
	c.shell.AddCmd(&ishell.Cmd{
		Name:     "stats-tube",
		Aliases:  []string{"st"},
		Help:     "stats the current tube",
		LongHelp: helpStatsTube,
		Completer: func([]string) []string {
			return c.server.ListTubes()
		},
		Func: func(i *ishell.Context) {
			tube, err := getTubeFromArgs(c, i)
			if err != nil {
				outputError(err, i)
				return
			}

			if stats, err := c.server.StatsTube(tube); err != nil {
				outputError(err, i)
			} else {
				cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
				var sb strings.Builder
				for _, key := range sortedMapKeys(stats) {
					sb.WriteString(fmt.Sprintf("%s: %s\n", cyan(key), stats[key]))
				}
				outputPaged(sb.String(), i)
			}
		},
	})
}

func (c *cli) addUseTubeCmd() {
	c.shell.AddCmd(&ishell.Cmd{
		Name:     "use",
		Aliases:  []string{"ut"},
		Help:     "use a tube",
		LongHelp: helpUse,
		Completer: func([]string) []string {
			return c.server.ListTubes()
		},
		Func: func(i *ishell.Context) {
			if len(i.Args) == 0 {
				outputError(errors.New("Tube required"), i)
				return
			}

			tube, err := getTubeFromArgs(c, i)
			if err != nil {
				outputError(err, i)
				return
			}
			c.server.UseTube(tube)
			c.setPrompt()
		},
	})
}

func (c *cli) addVersionCmd() {
	c.shell.AddCmd(&ishell.Cmd{
		Name:     "version",
		Help:     "display version information",
		LongHelp: helpVersion,
		Func: func(i *ishell.Context) {
			outputInfo("beany version: "+Version, i)
		},
	})
}

func getConfirmation(msg string, i *ishell.Context) bool {
	i.ShowPrompt(false)
	defer i.ShowPrompt(true)

	i.Println(msg + " [yn]?")
	choice := strings.ToLower(i.ReadLine())

	if choice == "y" || choice == "yes" {
		return true
	} else if choice == "n" || choice == "no" {
		return false
	}

	outputError(errors.New("Not a valid choice, defaulting to 'n'"), i)
	return false
}

func getTubeFromArgs(c *cli, i *ishell.Context) (string, error) {

	if len(i.Args) == 0 {
		return c.server.CurrentTubeName(), nil
	} else if len(i.Args) == 1 {
		return i.Args[0], nil
	}

	return "", errors.New("Too many arguments provided")
}

func outputConnectionInfo(c *cli, i *ishell.Context) {
	if s, err := c.server.ConnectionStr(); err != nil {
		outputError(err, i)
	} else {
		outputInfo(fmt.Sprintf("Connected to '%s'", s), i)
	}
}

func outputError(e error, i *ishell.Context) {
	boldRed := color.New(color.FgRed, color.Bold).SprintFunc()
	i.Printf("%s\n", boldRed(e))
}

func outputInfo(s string, i *ishell.Context) {
	boldCyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	i.Printf("%s\n", boldCyan(s))
}

func outputPaged(s string, i *ishell.Context) {
	if err := termbox.Init(); err != nil {
		panic(err)
	}
	_, h := termbox.Size()
	termbox.Close()

	if numLines := strings.Count(s, "\n"); numLines < h {
		i.Print(s)
	} else {
		i.ShowPaged(s)
	}
}

func (c *cli) Run() {
	c.shell.Run()
}

func (c *cli) setPrompt() {
	yellow := color.New(color.FgYellow).SprintFunc()
	boldMagenta := color.New(color.FgMagenta, color.Bold).SprintFunc()
	boldRed := color.New(color.FgRed, color.Bold).SprintFunc()

	var prompt string
	if c.server.isConnected() {
		tube := c.server.CurrentTubeName()

		prompt = fmt.Sprintf("%s%s%s",
			yellow("["), boldMagenta(tube), yellow("] >>> "))
	} else {
		prompt = fmt.Sprintf("%s%s%s",
			yellow("["), boldRed("none"), yellow("] >>> "))
	}

	c.shell.SetPrompt(prompt)
}

func sortedMapKeys(m interface{}) (sorted []string) {
	keys := reflect.ValueOf(m).MapKeys()
	for _, k := range keys {
		sorted = append(sorted, k.Interface().(string))
	}
	sort.Strings(sorted)
	return
}
