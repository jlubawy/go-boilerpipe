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
	// new DocumentTitleMatchClassifier(doc.getTitle()).process(doc)
	hasChanged = filter.NumWordsRulesClassifier(doc) || hasChanged
	hasChanged = filter.IgnoreBlocksAfterContent(doc) || hasChanged
	hasChanged = filter.TrailingHeadlineToBoilerplate(doc) || hasChanged
	hasChanged = filter.BlockProximityFusionMaxDistanceOne.Process(doc) || hasChanged
	// BoilerplateBlockFilter.INSTANCE_KEEP_TITLE.process(doc)
	hasChanged = filter.BlockProximityFusionMaxDistanceOneContentOnlySameTagLevel.Process(doc) || hasChanged
	hasChanged = filter.KeepLargestBlock(doc) || hasChanged
	// ExpandTitleToContentFilter.INSTANCE.process(doc)
	// LargeBlockSameTagLevelToContentFilter.INSTANCE.process(doc)
	// ListAtEndFilter.INSTANCE.process(doc);
	return hasChanged
}
