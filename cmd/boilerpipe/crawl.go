package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/jlubawy/go-boilerpipe/crawler"
	"github.com/jlubawy/go-boilerpipe/normurl"

	"golang.org/x/net/context"
)

var commandCrawl = &Command{
	Description: "crawl a website for HTML documents",
	CommandFunc: crawlFunc,
	HelpFunc:    crawlHelpFunc,
}

func crawlFunc(args []string) {
	var startPage, lastPage uint

	flagset := flag.NewFlagSet("", flag.ExitOnError)
	flagset.UintVar(&startPage, "startPage", 1, "page to start the crawl on")
	flagset.UintVar(&lastPage, "lastPage", 0, "page to end the crawl on (inclusive)")
	flagset.Usage = crawlHelpFunc
	flagset.Parse(args)

	if len(flagset.Args()) == 0 {
		crawlHelpFunc()
	} else if len(flagset.Args()) > 1 {
		fatalf("usage: boilerpipe crawl command\n\nToo many arguments given.\n")
	}

	rootURL := flagset.Args()[0]
	u, err := url.Parse(rootURL)
	if err != nil {
		fatalf("%s\n", err)
	}
	normRoot := normurl.NewURL(u, nil)

	urlMap := make(map[string]uint)

	client := NewClient()
	c := crawler.NewCrawler(client, crawler.DefaultThrottleDuration)
	c.LastPage = lastPage
	results, _ := c.Crawl(context.Background(), rootURL, startPage)
	for res := range results {
		if err := res.Error; err != nil {
			log.Fatal(err)
		}

		for _, u := range res.URLs {
			nu, err := normRoot.Parse(u)
			if err != nil {
				log.Fatal(err)
			}

			if normurl.IsChild(normRoot, nu) {
				if _, dateExists := nu.Date(); dateExists {
					if _, exists := urlMap[nu.String()]; exists {
						urlMap[nu.String()] += 1
					} else {
						urlMap[nu.String()] = 1
						log.Printf("Page %-2d: %s\n", res.Page, nu.String())
					}
				}
			}
		}
	}

	if err := json.NewEncoder(os.Stdout).Encode(&urlMap); err != nil {
		fatalf("error: %s\n", err)
	}
}

func crawlHelpFunc() {
	fmt.Fprint(os.Stderr, `usage: boilerpipe crawl [url]

Crawl crawls the URL provided and prints the results to stdout.
`)
	os.Exit(1)
}
