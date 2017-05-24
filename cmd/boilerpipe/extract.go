package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"

	"github.com/jlubawy/go-boilerpipe"
	"github.com/jlubawy/go-boilerpipe/extractor"

	"golang.org/x/net/publicsuffix"
)

var commandExtract = &Command{
	Description: "extract text content from an HTML document",
	CommandFunc: extractFunc,
	HelpFunc:    extractHelpFunc,
}

func extractFunc(args []string) {
	if len(args) > 1 {
		fatalf("usage: boilerpipe extract command\n\nToo many arguments given.\n")
	}

	if len(args) == 0 {
		extract(os.Stdin, nil)
	} else {
		u, errURL := url.Parse(args[0])
		if errURL == nil {
			httpExtract(u, func(r io.Reader, err error) {
				if err != nil {
					fatalf("error: %s\n", err)
				}

				extract(r, u)
			})
		} else {
			f, errFile := os.Open(args[0])
			if errFile != nil {
				fatalf("error: %s\nerror:%s\n", errURL, errFile)
			}
			defer f.Close()

			extract(f, nil)
		}
	}
}

func httpExtract(u *url.URL, fn func(io.Reader, error)) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		fn(nil, err)
		return
	}

	client := &http.Client{
		Jar: jar,
	}

	resp, err := client.Get(u.String())
	if err != nil {
		fn(nil, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		fn(nil, errors.New("received HTTP response "+resp.Status))
		return
	}

	fn(resp.Body, err)
}

func extract(r io.Reader, u *url.URL) {
	doc, err := boilerpipe.NewTextDocument(r, u)
	if err != nil {
		fatalf("error: %s\n", err)
	}

	extractor.Article().Process(doc)

	if err := json.NewEncoder(os.Stdout).Encode(doc); err != nil {
		fatalf("error: %s\n", err)
	}
}

func extractHelpFunc() {
	fmt.Fprint(os.Stderr, `usage: boilerpipe extract [document]

Extract extracts text from the provided HTML document and prints the results to
stdout.

If no argument is provided the document is read from stdin, else the argument is
parsed first as a URL and then a filename.
`)
}
