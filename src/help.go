package src

import (
	"io"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

func CustomHelp() {
	cli.AppHelpTemplate = `NAME:
   {{.Name | yellow}}{{if .Usage}} - {{.Usage}}{{end}}

USAGE:
   {{if .UsageText}}{{.UsageText | nindent 3 | trim}}{{else}}{{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Version}}{{if not .HideVersion}}

VERSION:
                 {{.Version}}
   Commit:       {{.Metadata.Commit}}
   Build Source: {{.Metadata.BuildSource}}
   Date:         {{.Metadata.Date}}{{end}}{{end}}{{if .Description}}

DESCRIPTION:
   {{.Description | nindent 3 | trim}}{{end}}{{if len .Authors}}

AUTHOR{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author}}{{end}}{{end}}{{if .VisibleCommands}}

COMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{else}}{{range .VisibleCommands}}
   {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}

GLOBAL OPTIONS:
   {{range $index, $option := .VisibleFlags}}{{if $index}}
   {{end}}{{$option}}{{end}}{{end}}{{if .Copyright}}

COPYRIGHT:
   {{.Copyright}}{{end}}
`
	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		cli.HelpPrinterCustom(w, templ, data, map[string]interface{}{
			"red":     func(v string) string { return color.RedString("%s", v) },
			"green":   func(v string) string { return color.GreenString("%s", v) },
			"yellow":  func(v string) string { return color.YellowString("%s", v) },
			"blue":    func(v string) string { return color.BlueString("%s", v) },
			"magenta": func(v string) string { return color.MagentaString("%s", v) },
			"cyan":    func(v string) string { return color.CyanString("%s", v) },
		})
	}
}
