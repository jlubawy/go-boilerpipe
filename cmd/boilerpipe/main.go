package main

import (
	//"errors"
	"fmt"
	"runtime"
	"text/template"
	//"net/http"
	//"net/http/cookiejar"
	"os"
	//
	"github.com/jlubawy/go-boilerpipe"
	//"github.com/jlubawy/go-boilerpipe/extractor"
	//url "github.com/jlubawy/go-boilerpipe/normurl"
	//
	//"golang.org/x/net/publicsuffix"
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
}

var templUsage = template.Must(template.New("").Parse(`Boilerpipe removes boilerplate and extracts text content from HTML documents.

Usage:

       boilerpipe command [arguments]

The commands are:
{{range $name, $command := .}}
       {{printf "%-7s    %s" $name $command.Description}}{{end}}

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

	} else if cmdStr == "version" {
		fmt.Fprintf(os.Stderr, "boilerpipe %s %s/%s\n", boilerpipe.Version, runtime.GOOS, runtime.GOARCH)

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

	//	if *debug {
	//		extractor.EnableLogging(".", true)
	//	}
	//
	//	if *file != "" {
	//		// If a file path was provided then read from the file
	//		f, err := os.Open(*file)
	//		if err != nil {
	//			fmt.Fprintln(os.Stderr, err)
	//			os.Exit(1)
	//		}
	//		defer f.Close()
	//
	//		text, err := boilerpipe.ExtractText(f)
	//		if err != nil {
	//			fmt.Fprintln(os.Stderr, err)
	//			os.Exit(1)
	//		}
	//
	//		fmt.Print(text)
	//
	//	} else if *port == "" {
	//		// Else if no port is provided take a URL from the command line and output the
	//		// results to stdout.
	//
	//		url := flag.Arg(0)
	//		if url == "" {
	//			fmt.Fprintln(os.Stderr, "Must specify url.\n")
	//			flag.Usage()
	//		}
	//
	//		doc, err := process(url)
	//		if err != nil {
	//			fmt.Fprintln(os.Stderr, err)
	//			os.Exit(1)
	//		}
	//
	//		fmt.Print(doc.Content())
	//
	//	} else {
	//		// Else if a port is provided start the HTTP server
	//
	//		http.HandleFunc("/", Handle(Index))
	//		http.HandleFunc("/extract", Handle(Extract))
	//
	//		fmt.Fprintln(os.Stderr, "Starting server on port", *port)
	//		if err := http.ListenAndServe(*port, nil); err != nil {
	//			fmt.Fprintln(os.Stderr, err.Error())
	//			os.Exit(1)
	//		}
	//	}
}

func fatalf(fmtStr string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmtStr, args...)
	os.Exit(1)
}
