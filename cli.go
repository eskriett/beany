package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/fatih/color"
	"github.com/nsf/termbox-go"
	"github.com/olekukonko/tablewriter"
)

const (
	historyFile = ".beany_history"
)

type cli struct {
	server *server
	shell  *ishell.Shell
}

func NewCli(serverOpts ...serverOption) *cli {
	shell := ishell.New()

	if pager := os.Getenv("PAGER"); pager != "" {
		pagerArgs := strings.Split(pager, " ")
		shell.SetPager(pagerArgs[0], pagerArgs[1:])
	} else {
		shell.SetPager("less", []string{"-R"})
	}

	shell.SetHomeHistoryPath(historyFile)

	server := &server{}

	for _, serverOpt := range serverOpts {
		serverOpt(server)
	}

	server.connect()

	cli := cli{server, shell}

	cli.addConnectCmd()
	cli.addDeleteCmd()
	cli.addDisconnectCmd()
	cli.addInfoCmd()
	cli.addKickCmd()
	cli.addListTubesCmd()
	cli.addPeekJobCmd()
	cli.addPutCmd()
	cli.addStatsCmd()
	cli.addStatsJobCmd()
	cli.addStatsTubeCmd()
	cli.addUseTubeCmd()
	cli.addVersionCmd()

	for _, state := range []string{"buried", "delayed", "ready"} {
		cli.addPeekCmd(state)
		cli.addDeleteAllCmd(state)
	}

	if len(os.Args) == 1 {
		if err := shell.Process("info"); err != nil {
			log.Fatal(err)
		}
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
				outputError(errors.New("too many arguments"), i)
				return
			}

			if err := c.server.Connect(host, port); err != nil {
				outputError(err, i)
				return
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
				outputError(errors.New("wrong number of arguments provided"), i)
				return
			}

			toDelete, err := strconv.ParseUint(toDeleteStr, 10, 64)
			if err != nil {
				outputError(err, i)
				return
			}

			msg := fmt.Sprintf("Are you sure you want to delete job #%v", toDelete)
			if !c.getConfirmation(msg, i) {
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
		Name:      fmt.Sprintf("delete-%s", state),
		Aliases:   []string{fmt.Sprintf("d%c", state[0])},
		Help:      fmt.Sprintf("deletes all %s jobs on the current tube", state),
		LongHelp:  fmt.Sprintf(helpDeleteAll, state, state, state[0]),
		Completer: c.listTubes,
		Func: func(i *ishell.Context) {
			tube, err := getTubeFromArgs(c, i)
			if err != nil {
				outputError(err, i)
				return
			}

			msg := fmt.Sprintf("Are you sure you want to delete all %s jobs from the %s tube",
				state, tube)
			if !c.getConfirmation(msg, i) {
				return
			}

			if n, _ := c.server.DeleteAll(state, tube); n > 0 {
				outputInfo(fmt.Sprintf("Deleted %d %s jobs", n, state), i)
			} else if n == 0 {
				outputError(fmt.Errorf("No %s jobs deleted", state), i)
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
			tube, err := c.server.CurrentTubeName()
			if err != nil {
				outputError(err, i)
				return
			}

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
				outputError(errors.New("too many arguments provided"), i)
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
			tubes, err := c.server.GetTubeStats()
			if err != nil {
				outputError(err, i)
				return
			}

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

func (c *cli) addPeekJobCmd() {
	c.shell.AddCmd(&ishell.Cmd{
		Name:      "peek",
		Aliases:   []string{"p"},
		Help:      "peek at the given job",
		LongHelp:  helpPeekJob,
		Completer: func(args []string) []string { return []string{} },
		Func: func(i *ishell.Context) {
			job, err := getJobFromArgs(c, i)
			if err != nil {
				outputError(err, i)
				return
			}

			if jobDetails, err := c.server.PeekJob(job); err != nil {
				outputError(err, i)
			} else {
				cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
				details := fmt.Sprintf("%s\n%s",
					cyan(fmt.Sprintf("Job #%d\n", job)),
					jobDetails,
				)
				outputPaged(details, i)
			}
		},
	})
}

func (c *cli) addPeekCmd(state string) {
	c.shell.AddCmd(&ishell.Cmd{
		Name:      fmt.Sprintf("peek-%s", state),
		Aliases:   []string{fmt.Sprintf("p%c", state[0])},
		Help:      fmt.Sprintf("peek at %s jobs", state),
		LongHelp:  fmt.Sprintf(helpPeek, state, state, state[0]),
		Completer: c.listTubes,
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

func (c *cli) addPutCmd() {
	c.shell.AddCmd(&ishell.Cmd{
		Name:      "put",
		Help:      "puts data on the current tube",
		LongHelp:  helpPut,
		Completer: c.listTubes,
		Func: func(i *ishell.Context) {
			tube, err := getTubeFromArgs(c, i)
			if err != nil {
				outputError(err, i)
				return
			}

			temp, err := ioutil.TempFile(os.TempDir(), "beany")
			if err != nil {
				outputError(err, i)
				return
			}
			defer os.Remove(temp.Name())

			var cmd *exec.Cmd

			if editor := os.Getenv("EDITOR"); editor != "" {
				editorArgs := strings.Split(editor, " ")
				editorArgs = append(editorArgs, temp.Name())
				cmd = exec.Command(editorArgs[0], editorArgs[1:]...)
			} else {
				cmd = exec.Command("vi", temp.Name())
			}
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Start(); err != nil {
				outputError(err, i)
				return
			}
			if err := cmd.Wait(); err != nil {
				outputError(err, i)
				return
			}

			job, err := ioutil.ReadFile(temp.Name())
			if err != nil {
				outputError(err, i)
				return
			}

			if len(job) == 0 {
				outputError(errors.New("no data in job, not adding to tube"), i)
				return
			}

			if id, err := c.server.Put(job, tube); err != nil {
				outputError(err, i)
			} else {
				outputInfo(fmt.Sprintf("Put job (#%d) onto %s", id, tube), i)
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

func (c *cli) addStatsJobCmd() {
	c.shell.AddCmd(&ishell.Cmd{
		Name:     "stats-job",
		Aliases:  []string{"sj"},
		Help:     "prints the stats for a job",
		LongHelp: helpStatsJob,
		Func: func(i *ishell.Context) {
			var toStatStr string
			if len(i.Args) == 1 {
				toStatStr = i.Args[0]
			} else {
				outputError(errors.New("wrong number of arguments provided"), i)
				return
			}

			toStat, err := strconv.ParseUint(toStatStr, 10, 64)
			if err != nil {
				outputError(err, i)
				return
			}

			if stats, err := c.server.StatsJob(toStat); err != nil {
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

func (c *cli) addStatsTubeCmd() {
	c.shell.AddCmd(&ishell.Cmd{
		Name:      "stats-tube",
		Aliases:   []string{"st"},
		Help:      "stats the current tube",
		LongHelp:  helpStatsTube,
		Completer: c.listTubes,
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
		Name:      "use",
		Aliases:   []string{"ut"},
		Help:      "use a tube",
		LongHelp:  helpUse,
		Completer: c.listTubes,
		Func: func(i *ishell.Context) {
			if len(i.Args) == 0 {
				outputError(errors.New("tube required"), i)
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

func (c *cli) getConfirmation(msg string, i *ishell.Context) bool {
	i.ShowPrompt(false)
	defer c.shell.SetHomeHistoryPath(historyFile)
	defer i.ShowPrompt(true)

	c.shell.SetHistoryPath("")

	i.Print(msg + " [yn]? ")
	choice := strings.ToLower(i.ReadLine())

	if choice == "y" || choice == "yes" {
		return true
	} else if choice == "n" || choice == "no" {
		return false
	}

	outputError(errors.New("not a valid choice, defaulting to 'n'"), i)
	return false
}

func getJobFromArgs(c *cli, i *ishell.Context) (uint64, error) {
	if len(i.Args) == 0 {
		return 0, errors.New("too few arguments provided")
	} else if len(i.Args) == 1 {
		job, err := strconv.ParseUint(i.Args[0], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("unable to parse job: %w", err)
		}

		return job, nil
	}

	return 0, errors.New("too many arguments provided")
}

func getTubeFromArgs(c *cli, i *ishell.Context) (string, error) {
	if len(i.Args) == 0 {
		return c.server.CurrentTubeName()
	} else if len(i.Args) == 1 {
		return i.Args[0], nil
	}

	return "", errors.New("too many arguments provided")
}

func (c *cli) listTubes([]string) []string {
	tubes, err := c.server.ListTubes()
	if err != nil {
		return nil
	}

	return tubes
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
		if err := i.ShowPaged(s); err != nil {
			log.Fatal(err)
		}
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
		tube, _ := c.server.CurrentTubeName()

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
