package main

import (
	"flag"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/fatih/color"
)

const Version = "0.0.1"

func main() {
	flagNoColor := flag.Bool("boring", false, "Disable color output")
	flagConnect := flag.String("connect", "127.0.0.1:11300", "Server to connect to")
	flag.Parse()

	if *flagNoColor || len(os.Args) > 1 {
		color.NoColor = true
	}

	host, portStr, err := net.SplitHostPort(*flagConnect)
	if err != nil {
		host = *flagConnect
	}

	opts := []serverOption{
		WithHost(host),
	}

	if portStr != "" {
		port, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			log.Fatalf("unable to parse port '%s': %s", portStr, err.Error())
		}

		opts = append(opts, WithPort(int(port)))
	}

	cli := NewCli(opts...)
	nonCLIArgs := flag.Args()

	if len(nonCLIArgs) != 0 {
		if err := cli.shell.Process(nonCLIArgs...); err != nil {
			log.Fatal(err)
		}
	} else {
		cli.Run()
	}
}
