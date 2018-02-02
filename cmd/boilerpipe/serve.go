package main

import (
	"errors"
	"flag"
	"fmt"
	htemp "html/template"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/jlubawy/go-boilerpipe"
)

var commandServe = &Command{
	Description: "start a HTTP server for extracting text from HTML documents",
	CommandFunc: serveFunc,
	HelpFunc:    serveHelpFunc,
}

func serveFunc(args []string) {
	var port uint

	flagset := flag.NewFlagSet("", flag.ExitOnError)
	flagset.Usage = serveHelpFunc
	flagset.UintVar(&port, "port", 8080, "TCP port to listen on")
	flagset.Parse(args)

	if len(flag.Args()) > 0 {
		fatalf("usage: boilerpipe serve command\n\nToo many arguments given.\n")
	}

	http.HandleFunc("/", runHandler(indexHandler))
	http.HandleFunc("/extract", runHandler(extractHandler))

	fmt.Fprintf(os.Stderr, "Listening on port %d\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		fatalf("error: %s\n", err)
	}
}

func serveHelpFunc() {
	fmt.Fprint(os.Stderr, `usage: boilerpipe serve [-port=8080]

Serve starts an HTTP server listening on the provided port.
`)
	os.Exit(1)
}

func runHandler(handler func(w http.ResponseWriter, r *http.Request) (int, error)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		code, err := handler(w, r)
		if err != nil {
			var data = struct {
				Version string
				Status  string
				Error   error
			}{
				Version: boilerpipe.Version,
				Status:  http.StatusText(code),
				Error:   err,
			}
			if err := templError.Execute(w, &data); err != nil {
				panic(err)
			}
		}

		var uriStr string
		s, err := url.QueryUnescape(r.RequestURI)
		if err != nil {
			uriStr = r.RequestURI
		} else {
			uriStr = s
		}

		fmt.Fprintf(os.Stderr, "[%s] \"%s %s %s\" %d\n",
			time.Now(),
			r.Method,
			uriStr,
			r.Proto,
			code,
		)
	}
}

var ErrMethodNotSupported = errors.New("method not supported")

func indexHandler(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed, ErrMethodNotSupported
	}

	data := struct {
		Version string
	}{
		Version: boilerpipe.Version,
	}

	if err := templIndex.Execute(w, data); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func extractHandler(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed, ErrMethodNotSupported
	}

	rawurl := r.FormValue("url")
	if rawurl == "" {
		return http.StatusBadRequest, errors.New("Must specify url.")
	}

	u, err := url.Parse(rawurl)
	if err != nil {
		return http.StatusBadRequest, err
	}

	rc, err := httpGet(rawurl)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer rc.Close()

	extractLogs := make([]htemp.HTML, 0)

	doc, err := boilerpipe.NewDocument(rc, u)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	boilerpipe.ArticlePipeline().Process(doc)

	data := map[string]interface{}{
		"Content":     htemp.HTML(doc.HTML()),
		"Date":        doc.Date.Format("January 2, 2006"),
		"Doc":         doc,
		"ExtractLogs": extractLogs,
		"RawURL":      rawurl,
		"Version":     boilerpipe.Version,
	}

	if err := templExtract.Execute(w, &data); err != nil {
		panic(err)
	}

	return http.StatusOK, nil
}

var templIndex = htemp.Must(htemp.New("").Parse(`<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8">

    <link rel="stylesheet" type="text/css" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" />

    <title>Boilerpipe {{.Version}}</title>
  </head>
  <body>
    <div class="container">
      <div class="row">
        <div class="col-xs-12 col-sm-12 col-md-12 col-lg-12">
          <nav class="navbar navbar-default">
            <div class="container-fluid">
              <div class="navbar-header">
                <a class="navbar-brand" href="/">Boilerpipe {{.Version}}</a>
              </div>
            </div>
          </nav>
          <form method="GET" action="extract">
            <div class="form-group">
              <label for="txtUrl">Article URL</label>
              <input type="text" id="txtUrl" name="url" class="form-control" placeholder="http://www.example.com/article-url" />
            </div>
            <div class="checkbox">
              <label>
                <input type="checkbox" name="logging"> Enable logging?
              </label>
            </div>
            <button type="submit" class="btn btn-default">Submit</button>
          </form>
        </div>
      </div>
    </div>
  </body>
</html>`))

var templExtract = htemp.Must(htemp.New("").Parse(`<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8">

    <link rel="stylesheet" type="text/css" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" />

    <title>Boilerpipe {{.Version}} | {{.Doc.Title}}</title>
  </head>
  <body>
    <div class="container">
      <div class="row">
        <div class="col-xs-12 col-sm-12 col-md-12 col-lg-12">
          <nav class="navbar navbar-default">
            <div class="container-fluid">
              <div class="navbar-header">
                <a class="navbar-brand" href="/">Boilerpipe {{.Version}}</a>
              </div>
            </div>
          </nav>
        </div>
      </div>
      <div class="row">
        <div class="col-xs-12 col-sm-12 col-md-12 col-lg-12">
          <dl class="dl-horizontal">
            <dt>Title</dt>
            <dd><a href="{{.RawURL}}" target="_blank">{{.Doc.Title}}</a></dd>

            <dt>Date</dt>
            <dd>{{.Date}}</dd>

            <dt>URL</dt>
            <dd>{{.Doc.URL}}</dd>

            <dt>Content</dt>
            <dd>{{.Content}}</dd>
          </dl>
        </div>
      </div>
{{range .ExtractLogs}}
      <div class="row">
        <div class="col-xs-12 col-sm-12 col-md-12 col-lg-12">
          {{.}}
        </div>
      </div>
{{end}}
    </div>
  </body>
</html>`))

var templError = htemp.Must(htemp.New("").Parse(`<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8">

    <link rel="stylesheet" type="text/css" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" />

    <title>Boilerpipe {{.Version}}</title>
  </head>
  <body>
    <div class="container">
      <div class="row">
        <div class="col-xs-12 col-sm-12 col-md-12 col-lg-12">
          <nav class="navbar navbar-default">
            <div class="container-fluid">
              <div class="navbar-header">
                <a class="navbar-brand" href="/">Boilerpipe {{.Version}}</a>
              </div>
            </div>
          </nav>
        </div>
      </div>
      <div class="row">
        <div class="col-xs-12 col-sm-12 col-md-12 col-lg-12">
          <h1>{{.Status}}</h1>
          <p>{{.Error}}</p>
        </div>
      </div>
    </div>
  </body>
</html>`))
