package extractor

import (
	"github.com/jlubawy/go-boilerpipe"
	"github.com/jlubawy/go-boilerpipe/filter"
)

type article struct{}

func Article(doc *boilerpipe.TextDocument) bool {
	a := article{}
	return a.Process(doc)
}

func (article) Process(doc *boilerpipe.TextDocument) bool {
	hasChanged := filter.TerminatingBlocks(doc)
	hasChanged = filter.DocumentTitleMatchClassifier(doc) || hasChanged
	hasChanged = filter.NumWordsRulesClassifier(doc) || hasChanged
	hasChanged = filter.IgnoreBlocksAfterContent(doc) || hasChanged
	hasChanged = filter.TrailingHeadlineToBoilerplate(doc) || hasChanged
	hasChanged = filter.BlockProximityFusionMaxDistanceOne.Process(doc) || hasChanged
	hasChanged = filter.BoilerplateBlock(doc) || hasChanged
	hasChanged = filter.BlockProximityFusionMaxDistanceOneContentOnlySameTagLevel.Process(doc) || hasChanged
	hasChanged = filter.KeepLargestBlock(doc) || hasChanged
	//hasChanged = filter.ExpandTitleToContent(doc) || hasChanged
	hasChanged = filter.LargeBlockSameTagLevelToContent(doc) || hasChanged
	// ListAtEndFilter.INSTANCE.process(doc);
	return hasChanged
}
