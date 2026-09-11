package main

import (
	"github.com/higordiego/ai-swtich/internal/cli"
	"os"
)

var exit = os.Exit

func main() { exit(cli.Main()) }
