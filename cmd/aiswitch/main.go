package main

import (
	"github.com/higordiego/ai-switch/internal/cli"
	"os"
)

var exit = os.Exit

func main() { exit(cli.Main()) }
