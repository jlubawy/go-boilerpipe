package extractor

import (
	"os"
	"text/template"

	"github.com/jlubawy/go-boilerpipe"
	"github.com/jlubawy/go-boilerpipe/filter"
)

var loggingEnabled = false

func EnableLogging(enabled bool) { loggingEnabled = enabled }

type Extractor interface {
	boilerpipe.Processor

	Pipeline() []boilerpipe.Processor
}

func defaultExtractorProcessor(e Extractor, doc *boilerpipe.TextDocument) bool {
	hasChanged := false
	for _, p := range e.Pipeline() {
		hasChanged = p.Process(doc) || hasChanged

		if loggingEnabled {
			data := struct {
				Name         string
				HasChanged   bool
				TextDocument *boilerpipe.TextDocument
			}{
				p.Name(),
				hasChanged,
				doc,
			}

			if err := processorTempl.Execute(os.Stderr, &data); err != nil {
				panic(err)
			}
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

var processorTemplStr = `Processor  : {{.Name}}
HasChanged : {{.HasChanged}}
TextBlocks : {{range $i, $el := .TextDocument.TextBlocks}}{{$i}})
                Labels    : {{.Labels}}
                IsContent : {{.IsContent}}
                Text      : {{.Text}}
             {{end}}
`
var processorTempl = template.Must(template.New("").Parse(processorTemplStr))
