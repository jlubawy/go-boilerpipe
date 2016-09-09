package extractor

import (
	"github.com/jlubawy/go-boilerpipe"
	"github.com/jlubawy/go-boilerpipe/filter"
)

type Extractor struct {
	name     string
	pipeline []boilerpipe.Processor
}

func (e *Extractor) Name() string { return e.name }

func (e *Extractor) Process(doc *boilerpipe.TextDocument) bool {
	hasChanged := false
	for _, p := range e.pipeline {
		hasChanged = p.Process(doc) || hasChanged
	}
	return hasChanged
}

var articleExtractor = &Extractor{
	name: "Article",
	pipeline: []boilerpipe.Processor{
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
		// ListAtEndFilter.INSTANCE.process();
	},
}

func Article() boilerpipe.Processor { return articleExtractor }
