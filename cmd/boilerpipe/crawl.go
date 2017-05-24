package main

import (
	"fmt"
	"os"
	"text/template"
)

var commandCrawl = &Command{
	Description: "crawl a website for HTML documents",
	CommandFunc: crawlFunc,
	HelpFunc:    crawlHelpFunc,
}

func crawlFunc(args []string) {

}

var templHelpCrawl = template.Must(template.New("help_crawl").Parse(``))

func crawlHelpFunc() {
	fmt.Fprint(os.Stderr, `usage: boilerpipe crawl [url]

Crawl crawls the URL provided and prints the results to stdout.
`)
}
