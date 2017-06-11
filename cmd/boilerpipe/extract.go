package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/jlubawy/go-boilerpipe"
	"github.com/jlubawy/go-boilerpipe/extractor"
)

var flagTest bool

var commandExtract = &Command{
	Description: "extract text content from an HTML document",
	CommandFunc: extractFunc,
	HelpFunc:    extractHelpFunc,
}

func extractFunc(args []string) {
	flagset := flag.NewFlagSet("", flag.ExitOnError)
	flagset.Usage = extractHelpFunc
	flagset.BoolVar(&flagTest, "test", false, "output JSON document for extractor_test.go")
	flagset.Parse(args)

	if len(flagset.Args()) > 1 {
		fatalf("usage: boilerpipe extract command\n\nToo many arguments given.\n")
	}

	if len(flagset.Args()) == 0 {
		extract(os.Stdin, nil)
	} else {
		u, errURL := url.Parse(flagset.Args()[0])
		if errURL == nil {
			httpExtract(u, func(r io.Reader, err error) {
				if err != nil {
					fatalf("error: %s\n", err)
				}

				extract(r, u)
			})
		} else {
			f, errFile := os.Open(flagset.Args()[0])
			if errFile != nil {
				fatalf("error: %s\nerror:%s\n", errURL, errFile)
			}
			defer f.Close()

			extract(f, nil)
		}
	}
}

func httpExtract(u *url.URL, fn func(io.Reader, error)) {
	client := NewClient()
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
	// Read all to buffer
	buf := &bytes.Buffer{}
	if _, err := buf.ReadFrom(r); err != nil {
		fatalf("error: %s\n", err)
	}

	d := buf.Bytes()
	bytesReader := bytes.NewReader(d)

	// Get text document and extract content
	doc, err := boilerpipe.NewDocument(bytesReader, u)
	if err != nil {
		fatalf("error: %s\n", err)
	}
	extractor.Article().Process(doc)

	var v interface{}
	if flagTest {
		m := make(map[string]interface{})

		if u != nil {
			m["url"] = u.String()
		} else {
			m["url"] = ""
		}
		m["document"] = d
		m["results"] = doc.GetHTMLDocument()
		v = m
	} else {
		v = doc
	}

	if err := json.NewEncoder(os.Stdout).Encode(v); err != nil {
		fatalf("error: %s\n", err)
	}
}

func extractHelpFunc() {
	fmt.Fprint(os.Stderr, `usage: boilerpipe extract [-test=false] [document]

Extract extracts text from the provided HTML document and prints the results to
stdout.

If no argument is provided the document is read from stdin, else the argument is
parsed first as a URL and then a filename.

If -test=true a JSON document is output for the purpose of being used by extractor_test.go.
A URL should be provided as the input document so that a date can be extracted if possible.
`)
	os.Exit(1)
}
