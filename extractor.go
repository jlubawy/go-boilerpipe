package boilerpipe

import (
	"bytes"
	"fmt"
	htemp "html/template"
	"sort"
	"strings"
)

type LoggerFunc func(stageName string, hasChanged bool, doc *Document)

var loggerFunc LoggerFunc

func EnableLogging(fn LoggerFunc) {
	loggerFunc = fn
}

func DisableLogging() {
	loggerFunc = nil
}

func EnableHTMLLogging(fn func(htmlStr string), isVerbose bool) {
	EnableLogging(func(stageName string, hasChanged bool, doc *Document) {
		var data = struct {
			StageName  string
			HasChanged bool
			Document   *Document
			IsVerbose  bool
		}{
			stageName,
			hasChanged,
			doc,
			isVerbose,
		}

		buf := &bytes.Buffer{}
		if err := templHTML.Execute(buf, &data); err != nil {
			panic(err)
		}

		fn(buf.String())
	})
}

type Extractor interface {
	Processor

	Pipeline() []Processor
}

func getStageName(i int, e Extractor, p Processor) string {
	if p == nil {
		return fmt.Sprintf("%s.%03d", e.Name(), i)
	} else {
		return fmt.Sprintf("%s.%03d.%s", e.Name(), i, p.Name())
	}
}

func defaultExtractorProcessor(e Extractor, doc *Document) bool {
	hasChanged := false

	if loggerFunc != nil {
		loggerFunc(getStageName(0, e, nil), hasChanged, doc)
	}

	for i, p := range e.Pipeline() {
		hasChanged = p.Process(doc) || hasChanged

		if loggerFunc != nil {
			loggerFunc(getStageName(i+1, e, p), hasChanged, doc)
		}
	}

	return hasChanged
}

type articleExtractor struct{}

func (e articleExtractor) Name() string { return "Article" }

func (e articleExtractor) Process(doc *Document) bool {
	return defaultExtractorProcessor(e, doc)
}

func (e articleExtractor) Pipeline() []Processor {
	return []Processor{
		TerminatingBlocks(),
		DocumentTitleMatchClassifier(),
		NumWordsRulesClassifier(),
		IgnoreBlocksAfterContent(),
		TrailingHeadlineToBoilerplate(),
		BlockProximityFusionMaxDistanceOne,
		BoilerplateBlock(),
		BlockProximityFusionMaxDistanceOneContentOnlySameTagLevel,
		KeepLargestBlocks(),
		ExpandTitleToContent(),
		LargeBlockSameTagLevelToContent(),
		ListAtEnd(),
	}
}

func Article() Processor { return articleExtractor{} }

func LabelCSV(labels map[Label]bool) string {
	ls := make([]string, len(labels))
	i := 0
	for label := range labels {
		ls[i] = string(label)
		i++
	}
	sort.Strings(ls)
	return strings.Join(ls, ", ")
}

var funcMap = htemp.FuncMap{
	"LabelCSV": LabelCSV,
}

var templHTML = htemp.Must(htemp.New("").Funcs(funcMap).Parse(`<h1>{{.StageName}}</h1>
<h2>HasChanged: {{.HasChanged}}</h2>
<div>
	<table class="table table-condensed">
		<thead>
			<th>Index</th>
			<th>Labels</th>
			<th>IsContent</th>
			<th>NumWords</th>
			<th>Text</th>
			{{if .IsVerbose}}
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
		{{range $i, $el := .Document.TextBlocks}}
			<tr{{if $el.IsContent}} class="success"{{end}}>
				<td>{{$i}}</td>
				<td>{{LabelCSV .Labels}}</td>
				<td>{{.IsContent}}</td>
				<td>{{.NumWords}}</td>
				<td>{{.Text}}</td>
				{{if $.IsVerbose}}
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
</div>`))
