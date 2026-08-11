package main

const (
	helpConnect = `Connects to a beanstalk server. With no arguments, will try to connect to the
127.0.0.1:11300. Can also provide host and port arguments.

To connect to a beanstalk server on <HOST> using port 11300:

  connect <HOST>

To connect to a beanstalk server on <HOST> using port <PORT>:

  connect <HOST> <PORT>

Will error if a connection cannot be established`

	helpDelete = `Deletes a job with the specified id:

  delete <ID>

This command is available via the 'del' and 'dj' aliases.`

	helpDeleteAll = `Deletes all %s jobs on the current tube.

Can also delete jobs not on the current tube by passing a tube argument:

  delete-%s <TUBE>

This command is available via the 'd%c' alias`

	helpDisconnect = `Disconnects from the currently connected beanstalk server`

	helpInfo = `Provides information, including hostname and port, about the current
connection`

	helpKick = `Kicks all jobs from the current tube. Alternatively the number of jobs can be
specified as an argument:

  kick <NUM_JOBS>`

	helpListTubes = `List the tubes for the connected beanstalk server. Outputs a table of results,
display tube, and details of the number of ready, delayed and buried jobs.

This command is available via the 'lt' and 'list' aliases`

	helpPeekJob = `Looks at the job with the given id.

  peek <JOB_ID>

This command is available via the 'p' alias`

	helpPeek = `Looks at the job at the front of the %s queue.

A tube argument can also be provided, otherwise uses the current active tube:

  peek-%s <TUBE>

This command is available via the 'p%c' alias`

	helpPut = `Opens an editor and allows data to be put onto the current tube. Alternatively
a tube can be provided:

  put <TUBE>

Will first attempt to open an editor defined with the $EDITOR environment
variable, otherwise defaults to vi.`

	helpStats = `Displays statistics for the connected beanstalk server`

	helpStatsJob = `Displays statistics for the specified job:

  stats-job <JOB>

This command is available via the 'sj' alias`

	helpStatsTube = `Displays stats for the current tube. Alternatively a tube argument can be
provided:

  stats-tube <TUBE>

This command is available via the 'st' alias`

	helpUse = `Change the current tube in use:

  use <TUBE>

This command is available via the 'ut' alias`

	helpVersion = `Displays beany version information`

	helpTUI = `In TUI mode, the screen is divided into three panes:

	- The left pane shows the tubes, with histograms of the number of ready, delayed and buried jobs in each tube. The list can be navigated with the arrow keys. Many commands act on the current tube - this means the one currently selected in this list.
	- The right pane shows the first job in each state on the currently selected tube. It also shows the most recent info or error message.
	- The footer shows the available commands.

The available commands are:

	- Help: Show this help message. This can also be opened using ?.
	- Put: Open a form allowing a new job to be put onto the selected tube, with configurable priority (lower is sooner), delay (in seconds, for which the job will remain in the delayed state before becoming ready), and body.
	- DeleteJob: Open a form allowing a single job to be deleted from any tube, by specifying its ID.
	- DeleteAllJobs: Open a modal allowing all ready, delayed or buried jobs to be deleted from the current tube.
	- KickJobs: Immediately kick all buried jobs on the current tube back to ready. If there are no buried jobs, kick all delayed jobs back to ready.
	- KickAllTubes: Immediately kick all buried jobs on all tubes back to ready. Delayed jobs are not affected, and must be kicked using KickJobs.
	- PauseTube: Open a modal where the current tube can be paused for one hour, or unpaused. Jobs in a paused tube can't be reserved, and delayed jobs' delay timers do not decrement. An already-paused tube can be paused again, which extends the remaining pause period to one hour.
	- PauseScreen: Pause the screen, so no more updates are shown. Pausing the screen allows data to be copied to the clipboard.
		- UnpauseScreen: When paused, the screen can be unpaused with the same hotkey.
	- Disconnect: Disconnect from the currently connected beanstalk server.
		- Connect: When disconnected, the same hotkey opens a modal allowing a beanstalk server to be connected to by specifying its hostname and port. By default, beany will attempt to connect to localhost:11300.
	- Exit: Exit the application Pressing q also exits.

Each modal and form can be closed by pressing the hotkey which opened it. All overlays can be closed by pressing Escape.

More commands are available when running beany in CLI mode.`
)
