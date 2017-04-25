package extractor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/jlubawy/go-boilerpipe"
	"github.com/jlubawy/go-boilerpipe/filter"
)

var loggingPath string
var loggingVerbose bool

func EnableLogging(path string, verbose bool) {
	createLoggingDir(path)
	loggingPath = path
	loggingVerbose = verbose
}

func createLoggingDir(path string) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0664); err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	} else if info.IsDir() == false {
		panic(fmt.Errorf("extractor: path '%s' is not a directory", path))
	}
}

type Extractor interface {
	boilerpipe.Processor

	Pipeline() []boilerpipe.Processor
}

func defaultExtractorProcessor(e Extractor, doc *boilerpipe.TextDocument) bool {
	hasChanged := false

	if loggingPath != "" {
		logToFile(e.Name(), hasChanged, doc)
	}

	for i, p := range e.Pipeline() {
		hasChanged = p.Process(doc) || hasChanged

		if loggingPath != "" {
			name := fmt.Sprintf("%s.%03d.%s", e.Name(), i, p.Name())
			logToFile(name, hasChanged, doc)
		}
	}

	return hasChanged
}

type articleExtractor struct{}

func (e articleExtractor) Name() string { return "Article" }

func (e articleExtractor) Process(doc *boilerpipe.TextDocument) bool {
	return defaultExtractorProcessor(e, doc)
}

func (e articleExtractor) Pipeline() []boilerpipe.Processor {
	return []boilerpipe.Processor{
		filter.TerminatingBlocks(),
		filter.DocumentTitleMatchClassifier(),
		filter.NumWordsRulesClassifier(),
		filter.IgnoreBlocksAfterContent(),
		filter.TrailingHeadlineToBoilerplate(),
		filter.BlockProximityFusionMaxDistanceOne,
		filter.BoilerplateBlock(),
		filter.BlockProximityFusionMaxDistanceOneContentOnlySameTagLevel,
		filter.KeepLargestBlock(),
		filter.ExpandTitleToContent(),
		filter.LargeBlockSameTagLevelToContent(),
		filter.ListAtEnd(),
	}
}

func Article() boilerpipe.Processor { return articleExtractor{} }

func logToFile(name string, hasChanged bool, doc *boilerpipe.TextDocument) {
	path := filepath.Join(loggingPath, name+".html")
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}

	data := struct {
		Name         string
		HasChanged   bool
		TextDocument *boilerpipe.TextDocument
		Verbose      bool
	}{
		name,
		hasChanged,
		doc,
		loggingVerbose,
	}

	if err := processorTempl.Execute(f, &data); err != nil {
		panic(err)
	}

	f.Close()
}

var processorTemplStr = `<!DOCTYPE html>
<html>
	<head>
		<title>{{.Name}}</title>

		<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" />
	</head>
	<body>
		<h1>{{.Name}}</h1>
		<h2>HasChanged: {{.HasChanged}}</h2>
		<div>
			<table class="table table-condensed">
				<thead>
					<th>Index</th>
					<th>Labels</th>
					<th>IsContent</th>
					<th>NumWords</th>
					<th>Text</th>
					{{if .Verbose}}
					<th>OffsetBlocksStart</th>
					<th>OffsetBlocksEnd</th>
					<th>NumLinkedWords</th>
					<th>NumWordsInWrappedLines</th>
					<th>NumWrappedLines</th>
					<th>TagLevel</th>
					<th>TextDensity</th>
					<th>LinkDensity</th>
					{{end}}
				</thead>
				<tbody>
				{{$verbose := .Verbose}}
				{{range $i, $el := .TextDocument.TextBlocks}}
					<tr{{if $el.IsContent}} class="success"{{end}}>
						<td>{{$i}}</td>
						<td>{{LabelCSV .Labels}}</td>
						<td>{{.IsContent}}</td>
						<td>{{.NumWords}}</td>
						<td>{{.Text}}</td>
						{{if $verbose}}
						<td>{{.OffsetBlocksStart}}</td>
						<td>{{.OffsetBlocksEnd}}</td>
						<td>{{.NumLinkedWords}}</td>
						<td>{{.NumWordsInWrappedLines}}</td>
						<td>{{.NumWrappedLines}}</td>
						<td>{{.TagLevel}}</td>
						<td>{{.TextDensity}}</td>
						<td>{{.LinkDensity}}</td>
						{{end}}
					</tr>
	            {{end}}
	            </tbody>
	        </table>
        </div>
    </body>
</html>
`

func LabelCSV(labels map[boilerpipe.Label]bool) string {
	ls := make([]string, len(labels))
	i := 0
	for label := range labels {
		ls[i] = string(label)
		i++
	}
	sort.Strings(ls)
	return strings.Join(ls, ", ")
}

var funcMap = template.FuncMap{
	"LabelCSV": LabelCSV,
}

var processorTempl = template.Must(template.New("").Funcs(funcMap).Parse(processorTemplStr))
