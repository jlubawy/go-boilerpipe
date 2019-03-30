package boilerpipe

import (
	"math"
	"regexp"
	"strings"
)

// A Pipeline is a collection of filters that itself satisfies the Filter
// interface.
type Pipeline struct {
	PipelineName string
	Filters      []Filter
}

// Statically check that *Pipeline satisfies the Filter interface.
var _ Filter = (*Pipeline)(nil)

// Name returns the pipeline name.
func (pipeline *Pipeline) Name() string { return pipeline.PipelineName }

// Process runs a document through the collection of filters in the pipeline.
func (pipeline *Pipeline) Process(doc *Document) (hasChanged bool) {
	for _, filter := range pipeline.Filters {
		hasChanged = filter.Process(doc) || hasChanged
	}
	return
}

var ArticlePipeline = &Pipeline{
	PipelineName: "Article",
	Filters: []Filter{
		TerminatingBlocks(),
		DocumentTitleMatchClassifier(),
		NumWordsRulesClassifier(),
		IgnoreBlocksAfterContent(),
		TrailingHeadlineToBoilerplate(),
		BlockProximityFusionMaxDistanceOne(),
		BoilerplateBlock(),
		BlockProximityFusionMaxDistanceOneContentOnlySameTagLevel(),
		KeepLargestBlocks(),
		ExpandTitleToContent(),
		LargeBlockSameTagLevelToContent(),
		ListAtEnd(),
	},
}

// Filter is the interface that processes documents and notifies if it has
// changed.
type Filter interface {
	// Name returns the name of the filter.
	Name() string

	// Process processes the document and notifies if it has been changed.
	Process(doc *Document) (hasChanged bool)
}

func TerminatingBlocks() Filter { return terminatingBlocks{} }

type terminatingBlocks struct{}

func (terminatingBlocks) Name() string { return "TerminatingBlocks" }

func (terminatingBlocks) Process(doc *Document) bool {
	hasChanged := false

	for i := range doc.TextBlocks {
		tb := doc.TextBlocks[i]

		numWords := tb.NumWords

		if numWords < 15 {  // according to boilerpipe-1.2.1-sources.jar TerminatingBlocksFinder class
			text := strings.TrimSpace(tb.Text)

			if len(text) >= 8 {
				textLC := strings.ToLower(text)

				if strings.HasPrefix(textLC, "comments") ||
					startsWithNumber(textLC, " comments", " users responded in") ||
					strings.HasPrefix(textLC, "© reuters") ||
					strings.HasPrefix(textLC, "please rate this") ||
					strings.HasPrefix(textLC, "post a comment") ||
					strings.Contains(textLC, "what you think...") ||
					strings.Contains(textLC, "add your comment") ||
					strings.Contains(textLC, "add comment") ||
					strings.Contains(textLC, "reader views") ||
					strings.Contains(textLC, "have your say") ||
					strings.Contains(textLC, "reader comments") ||
					strings.Contains(textLC, "rätta artikeln") ||
					textLC == "thanks for your comments - this feedback is now closed" {

					tb.AddLabels(LabelIndicatesEndOfText)
					hasChanged = true
				}

			} else if tb.LinkDensity() == 1.0 {
				if text == "Comment" {
					tb.AddLabels(LabelIndicatesEndOfText)
				}
			}
		}
	}

	return hasChanged
}

// startsWithNumber returns true if a string contains any of the prefixes
// after skipping over any digits.
func startsWithNumber(text string, prefixes ...string) bool {
	i := 0
	for i < len(text) && isDigit(text[i]) {
		i++
	}

	if i != 0 {
		for _, p := range prefixes {
			if strings.HasPrefix(text[i:], p) {
				return true
			}
		}
	}
	return false
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func DocumentTitleMatchClassifier() Filter { return documentTitleMatchClassifier{} }

type documentTitleMatchClassifier struct{}

func (documentTitleMatchClassifier) Name() string { return "DocumentTitleMatchClassifier" }

func (filter documentTitleMatchClassifier) Process(doc *Document) bool {
	if len(doc.Title) == 0 {
		return false
	}

	title := doc.Title
	title = strings.Replace(title, "\u00a0", " ", -1)
	title = strings.Replace(title, "'", "", -1)
	title = strings.TrimSpace(title)
	title = strings.ToLower(title)

	if len(title) == 0 {
		return false
	}

	potentialTitles := make(map[string]bool)
	potentialTitles[title] = true

	var pot string

	pot = getLongestPart(title, "[ ]*[\\|»|-][ ]*")
	if len(pot) > 0 {
		potentialTitles[pot] = true
	}
	pot = getLongestPart(title, "[ ]*[\\|»|:][ ]*")
	if len(pot) > 0 {
		potentialTitles[pot] = true
	}
	pot = getLongestPart(title, "[ ]*[\\|»|:\\(\\)][ ]*")
	if len(pot) > 0 {
		potentialTitles[pot] = true
	}
	pot = getLongestPart(title, "[ ]*[\\|»|:\\(\\)\\-][ ]*")
	if len(pot) > 0 {
		potentialTitles[pot] = true
	}
	pot = getLongestPart(title, "[ ]*[\\|»|,|:\\(\\)\\-][ ]*")
	if len(pot) > 0 {
		potentialTitles[pot] = true
	}
	pot = getLongestPart(title, "[ ]*[\\|»|,|:\\(\\)\\-\u00a0][ ]*")
	if len(pot) > 0 {
		potentialTitles[pot] = true
	}

	addPotentialTitles(potentialTitles, title, "[ ]+[\\|][ ]+", 4)
	addPotentialTitles(potentialTitles, title, "[ ]+[\\-][ ]+", 4)

	potentialTitles[removeFirst(title, " - [^\\-]+$")] = true
	potentialTitles[removeFirst(title, "^[^\\-]+ - ")] = true

	hasChanged := false

	for i := 0; i < len(doc.TextBlocks); i++ {
		tb := doc.TextBlocks[i]

		text := tb.Text
		text = strings.Replace(text, "\u00a0", " ", -1)
		text = strings.Replace(text, "'", "", -1)
		text = strings.TrimSpace(text)
		text = strings.ToLower(text)

		if _, contains := potentialTitles[text]; contains {
			tb.AddLabels(LabelTitle)
			hasChanged = true
			break
		}

		text = strings.TrimSpace(regexp.MustCompile("[\\?\\!\\.\\-\\:]+").ReplaceAllString(text, ""))
		if _, contains := potentialTitles[text]; contains {
			tb.AddLabels(LabelTitle)
			hasChanged = true
			break
		}
	}

	return hasChanged
}

func removeFirst(s string, pattern string) string {
	m := regexp.MustCompile(pattern).FindString(s)
	if len(m) == 0 {
		return s
	}
	return strings.Replace(s, m, "", 1)
}

func addPotentialTitles(potentialTitles map[string]bool, title string, pattern string, minWords int) {
	parts := strings.Split(title, " ")
	if len(parts) == 1 {
		return
	}

	for _, p := range parts {
		if strings.Contains(p, ".com") {
			continue
		}

		numWords := len(regexp.MustCompile("[\b ]+").Split(p, -1))
		if numWords >= minWords {
			potentialTitles[p] = true
		}
	}
}

func getLongestPart(title, pattern string) string {
	parts := regexp.MustCompile(pattern).Split(title, -1)
	if len(parts) == 1 {
		return ""
	}

	longestNumWords := 0
	longestPart := ""

	for _, p := range parts {
		if strings.Contains(p, ".com") {
			continue
		}

		numWords := len(regexp.MustCompile("[\b ]+").Split(p, -1))
		if numWords > longestNumWords || len(p) > len(longestPart) {
			longestNumWords = numWords
			longestPart = p
		}
	}

	if len(longestPart) == 0 {
		return ""
	}

	return strings.TrimSpace(longestPart)
}

func TrailingHeadlineToBoilerplate() Filter { return trailingHeadlineToBoilerplate{} }

type trailingHeadlineToBoilerplate struct{}

func (trailingHeadlineToBoilerplate) Name() string { return "TrailingHeadlineToBoilerplate" }

func (filter trailingHeadlineToBoilerplate) Process(doc *Document) bool {
	hasChanged := false

	for i := len(doc.TextBlocks) - 1; i >= 0; i-- {
		tb := doc.TextBlocks[i]

		if tb.IsContent {
			if tb.HasLabel(LabelHeading) {
				tb.IsContent = false
				hasChanged = true
			} else {
				break
			}
		}
	}

	return hasChanged
}

func BlockProximityFusionMaxDistanceOne() Filter {
	return &blockProximityFusionParams{"One", 1, false, false}
}

func BlockProximityFusionMaxDistanceOneSameTagLevel() Filter {
	return &blockProximityFusionParams{"OneSameTagLevel", 1, false, true}
}

func BlockProximityFusionMaxDistanceOneContentOnly() Filter {
	return &blockProximityFusionParams{"OneContentOnly", 1, true, false}
}

func BlockProximityFusionMaxDistanceOneContentOnlySameTagLevel() Filter {
	return &blockProximityFusionParams{"OneContentOnlySameTagLevel", 1, true, true}
}

type blockProximityFusionParams struct {
	suffix            string
	maxBlocksDistance int
	contentOnly       bool
	sameTagLevelOnly  bool
}

func (filter *blockProximityFusionParams) Name() string {
	return "BlockProximityFusionMaxDistance" + filter.suffix
}

func (filter *blockProximityFusionParams) Process(doc *Document) bool {
	if len(doc.TextBlocks) < 2 {
		return false
	}

	hasChanged := false

	maxBlocksDistance := filter.maxBlocksDistance
	contentOnly := filter.contentOnly
	sameTagLevelOnly := filter.sameTagLevelOnly

	var prevBlock *TextBlock
	startBlock := 0

	if contentOnly {
		for i := range doc.TextBlocks {
			tb := doc.TextBlocks[i]
			startBlock++

			if tb.IsContent {
				prevBlock = tb
				break
			}
		}

		if prevBlock == nil {
			return false
		}
	} else {
		prevBlock = doc.TextBlocks[0]
		startBlock = 1
	}

	for i := startBlock; i < len(doc.TextBlocks); i++ {
		tb := doc.TextBlocks[i]

		if tb.IsContent == false {
			prevBlock = tb
			continue
		}

		diffBlocks := tb.OffsetBlocksStart - tb.OffsetBlocksEnd - 1
		if diffBlocks <= maxBlocksDistance {
			merge := true
			if contentOnly {
				if prevBlock.IsContent == false || tb.IsContent == false {
					merge = false
				}
			}

			if merge && sameTagLevelOnly && prevBlock.TagLevel != tb.TagLevel {
				merge = false
			}

			if merge {
				prevBlock.MergeNext(tb)

				// Remove merged text block
				doc.TextBlocks = append(doc.TextBlocks[:i], doc.TextBlocks[i+1:]...)
				i--

				hasChanged = true
			} else {
				prevBlock = tb
			}
		} else {
			prevBlock = tb
		}
	}

	return hasChanged
}

func BoilerplateBlock() Filter { return boilerplateBlock{} }

type boilerplateBlock struct{}

func (boilerplateBlock) Name() string { return "BoilerplateBlock" }

func (filter boilerplateBlock) Process(doc *Document) bool {
	hasChanged := false

	for i := 0; i < len(doc.TextBlocks); i++ {
		tb := doc.TextBlocks[i]

		if tb.IsContent == false && tb.HasLabel(LabelTitle) == false {
			doc.TextBlocks = append(doc.TextBlocks[:i], doc.TextBlocks[i+1:]...)
			i--
			hasChanged = true
		}
	}

	return hasChanged
}

const expandToSameTagLevelMinimumWords = 150

func KeepLargestBlocks() Filter {
	return keepLargestBlocks{true, expandToSameTagLevelMinimumWords}
}

type keepLargestBlocks struct {
	expandToSameLevelText bool
	minWords              int
}

func (keepLargestBlocks) Name() string { return "KeepLargestBlocks" }

func (filter keepLargestBlocks) Process(doc *Document) bool {
	if len(doc.TextBlocks) < 2 {
		return false
	}

	var (
		maxNumWords  = -1
		largestBlock *TextBlock
		level        = -1
		j            = 0
		n            = -1
	)

	for i := range doc.TextBlocks {
		tb := doc.TextBlocks[i]

		if tb.IsContent {
			nw := tb.NumWords

			if nw > maxNumWords {
				largestBlock = tb
				maxNumWords = nw
				n = j

				if filter.expandToSameLevelText {
					level = tb.TagLevel
				}
			}
		}

		j++
	}

	for i := range doc.TextBlocks {
		tb := doc.TextBlocks[i]

		if tb == largestBlock {
			tb.IsContent = true
			tb.AddLabels(LabelVeryLikelyContent)
		} else {
			tb.IsContent = isLargestBlock(maxNumWords, tb)
			tb.AddLabels(LabelMightBeContent)
		}
	}

	if filter.expandToSameLevelText && n != -1 {
		// Expand to blocks below the largest
		for i := len(doc.TextBlocks) - 1; i >= 0; i-- {
			tb := doc.TextBlocks[i]

			tl := tb.TagLevel
			if tl < level {
				break
			} else if tl == level {
				if tb.NumWords >= filter.minWords {
					tb.IsContent = true
				}
			}
		}

		// Expand to blocks above the largest
		for i := range doc.TextBlocks {
			tb := doc.TextBlocks[i]

			tl := tb.TagLevel
			if tl < level {
				break
			} else if tl == level {
				if tb.NumWords >= filter.minWords {
					tb.IsContent = true
				}
			}
		}
	}

	return true
}

func isLargestBlock(maxNumWords int, tb *TextBlock) bool {
	var minWordPercent float64
	switch {
	case maxNumWords >= 1000:
		minWordPercent = 0.25
	case maxNumWords >= 500:
		minWordPercent = 0.6
	default:
		return tb.IsContent && tb.NumWords == maxNumWords
	}

	return tb.IsContent && tb.NumWords >= int(minWordPercent*float64(maxNumWords))
}

func KeepLargestFulltextBlock() Filter { return keepLargestFulltextBlock{} }

type keepLargestFulltextBlock struct{}

func (keepLargestFulltextBlock) Name() string { return "KeepLargestFulltextBlock" }

func (filter keepLargestFulltextBlock) Process(doc *Document) bool {
	if len(doc.TextBlocks) < 2 {
		return false
	}

	max := -1
	var largestBlock *TextBlock

	for i := range doc.TextBlocks {
		tb := doc.TextBlocks[i]

		if tb.IsContent == false {
			continue
		}

		numWords := getNumFullTextWords(tb)
		if numWords > max {
			largestBlock = tb
			max = numWords
		}
	}

	if largestBlock == nil {
		return false
	}

	for i := range doc.TextBlocks {
		tb := doc.TextBlocks[i]

		if tb == largestBlock {
			tb.IsContent = true
		} else {
			tb.IsContent = false
			tb.AddLabels(LabelMightBeContent)
		}
	}

	return true
}

func ExpandTitleToContent() Filter { return expandTitleToContent{} }

type expandTitleToContent struct{}

func (expandTitleToContent) Name() string { return "ExpandTitleToContent" }

func (filter expandTitleToContent) Process(doc *Document) bool {
	j := 0
	title := -1
	contentStart := -1

	for i := range doc.TextBlocks {
		tb := doc.TextBlocks[i]

		if contentStart == -1 && tb.HasLabel(LabelTitle) {
			title = j
			contentStart = -1
		}

		if contentStart == -1 && tb.IsContent {
			contentStart = j
		}

		j++
	}

	if contentStart <= title || title == -1 {
		return false
	}

	hasChanged := false
	for i := range doc.TextBlocks[title:contentStart] {
		tb := doc.TextBlocks[i]

		if tb.HasLabel(LabelMightBeContent) {
			hasChanged = (tb.IsContent == false) || hasChanged
			tb.IsContent = true
		}
	}

	return hasChanged
}

func LargeBlockSameTagLevelToContent() Filter { return largeBlockSameTagLevelToContent{} }

type largeBlockSameTagLevelToContent struct{}

func (largeBlockSameTagLevelToContent) Name() string { return "LargeBlockSameTagLevelToContent" }

func (filter largeBlockSameTagLevelToContent) Process(doc *Document) bool {
	hasChanged := false
	tagLevel := -1

	for i := range doc.TextBlocks {
		tb := doc.TextBlocks[i]

		if tb.IsContent && tb.HasLabel(LabelVeryLikelyContent) {
			tagLevel = tb.TagLevel
			break
		}
	}

	if tagLevel == -1 {
		return false
	}

	for i := range doc.TextBlocks {
		tb := doc.TextBlocks[i]

		if tb.IsContent == false {
			if tb.NumWords >= 100 && tb.TagLevel == tagLevel {
				tb.IsContent = true
				hasChanged = true
			}
		}
	}

	return hasChanged
}

const defaultMinNumberOfWords = 60

func IgnoreBlocksAfterContent() Filter {
	return ignoreBlocksAfterContent{defaultMinNumberOfWords}
}

type ignoreBlocksAfterContent struct{ minNumWords int }

func (ignoreBlocksAfterContent) Name() string { return "IgnoreBlocksAfterContent" }

func (filter ignoreBlocksAfterContent) Process(doc *Document) bool {
	hasChanged := false
	numWords := 0
	foundEndOfText := false

	for i := range doc.TextBlocks {
		tb := doc.TextBlocks[i]

		eot := tb.HasLabel(LabelIndicatesEndOfText)

		if tb.IsContent {
			numWords += getNumFullTextWords(tb)
		}
		if eot && numWords >= filter.minNumWords {
			foundEndOfText = true
		}
		if foundEndOfText {
			hasChanged = true
			tb.IsContent = false
		}
	}

	return hasChanged
}

func NumWordsRulesClassifier() Filter { return numWordsRulesClassifier{} }

type numWordsRulesClassifier struct{}

func (numWordsRulesClassifier) Name() string { return "NumWordsRulesClassifier" }

func (filter numWordsRulesClassifier) Process(doc *Document) bool {
	hasChanged := false

	if len(doc.TextBlocks) == 0 {
		return false
	}

	prevBlock := textBlockEmptyStart
	currentBlock := doc.TextBlocks[0]
	var nextBlock *TextBlock

	if len(doc.TextBlocks) >= 2 {
		nextBlock = doc.TextBlocks[1]
	} else {
		nextBlock = textBlockEmptyStart
	}

	hasChanged = classify(prevBlock, currentBlock, nextBlock) || hasChanged

	if nextBlock != textBlockEmptyStart {
		for i := 2; i < len(doc.TextBlocks); i++ {
			prevBlock = currentBlock
			currentBlock = nextBlock
			nextBlock = doc.TextBlocks[i]
			hasChanged = classify(prevBlock, currentBlock, nextBlock) || hasChanged
		}
		prevBlock = currentBlock
		currentBlock = nextBlock
		nextBlock = textBlockEmptyEnd
		hasChanged = classify(prevBlock, currentBlock, nextBlock) || hasChanged
	}

	return hasChanged
}

func classify(prev, curr, next *TextBlock) bool {
	isContent := false

	if curr.LinkDensity() <= 0.333333 {
		if prev.LinkDensity() <= 0.555556 {
			if curr.NumWords <= 16 {
				if next.NumWords <= 15 {
					if prev.NumWords <= 4 {
						isContent = false
					} else {
						isContent = true
					}
				} else {
					isContent = true
				}
			} else {
				isContent = true
			}
		} else {
			if curr.NumWords <= 40 {
				if next.NumWords <= 17 {
					isContent = false
				} else {
					isContent = true
				}
			} else {
				isContent = true
			}
		}
	} else {
		isContent = false
	}

	curr.IsContent = isContent
	return isContent
}

func getNumFullTextWords(tb *TextBlock) int {
	minTextDensity := 9.0

	if tb.TextDensity() >= minTextDensity {
		return tb.NumWords
	} else {
		return 0
	}
}

func ListAtEnd() Filter { return listAtEnd{} }

type listAtEnd struct{}

func (listAtEnd) Name() string { return "ListAtEnd" }

func (filter listAtEnd) Process(doc *Document) bool {
	hasChanged := false
	tagLevel := math.MaxInt32

	for i := range doc.TextBlocks {
		tb := doc.TextBlocks[i]

		if tb.IsContent && tb.HasLabel(LabelVeryLikelyContent) {
			tagLevel = tb.TagLevel
		} else {
			if tb.TagLevel > tagLevel && tb.HasLabel(LabelMightBeContent) &&
				tb.HasLabel(LabelList) && tb.LinkDensity() == 0.0 {
				tb.IsContent = true
				hasChanged = true
			} else {
				tagLevel = math.MaxInt32
			}
		}
	}

	return hasChanged

}
