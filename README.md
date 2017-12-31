# go-boilerpipe

Golang port of the [boilerpipe](https://github.com/kohlschutter/boilerpipe)
Java library written by Christian Kohlschütter.

Boilerpipe removes boilerplate and extracts text content from HTML documents.

Currently it only supports article extraction which includes the title, a
normalized URL, the date, and the content.


## Command-line

To install ```go get -u github.com/jlubawy/go-boilerpipe/...```

    > boilerpipe help

    Boilerpipe removes boilerplate and extracts text content from HTML documents.

    Usage:

           boilerpipe command [arguments]

    The commands are:

           crawl      crawl a website for HTML documents
           extract    extract text content from an HTML document
           serve      start a HTTP server for extracting text from HTML documents
           version    print boilerpipe version

    Use "boilerpipe help [command]" for more information about a command.


## Using the library

See the [boilerpipe](cmd/boilerpipe/main.go) command-line tool for an example on how to use the library.

    import (
        "fmt"

        "github.com/jlubawy/go-boilerpipe"
        "github.com/jlubawy/go-boilerpipe/extractor"
    )

    // Must provide an io.Reader (e.g. http.Response.Body) and an option *url.URL
    // which helps to extract a date for the article.
    doc, err := boilerpipe.NewDocument(r io.Reader, u *url.URL)
    if err != nil {
        return nil, err
    }

    extractor.Article().Process(doc)

    fmt.Println(doc.Title)
    fmt.Println(doc.URL) // normalized URL which can be used to test for URL equality
    fmt.Println(doc.Date)
    fmt.Print(doc.Content())
