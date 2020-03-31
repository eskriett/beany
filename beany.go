package main

import (
	"flag"
	"log"
	"os"

	"github.com/fatih/color"
)

const Version = "0.0.1"

func main() {
	flagNoColor := flag.Bool("boring", false, "Disable color output")
	flag.Parse()

	if *flagNoColor || len(os.Args) > 1 {
		color.NoColor = true
	}

	cli := NewCli()

	if len(os.Args) > 1 {
		if err := cli.shell.Process(os.Args[1:]...); err != nil {
			log.Fatal(err)
		}
	} else {
		cli.Run()
	}
}
