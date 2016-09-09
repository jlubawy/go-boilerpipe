package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/jlubawy/go-boilerpipe"
	"github.com/jlubawy/go-boilerpipe/extractor"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "must specify url")
		os.Exit(1)
	}

	resp, err := http.Get(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	doc, err := boilerpipe.NewTextDocument(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	isChanged := extractor.Article().Process(doc)

	fmt.Println("isChanged:", isChanged)
	fmt.Println("Title:", doc.Title)
	fmt.Println("Content:", doc.Content())
	//fmt.Println("Text:", doc.Text(true, true))
}
