// Command kitsu is the entrypoint of the CLI.
package main

import (
	"os"

	"github.com/samuelsulo/kitsu/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
