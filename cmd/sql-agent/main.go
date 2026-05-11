package main

import (
	"os"

	"github.com/davidfantasy/sql-agent-cli/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
