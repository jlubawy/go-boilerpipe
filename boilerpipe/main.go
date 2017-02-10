package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/jlubawy/go-boilerpipe"
	"github.com/jlubawy/go-boilerpipe/extractor"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: boilerpipe [OPTIONS] <article URL>\n\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nVersion: %s", boilerpipe.VERSION)
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	file := flag.String("file", "", "extract content from a file")
	port := flag.String("http", "", "start an HTTP server on the port specified if any (e.g. ':8080')")
	debug := flag.Bool("debug", false, "enable debug logging in the current directory")
	flag.Parse()

	if *debug {
		extractor.EnableLogging(".", true)
	}

	if *file != "" {
		// If a file path was provided then read from the file
		f, err := os.Open(*file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer f.Close()

		text, err := boilerpipe.ExtractText(f)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Print(text)

	} else if *port == "" {
		// Else if no port is provided take a URL from the command line and output the
		// results to stdout.

		url := flag.Arg(0)
		if url == "" {
			fmt.Fprintln(os.Stderr, "Must specify url.\n")
			flag.Usage()
		}

		doc, err := process(url)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if errs := doc.Errors(); len(errs) > 0 {
			for _, err := range errs {
				fmt.Fprintln(os.Stderr, "Error:", err.Error())
			}
		}

		fmt.Print(doc.Content())

	} else {
		// Else if a port is provided start the HTTP server

		http.HandleFunc("/", Handle(Index))
		http.HandleFunc("/extract", Handle(Extract))

		fmt.Fprintln(os.Stderr, "Starting server on port", *port)
		if err := http.ListenAndServe(*port, nil); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}
}

func process(url string) (*boilerpipe.TextDocument, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := boilerpipe.NewTextDocument(resp.Body)
	if err != nil {
		return nil, err
	}

	extractor.Article().Process(doc)

	return doc, nil
}

func Handle(handler func(w http.ResponseWriter, r *http.Request) (int, error)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		code, err := handler(w, r)
		if err != nil {
			http.Error(w, err.Error(), code)
		}
	}
}

var ErrMethodNotSupported = errors.New("Method not supported")

func Index(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != "GET" {
		return http.StatusMethodNotAllowed, ErrMethodNotSupported
	}

	data := struct {
		Version string
	}{
		Version: boilerpipe.VERSION,
	}

	if err := indexTempl.Execute(w, data); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func Extract(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != "GET" {
		return http.StatusMethodNotAllowed, ErrMethodNotSupported
	}

	url := r.FormValue("url")
	if url == "" {
		return http.StatusBadRequest, errors.New("Must specify url.")
	}

	doc, err := process(url)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	data := struct {
		Version string
		URL     string
		Title   string
		Content string
	}{
		Version: boilerpipe.VERSION,
		URL:     url,
		Title:   doc.Title,
		Content: doc.Content(),
	}

	if err := extractTempl.Execute(w, data); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

var indexTempl = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html>
    <head>
        <title>Boilerpipe {{.Version}}</title>

        <link rel="stylesheet" type="text/css" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" />
    </head>
    <body>
    	<div class="container">
    		<div class="row">
    			<div class="col-xs-12 col-sm-12 col-md-12 col-lg-12">
				    <nav class="navbar navbar-default">
					    <div class="container-fluid">
					        <div class="navbar-header">
					            <a class="navbar-brand" href="#">Boilerpipe {{.Version}}</a>
					        </div>
					    </div>
					</nav>
			    	<form method="GET" action="extract" target="_blank">
						<div class="form-group">
							<label for="txtUrl">Article URL</label>
							<input type="text" id="txtUrl" name="url" class="form-control" placeholder="http://www.example.com/article-url" />
						</div>
						<button type="submit" class="btn btn-default">Submit</button>
			    	</form>
		    	</div>
	    	</div>
    	</div>
    </body>
</html>`))

var extractTempl = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html>
    <head>
        <title>Boilerpipe {{.Version}} | {{.Title}}</title>

        <link rel="stylesheet" type="text/css" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" />
    </head>
    <body>
    	<div class="container">
    		<div class="row">
    			<div class="col-xs-12 col-sm-12 col-md-12 col-lg-12">
				    <nav class="navbar navbar-default">
					    <div class="container-fluid">
					        <div class="navbar-header">
					            <a class="navbar-brand" href="#">Boilerpipe {{.Version}}</a>
					        </div>
					    </div>
					</nav>
				</div>
			</div>
    		<div class="row">
    			<div class="col-xs-12 col-sm-12 col-md-12 col-lg-12">
		    		<h1>{{.Title}}</h1>
		    		<h2>{{.URL}}</h2>
					<p>{{.Content}}</p>
				</div>
			</div>
    	</div>
    </body>
</html>`))
