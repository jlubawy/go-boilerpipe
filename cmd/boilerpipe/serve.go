package main

import (
	"errors"
	"flag"
	"fmt"
	htemp "html/template"
	"io"
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

	if err := ParseTemplates(); err != nil {
		fatalf("Error parsing templates: %v\n", err)
	}

	http.HandleFunc("/", runHandler(indexHandler))
	http.HandleFunc("/extract", runHandler(extractHandler))

	fmt.Fprintf(os.Stderr, "Listening on port %d\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		fatalf("Error starting server: %v\n", err)
	}
}

func serveHelpFunc() {
	fmt.Fprint(os.Stderr, `usage: boilerpipe serve [-port=8080]

Serve starts an HTTP server listening on the provided port.
`)
	os.Exit(1)
}

func runHandler(handler func(w http.ResponseWriter, req *http.Request) (int, error)) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		code, err := handler(w, req)
		if err != nil {
			data := map[string]interface{}{
				"Status": http.StatusText(code),
				"Error":  err,
			}
			if err := Execute("error", w, data); err != nil {
				panic(err)
			}
		}

		var uriStr string
		s, err := url.QueryUnescape(req.RequestURI)
		if err != nil {
			uriStr = req.RequestURI
		} else {
			uriStr = s
		}

		fmt.Fprintf(os.Stderr, "[%s] \"%s %s %s\" %d\n",
			time.Now(),
			req.Method,
			uriStr,
			req.Proto,
			code,
		)
	}
}

var ErrMethodNotSupported = errors.New("method not supported")

func indexHandler(w http.ResponseWriter, req *http.Request) (int, error) {
	if req.Method != http.MethodGet {
		return http.StatusMethodNotAllowed, ErrMethodNotSupported
	}

	if err := Execute("index", w, nil); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func extractHandler(w http.ResponseWriter, req *http.Request) (int, error) {
	if req.Method != http.MethodGet {
		return http.StatusMethodNotAllowed, ErrMethodNotSupported
	}

	rawurl := req.FormValue("url")
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

	pipelineFilter := &LoggingPipeline{
		Pipeline:   boilerpipe.NewArticlePipeline(),
		LogEntries: make([]LogEntry, 0),
	}

	doc, err := boilerpipe.NewDocument(rc, u)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	pipelineFilter.Process(doc)

	data := map[string]interface{}{
		"Content":        htemp.HTML(doc.HTML()),
		"Date":           doc.Date.Format("January 2, 2006"),
		"Doc":            doc,
		"pipelineFilter": pipelineFilter,
		"RawURL":         rawurl,
	}
	if err := Execute("extract", w, data); err != nil {
		panic(err)
	}

	return http.StatusOK, nil
}

type LogEntry struct {
	FilterName string
	Document   boilerpipe.Document
	HasChanged bool
}

type LoggingPipeline struct {
	Pipeline   *boilerpipe.Pipeline
	LogEntries []LogEntry
}

var _ boilerpipe.Filter = (*LoggingPipeline)(nil)

func (pipeline *LoggingPipeline) Name() string { return pipeline.Pipeline.Name() }

func (pipeline *LoggingPipeline) Process(doc *boilerpipe.Document) (hasChanged bool) {
	pipeline.LogEntries = append(pipeline.LogEntries, LogEntry{
		FilterName: fmt.Sprintf("%s.000", pipeline.Pipeline.Name()),
		Document:   *doc,
		HasChanged: false,
	})

	for i, filter := range pipeline.Pipeline.Filters {
		hasChanged = filter.Process(doc) || hasChanged

		pipeline.LogEntries = append(pipeline.LogEntries, LogEntry{
			FilterName: fmt.Sprintf("%s.%03d.%s", pipeline.Pipeline.Name(), i+1, filter.Name()),
			Document:   *doc,
			HasChanged: hasChanged,
		})
	}
	return
}

var templateMap = make(map[string]*htemp.Template)

func ParseTemplates() error {
	for name, s := range templStrs {
		rootTempl, err := htemp.New("").Parse(templRootStr)
		if err != nil {
			return err
		}

		t, err := rootTempl.Parse(s)
		if err != nil {
			return fmt.Errorf("template '%s': %v\n", name, err)
		}
		templateMap[name] = t
	}

	return nil
}

func Execute(name string, w io.Writer, data map[string]interface{}) error {
	if data == nil {
		data = make(map[string]interface{})
	}

	{
		data["version"] = boilerpipe.Version
	}

	t, exists := templateMap[name]
	if !exists {
		return fmt.Errorf("template %s does not exist")
	}

	t = t.Lookup("Root")
	if t == nil {
		return fmt.Errorf("Root template not found")
	}

	return t.Execute(w, data)
}

var templRootStr = `{{define "Root"}}<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8">

    <link rel="stylesheet" type="text/css" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" />

    <title>Boilerpipe {{.version}}</title>
  </head>

  <body>
    <div class="container">
      <div class="row">
        <div class="col-xs-12">
          <nav class="navbar navbar-default">
            <div class="container-fluid">
              <div class="navbar-header">
                <a class="navbar-brand" href="/">Boilerpipe {{.version}}</a>
              </div>
            </div>
          </nav>
        </div>
      </div><!-- row -->
      {{template "Body" $}}
    </div><!-- container -->

  </body>
</html>
{{end}}`

var templStrs = map[string]string{
	"index": `{{define "Body"}}<div class="row">
  <div class="col-xs-12">
    <form method="GET" action="extract">
      <div class="form-group">
        <label for="txtUrl">Article URL</label>
        <input type="text" id="txtUrl" name="url" class="form-control" placeholder="http://www.example.com/article-url" />
      </div>
      <button type="submit" class="btn btn-default">Submit</button>
    </form>
  </div><!-- col -->
</div><!-- row -->
{{end}}`,

	"extract": `{{define "Body"}}<div class="row">
  <div class="col-xs-12">
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
  </div><!-- col -->
</div><!-- row -->
{{range $.pipelineFilter.LogEntries}}
<div class="row">
  <div class="col-xs-12">
    {{.FilterName}} - {{.HasChanged}}
  </div><!-- col -->
</div><!-- row -->
{{end}}{{end}}`,

	"error": `{{define "Body"}}<div class="row">
  <div class="col-xs-12">
    <h1>{{.Status}}</h1>
    <p>{{.Error}}</p>
  </div><!-- col -->
</div><!-- row -->
{{end}}`,
}
