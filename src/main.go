package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	fmt.Println("Regolith | v0.1.0-dev")
	(&cli.App{
		Name:  "Regolith",
		Usage: "A bedrock addon compiler pipeline",
		Action: func(c *cli.Context) error {
			//this is where the actual app goes
			fmt.Printf("Hello %q", c.Args().Get(0))
			return nil
		},
	}).Run(os.Args)
}
