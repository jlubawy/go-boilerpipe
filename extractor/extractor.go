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
	return filter.TerminatingBlocks(doc)
}
