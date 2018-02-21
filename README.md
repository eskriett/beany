# beany

Beany is an interactive command-line client for
[beanstalkd](https://github.com/kr/beanstalkd) written in
[Go](https://golang.org/)

## Features

* Persistent history
* Mass deletion of jobs on selected tube
* Paged output when viewing jobs
* Tube autocompletion for commands

## Note

Still in early development, features are likely to change. Roadmap:

* [ ] Ensure all beanstalk commands are supported
* [ ] Improve documentation
* [ ] Tests
* [ ] Package

## Installation

```
go get github.com/eskriett/beany
```

Requires Go 1.10+

## Usage

```
beany [options]
```

Running `beany` without any arguments will start it in interactive mode. By
default it will attempt to connect to `127.0.0.1:11300`:

```
$ beany
Connected to '127.0.0.1:11300'
[default] >>> version
beany version: 0.0.1
```

All commands can also be provided as arguments, e.g.

```
$ beany version
beany version: 0.0.1
```

A list of available commands can be viewed with:

```
$ beany help

Commands:
  clear               clear the screen
  connect             connects to a beanstalk server
  delete              delete a job
  delete-buried       deletes all buried jobs on the current tube
  delete-delayed      deletes all delayed jobs on the current tube
  delete-ready        deletes all ready jobs on the current tube
  disconnect          disconnects from the beanstalk server
  exit                exit the program
  help                display help
  info                info about the current connection
  kick                kick jobs from the current tube
  list-tubes          lists tubes
  peek-buried         peek at buried jobs
  peek-delayed        peek at delayed jobs
  peek-ready          peek at ready jobs
  put                 puts data on the current tube
  stats               display server statistics
  stats-tube          stats the current tube
  use                 use a tube
  version             display version information
```

Additional information for a particular command can be view with:

```
$ beany peek-ready help

Looks at the job at the front of the ready queue.

A tube argument can also be provided, otherwise uses the current active tube:

  peek-ready <TUBE>

This command is available via the 'pr' alias
```

### History

`beany` maintains a persistent history, this can be found at `~/.beany_history`.

### Colours

Coloured output can be disabled with `beany --boring`

### Pager

By default `beany` will use whatever the `$PAGER` environment variable is
configured to, otherwise it will default to `less -R`. For example to run
`beany` with `more` run:

```
$ PAGER=more beany
```

### Editor

When use the `put` command `beany` will first look for the editor defined by the
`$EDITOR` envrionment variable. If this cannot be found, `beany` will fallback
to using `vi`. For example, to run `beany` with `nano`:

```
$ EDITOR=nano beany
```

## License

MIT

## Credits

Library | Use
------- | -----
[github.com/abiosoft/ishell](https://github.com/abiosoft/ishell) | interactive shell library
[github.com/kr/beanstalk](https://github.com/kr/beanstalk) | beanstalk client
[github.com/fatih/color](https://github.com/fatih/color) | colour output
[github.com/olekukonko/tablewriter](https://github.com/olekukonko/tablewriter) | ascii table

Also thanks to [beanwalker](https://github.com/kadekcipta/beanwalker) for the
initial inspiration for this tool
