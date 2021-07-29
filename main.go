package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"os"
	"runtime"
)

var (
	commit      string
	version     = "unversioned"
	date        string
	buildSource = "DEV"
)

func main() {
	err := (&cli.App{
		Name:                 "Regolith",
		Usage:                "A bedrock addon compiler pipeline",
		EnableBashCompletion: true,
		Version: fmt.Sprintf(
			"%s\n   Date: %s\n   BuildSource: %s\n   Commit: %s\n   OS: %s\n   Arch: %s",
			version,
			date,
			buildSource,
			commit,
			runtime.GOOS,
			runtime.GOARCH,
		),
		Commands: []*cli.Command{
			{
				Name:  "build",
				Usage: "Placeholder",
				Action: func(c *cli.Context) error {
					fmt.Println("Placeholder")
					return nil
				},
			},
			{
				Name:  "install",
				Usage: "Placeholder",
				Action: func(c *cli.Context) error {
					fmt.Println("Placeholder")
					return nil
				},
			},
		},
	}).Run(os.Args)
	if err != nil {
		panic(err)
	}
}
