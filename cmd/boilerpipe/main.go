package main

import (
	"fmt"
	"os"
	"text/template"

	"github.com/jlubawy/go-boilerpipe"
)

type Command struct {
	Description string
	CommandFunc func(args []string)
	HelpFunc    func()
}

var commands = map[string]*Command{
	"extract": commandExtract,
	"serve":   commandServe,
	"version": commandVersion,
}

var templUsage = template.Must(template.New("").Parse(`Boilerpipe removes boilerplate and extracts text content from HTML documents.

Usage:

       boilerpipe command [arguments]

The commands are:
{{range $name, $command := .}}
       {{printf "%-7s    %s" $name $command.Description}}{{end}}

Use "boilerpipe help [command]" for more information about a command.
`))

func usage() {
	if err := templUsage.Execute(os.Stderr, &commands); err != nil {
		panic(err)
	}
	os.Exit(1)
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		usage()
	}

	cmdStr := args[0]
	if cmdStr == "help" || cmdStr == "-help" || cmdStr == "--help" || cmdStr == "-h" {
		if cmdStr == "help" {
			// Check if sub-command help
			if len(args) > 2 {
				fatalf("usage: boilerpipe help command\n\nToo many arguments given.\n")
			} else if len(args) == 2 {
				if command, exists := commands[args[1]]; exists {
					command.HelpFunc()
					os.Exit(1)
				}
			}
		}

		usage()

	} else {
		command, exists := commands[cmdStr]
		if exists {
			command.CommandFunc(args[1:])
		} else {
			fmt.Fprintf(os.Stderr, `boilerpipe: unknown subcommand "%s"
Run 'boilerpipe help' for usage.
`, cmdStr)
		}
	}
}

func infof(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func fatalf(format string, args ...interface{}) {
	infof(format, args...)
	os.Exit(1)
}

var commandVersion = &Command{
	Description: "print boilerpipe version",
	CommandFunc: func(args []string) {
		fmt.Fprintln(os.Stderr, boilerpipe.FullVersion)
	},
	HelpFunc: func() {
		fmt.Fprintf(os.Stderr, `usage: boilerpipe version

Version prints the boilerpipe version, as reported by boilerpipe.Version.
`)
	},
}
