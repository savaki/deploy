package command

import (
	"fmt"

	"github.com/urfave/cli"
)

var Version = cli.Command{
	Name:   "version",
	Action: versionCommand,
}

func versionCommand(_ *cli.Context) error {
	fmt.Println("latest")
	return nil
}
