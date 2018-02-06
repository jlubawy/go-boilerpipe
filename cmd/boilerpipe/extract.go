package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"

	"github.com/jlubawy/go-boilerpipe"
	"github.com/jlubawy/go-boilerpipe/normurl"

	"golang.org/x/net/publicsuffix"
)

var (
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
	flagset.BoolVar(&FlagPrettyPrint, "pretty-print", false, "pretty print JSON output")
	flagset.Parse(args)

	if len(flagset.Args()) > 1 {
		fatalf("usage: boilerpipe extract command\n\nToo many arguments given.\n")
	}

	argDocumentPath := flagset.Arg(0)

	var (
		r io.Reader
		u *normurl.URL
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
			u, err = normurl.Parse(argDocumentPath)
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

func NewClient() *http.Client {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		fatalf("Error creating cookie jar: %v\n", err)
	}

	return &http.Client{
		Jar: jar,
	}
}

func httpGet(urlStr string) (io.ReadCloser, error) {
	resp, err := NewClient().Get(urlStr)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, errors.New("received HTTP response " + resp.Status)
	}

	return resp.Body, nil
}

func extract(r io.Reader, u *normurl.URL) {
	var (
		doc *boilerpipe.Document
		b   []byte
		err error
	)

	// Get text document and extract content
	doc, err = boilerpipe.ParseDocument(r)
	if err != nil {
		fatalf("Error creating new document: %v\n", err)
	}
	boilerpipe.ArticlePipline.Process(doc)

	if FlagPrettyPrint {
		b, err = json.MarshalIndent(doc, "", "  ")
	} else {
		b, err = json.Marshal(doc)
	}
	if err != nil {
		fatalf("Error encoding JSON: %v\n", err)
	}

	io.Copy(os.Stdout, bytes.NewReader(b))
}

func extractHelpFunc() {
	fmt.Fprint(os.Stderr, `usage: boilerpipe extract [document path]

Extract extracts text from the provided HTML document and prints the results to
stdout.

If no argument is provided the document is read from stdin, else the argument is
parsed first as a URL and then a filename.
`)
	os.Exit(1)
}
