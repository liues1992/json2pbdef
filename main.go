package main

import (
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/urfave/cli"
	"log"
	"os"
)

func main() {
	app := cli.NewApp()
	cli.AppHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}
USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .ArgsUsage}}{{.ArgsUsage}}{{else}} jsonData{{end}}
   {{if len .Authors}}
AUTHOR:
   {{range .Authors}}{{ . }}{{end}}
   {{end}}{{if .Commands}}
COMMANDS:
{{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
GLOBAL OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
   {{.Copyright}}
   {{end}}{{if .Version}}
VERSION:
   {{.Version}}
   {{end}}
`
	app.Name = "json2protodef"
	app.Usage = "Create protobuf definition from json data"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "flat",
			Usage: "generate flat message definition instead of nested messages",
		},
	}
	app.Action = func(c *cli.Context) error {
		fmt.Println("boom! I say!", c.NArg())
		if c.NArg() == 0 {
			return cli.ShowAppHelp(c)
		}
		data := []byte(c.Get(0))
		j, err := simplejson.NewJson()
		if err != nil {
			return err
		}
		m := j.MustMap()
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
