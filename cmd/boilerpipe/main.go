package main

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"runtime"
	"text/template"

	"github.com/jlubawy/go-boilerpipe"

	"golang.org/x/net/publicsuffix"
)

type Command struct {
	Description string
	CommandFunc func(args []string)
	HelpFunc    func()
}

var commands = map[string]*Command{
	"crawl":   commandCrawl,
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
			fmt.Fprint(os.Stderr, `boilerpipe: unknown subcommand "%s"
Run 'boilerpipe help' for usage.
`, cmdStr)
		}
	}
}

func fatalf(fmtStr string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmtStr, args...)
	os.Exit(1)
}

var commandVersion = &Command{
	Description: "print boilerpipe version",
	CommandFunc: func(args []string) {
		fmt.Fprintf(os.Stderr, "boilerpipe %s %s/%s\n", boilerpipe.Version, runtime.GOOS, runtime.GOARCH)
	},
	HelpFunc: func() {
		fmt.Fprintf(os.Stderr, `usage: boilerpipe version

Version prints the boilerpipe version, as reported by boilerpipe.Version.
`)
	},
}

func NewClient() *http.Client {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		fatalf("error: %s\n", err)
	}

	return &http.Client{
		Jar: jar,
	}
}
