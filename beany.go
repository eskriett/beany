package main

import (
	"flag"
	"os"

	"github.com/fatih/color"
)

const Version = "0.0.1"

func main() {
	var flagNoColor = flag.Bool("boring", false, "Disable color output")
	flag.Parse()

	if *flagNoColor || len(os.Args) > 1 {
		color.NoColor = true
	}

	cli := NewCli()

	if len(os.Args) > 1 {
		cli.shell.Process(os.Args[1:]...)
	} else {
		cli.Run()
	}
}
