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
)

var (
	FlagTest        bool
	FlagPrettyPrint bool
)

var commandExtract = &Command{
	Description: "extract text content from an HTML document",
	CommandFunc: extractFunc,
	HelpFunc:    extractHelpFunc,
}

func extractFunc(args []string) {
	flagset := flag.NewFlagSet("", flag.ExitOnError)
	flagset.Usage = extractHelpFunc
	flagset.BoolVar(&FlagTest, "test", false, "output JSON document for extractor_test.go")
	flagset.BoolVar(&FlagPrettyPrint, "pretty-print", false, "pretty print JSON output")
	flagset.Parse(args)

	if len(flagset.Args()) > 1 {
		fatalf("usage: boilerpipe extract command\n\nToo many arguments given.\n")
	}

	argDocumentPath := flagset.Arg(0)

	var (
		r io.Reader
		u *url.URL
	)

	if argDocumentPath == "" {
		// If no URL is provided, read from stdin
		r = os.Stdin

	} else {
		// Else attempt to parse the URL or path

		if _, err := os.Stat(argDocumentPath); err == nil {
			// If no error then the path is likely to a local file
			f, err := os.Open(flagset.Args()[0])
			if err != nil {
				fatalf("Error opening file: %v\n", err)
			}
			defer f.Close()
			r = f

		} else {
			// Else it's likely a URL
			var err error
			u, err = url.Parse(argDocumentPath)
			if err != nil {
				fatalf("Error parsing URL: %v\n", err)
			}

			rc, err := httpGet(argDocumentPath)
			if err != nil {
				fatalf("Error getting document: %v\n", err)
			}
			defer rc.Close()
			r = rc
		}
	}

	extract(r, u)
}

func httpGet(urlStr string) (io.ReadCloser, error) {
	client := NewClient()
	resp, err := client.Get(urlStr)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, errors.New("received HTTP response " + resp.Status)
	}

	return resp.Body, nil
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
	boilerpipe.Article().Process(doc)

	var v interface{}
	if FlagTest {
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

	enc := json.NewEncoder(os.Stdout)
	if FlagPrettyPrint {
		enc.SetIndent("", "  ")
	}

	if err := enc.Encode(v); err != nil {
		fatalf("error: %s\n", err)
	}
}

func extractHelpFunc() {
	fmt.Fprint(os.Stderr, `usage: boilerpipe extract [-test=false] [document path]

Extract extracts text from the provided HTML document and prints the results to
stdout.

If no argument is provided the document is read from stdin, else the argument is
parsed first as a URL and then a filename.

If -test=true a JSON document is output for the purpose of being used by extractor_test.go.
A URL should be provided as the input document so that a date can be extracted if possible.
`)
	os.Exit(1)
}
