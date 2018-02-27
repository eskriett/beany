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
)
