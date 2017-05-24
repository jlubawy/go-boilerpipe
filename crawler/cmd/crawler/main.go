package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/jlubawy/go-boilerpipe/crawler"
	"github.com/jlubawy/go-boilerpipe/normurl"
)

func fatalln(v interface{}) {
	fmt.Fprintln(os.Stderr, v)
	os.Exit(1)
}

func main() {
	if len(os.Args) != 2 {
		fatalln("must provide URL to extract from")
	}

	root, err := url.Parse(os.Args[1])
	if err != nil {
		fatalln(err)
	}

	normRoot := normurl.NewURL(root, nil)
	c := crawler.NewCrawler(1, 11)

	pageFunc := func(u *url.URL, page uint) *url.URL {
		vals := u.Query()
		vals.Set("page", fmt.Sprintf("%d", page))
		u.RawQuery = vals.Encode()
		return u
	}

	hrefMap := make(map[string]uint)

	results := c.Crawl(http.DefaultClient, root, pageFunc)
	for res := range results {
		log.Println(res.URL())

		for _, err := range res.Errors() {
			log.Println(err)
		}

		for _, href := range res.Hrefs() {
			u, err := normRoot.Parse(href)
			if err != nil {
				fatalln(err)
			}

			if normurl.IsChild(normRoot, u) {
				if _, dateExists := u.Date(); dateExists {
					if _, exists := hrefMap[u.String()]; !exists {
						hrefMap[u.String()] = 1
					} else {
						hrefMap[u.String()] += 1
					}
				}
			}
		}
	}

	f, err := os.Create("index.json")
	if err != nil {
		fatalln(err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(hrefMap); err != nil {
		fatalln(err)
	}
}
