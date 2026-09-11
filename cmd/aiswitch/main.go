package main

import (
	"aiswitch/internal/cli"
	"os"
)

var exit = os.Exit

func main() { exit(cli.Main()) }
