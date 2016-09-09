package filter

import (
	"log"
	"regexp"
	"strings"

	"github.com/jlubawy/go-boilerpipe"
)

func TerminatingBlocks() boilerpipe.Processor { return terminatingBlocks{} }

type terminatingBlocks struct{}

func (terminatingBlocks) Name() string { return "TerminatingBlocks" }

func (terminatingBlocks) Process(doc *boilerpipe.TextDocument) bool {
	hasChanged := false

	for i := range doc.TextBlocks {
		tb := doc.TextBlocks[i]

		numWords := tb.NumWords

		if numWords < 15 {
			text := tb.Text

			if len(text) >= 8 {
				textLC := strings.ToLower(text)

				if strings.HasPrefix(textLC, "comments") ||
					//|| startsWithNumber(textLC, len, " comments", " users responded in")
					strings.HasPrefix(textLC, "© reuters") || strings.HasPrefix(textLC, "please rate this") ||
					strings.HasPrefix(textLC, "post a comment") || strings.Contains(textLC, "what you think...") ||
					strings.Contains(textLC, "add your comment") || strings.Contains(textLC, "add comment") ||
					strings.Contains(textLC, "reader views") || strings.Contains(textLC, "have your say") ||
					strings.Contains(textLC, "reader comments") || strings.Contains(textLC, "rätta artikeln") ||
					textLC == "thanks for your comments - this feedback is now closed" {

					tb.AddLabel(boilerpipe.LabelIndicatesEndOfText)
					hasChanged = true
				}

			} else if tb.LinkDensity == 1.0 {
				if text == "Comment" {
					tb.AddLabel(boilerpipe.LabelIndicatesEndOfText)
				}
			}
		}
	}

	return hasChanged
}

//  TODO:
// /**
//   * Checks whether the given text t starts with a sequence of digits, followed by one of the given
//   * strings.
//   *
//   * @param t The text to examine
//   * @param len The length of the text to examine
//   * @param str Any strings that may follow the digits.
//   * @return true if at least one combination matches
//   */
//  private static boolean startsWithNumber(final String t, final int len, final String... str) {
//    int j = 0;
//    while (j < len && isDigit(t.charAt(j))) {
//      j++;
//    }
//    if (j != 0) {
//      for (String s : str) {
//        if (t.startsWith(s, j)) {
//          return true;
//        }
//      }
//    }
//    return false;
//  }
//
//  private static boolean isDigit(final char c) {
//    return c >= '0' && c <= '9';
//  }

func DocumentTitleMatchClassifier() boilerpipe.Processor { return documentTitleMatchClassifier{} }

type documentTitleMatchClassifier struct{}

func (documentTitleMatchClassifier) Name() string { return "DocumentTitleMatchClassifier" }

func (p documentTitleMatchClassifier) Process(doc *boilerpipe.TextDocument) bool {
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
			tb.AddLabel(boilerpipe.LabelTitle)
			hasChanged = true
			break
		}

		text = strings.TrimSpace(regexp.MustCompile("[\\?\\!\\.\\-\\:]+").ReplaceAllString(text, ""))
		if _, contains := potentialTitles[text]; contains {
			tb.AddLabel(boilerpipe.LabelTitle)
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

func TrailingHeadlineToBoilerplate() boilerpipe.Processor { return trailingHeadlineToBoilerplate{} }

type trailingHeadlineToBoilerplate struct{}

func (trailingHeadlineToBoilerplate) Name() string { return "TrailingHeadlineToBoilerplate" }

func (p trailingHeadlineToBoilerplate) Process(doc *boilerpipe.TextDocument) bool {
	hasChanged := false

	for i := len(doc.TextBlocks) - 1; i >= 0; i-- {
		tb := doc.TextBlocks[i]

		if tb.IsContent {
			if tb.HasLabel(boilerpipe.LabelHeading) {
				tb.IsContent = false
				hasChanged = true
			} else {
				break
			}
		}
	}

	return hasChanged
}

var (
	BlockProximityFusionMaxDistanceOne                        = &blockProximityFusionParams{"BlockProximityFusionMaxDistanceOne", 1, false, false}
	BlockProximityFusionMaxDistanceOneSameTagLevel            = &blockProximityFusionParams{"BlockProximityFusionMaxDistanceOneSameTagLevel", 1, false, true}
	BlockProximityFusionMaxDistanceOneContentOnly             = &blockProximityFusionParams{"BlockProximityFusionMaxDistanceOneContentOnly", 1, true, false}
	BlockProximityFusionMaxDistanceOneContentOnlySameTagLevel = &blockProximityFusionParams{"BlockProximityFusionMaxDistanceOneContentOnlySameTagLevel", 1, true, true}
)

type blockProximityFusionParams struct {
	name              string
	maxBlocksDistance int
	contentOnly       bool
	sameTagLevelOnly  bool
}

func (p *blockProximityFusionParams) Name() string { return p.name }

func (p *blockProximityFusionParams) Process(doc *boilerpipe.TextDocument) bool {
	hasChanged := false

	maxBlocksDistance := p.maxBlocksDistance
	contentOnly := p.contentOnly
	sameTagLevelOnly := p.sameTagLevelOnly

	if len(doc.TextBlocks) < 2 {
		return false
	}

	var prevBlock *boilerpipe.TextBlock
	offset := 0

	if contentOnly {
		for i := range doc.TextBlocks {
			tb := doc.TextBlocks[i]
			offset++

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
		offset = 1
	}

	for i := 0; i < len(doc.TextBlocks); i++ {
		tb := doc.TextBlocks[i]

		if tb.IsContent == false {
			prevBlock = tb
			continue
		}

		diffBlocks := tb.OffsetBlocksStart - tb.OffsetBlocksEnd - 1
		if diffBlocks <= maxBlocksDistance {
			ok := true
			if contentOnly {
				if prevBlock.IsContent == false || tb.IsContent == false {
					ok = false
				}
			}
			if ok && sameTagLevelOnly && prevBlock.TagLevel != tb.TagLevel {
				ok = false
			}
			if ok {
				prevBlock.MergeNext(tb)
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

func BoilerplateBlock() boilerpipe.Processor { return boilerplateBlock{} }

type boilerplateBlock struct{}

func (boilerplateBlock) Name() string { return "BoilerplateBlock" }

func (p boilerplateBlock) Process(doc *boilerpipe.TextDocument) bool {
	hasChanged := false

	for i := 0; i < len(doc.TextBlocks); i++ {
		tb := doc.TextBlocks[i]

		if tb.IsContent == false && tb.HasLabel(boilerpipe.LabelTitle) == false {
			doc.TextBlocks = append(doc.TextBlocks[:i], doc.TextBlocks[i+1:]...)
			i--
			hasChanged = true
		}
	}

	return hasChanged
}

func KeepLargestBlock() boilerpipe.Processor {
	return keepLargestBlock{true, ExpandToSameTagLevelMinimumWords}
}

type keepLargestBlock struct {
	expandToSameLevelText bool
	minWords              int
}

const (
	ExpandToSameTagLevel             int = 0
	ExpandToSameTagLevelMinimumWords int = 150
)

func (keepLargestBlock) Name() string { return "KeepLargestBlock" }

func (p keepLargestBlock) Process(doc *boilerpipe.TextDocument) bool {
	if len(doc.TextBlocks) < 2 {
		return false
	}

	maxNumWords := -1
	var largestBlock *boilerpipe.TextBlock
	level := -1
	j := 0
	n := -1

	for i := range doc.TextBlocks {
		tb := doc.TextBlocks[i]

		if tb.IsContent {
			nw := tb.NumWords

			if nw > maxNumWords {
				largestBlock = tb
				maxNumWords = nw

				n = j

				if p.expandToSameLevelText {
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
			tb.AddLabel(boilerpipe.LabelVeryLikelyContent)
		} else {
			tb.IsContent = false
			tb.AddLabel(boilerpipe.LabelMightBeContent)
		}
	}

	if p.expandToSameLevelText && n != -1 {

		for i := len(doc.TextBlocks) - 1; i >= 0; i-- {
			tb := doc.TextBlocks[i]

			tl := tb.TagLevel
			if tl < level {
				break
			} else if tl == level {
				if tb.NumWords >= p.minWords {
					tb.IsContent = true
				}
			}
		}

		for i := range doc.TextBlocks {
			tb := doc.TextBlocks[i]

			tl := tb.TagLevel
			if tl < level {
				break
			} else if tl == level {
				if tb.NumWords >= p.minWords {
					tb.IsContent = true
				}
			}
		}
	}

	return true
}

func KeepLargestFulltextBlock() boilerpipe.Processor { return keepLargestFulltextBlock{} }

type keepLargestFulltextBlock struct{}

func (keepLargestFulltextBlock) Name() string { return "KeepLargestFulltextBlock" }

func (p keepLargestFulltextBlock) Process(doc *boilerpipe.TextDocument) bool {
	if len(doc.TextBlocks) < 2 {
		return false
	}

	max := -1
	var largestBlock *boilerpipe.TextBlock

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
			tb.AddLabel(boilerpipe.LabelMightBeContent)
		}
	}

	return true
}

func ExpandTitleToContent() boilerpipe.Processor { return expandTitleToContent{} }

type expandTitleToContent struct{}

func (expandTitleToContent) Name() string { return "ExpandTitleToContent" }

func (p expandTitleToContent) Process(doc *boilerpipe.TextDocument) bool {
	j := 0
	title := -1
	contentStart := -1

	for i := range doc.TextBlocks {
		tb := doc.TextBlocks[i]

		if contentStart == -1 && tb.HasLabel(boilerpipe.LabelTitle) {
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

		if tb.HasLabel(boilerpipe.LabelMightBeContent) {
			log.Println("Expand:", tb.Text)
			hasChanged = (tb.IsContent == false) || hasChanged
			tb.IsContent = true
		}
	}

	return hasChanged
}

func LargeBlockSameTagLevelToContent() boilerpipe.Processor { return largeBlockSameTagLevelToContent{} }

type largeBlockSameTagLevelToContent struct{}

func (largeBlockSameTagLevelToContent) Name() string { return "LargeBlockSameTagLevelToContent" }

func (p largeBlockSameTagLevelToContent) Process(doc *boilerpipe.TextDocument) bool {
	hasChanged := false
	tagLevel := -1

	for i := range doc.TextBlocks {
		tb := doc.TextBlocks[i]

		if tb.IsContent && tb.HasLabel(boilerpipe.LabelVeryLikelyContent) {
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

func IgnoreBlocksAfterContent() boilerpipe.Processor {
	return ignoreBlocksAfterContent{DefaultMinNumberOfWords}
}

type ignoreBlocksAfterContent struct{ minNumWords int }

const DefaultMinNumberOfWords = 60

func (ignoreBlocksAfterContent) Name() string { return "IgnoreBlocksAfterContent" }

func (p ignoreBlocksAfterContent) Process(doc *boilerpipe.TextDocument) bool {
	hasChanged := false
	numWords := 0
	foundEndOfText := false

	for i := range doc.TextBlocks {
		tb := doc.TextBlocks[i]
		eot := tb.HasLabel(boilerpipe.LabelIndicatesEndOfText)

		if tb.IsContent {
			numWords += getNumFullTextWords(tb)
		}
		if eot && numWords >= p.minNumWords {
			foundEndOfText = true
		}
		if foundEndOfText {
			hasChanged = true
			tb.IsContent = false
		}
	}

	return hasChanged
}

func NumWordsRulesClassifier() boilerpipe.Processor { return numWordsRulesClassifier{} }

type numWordsRulesClassifier struct{}

func (numWordsRulesClassifier) Name() string { return "NumWordsRulesClassifier" }

func (p numWordsRulesClassifier) Process(doc *boilerpipe.TextDocument) bool {
	hasChanged := false

	if len(doc.TextBlocks) == 0 {
		return false
	}

	prevBlock := boilerpipe.TextBlockEmptyStart
	currentBlock := doc.TextBlocks[0]
	var nextBlock *boilerpipe.TextBlock

	if len(doc.TextBlocks) >= 2 {
		nextBlock = doc.TextBlocks[1]
	} else {
		nextBlock = boilerpipe.TextBlockEmptyStart
	}

	hasChanged = classify(prevBlock, currentBlock, nextBlock) || hasChanged

	if nextBlock != boilerpipe.TextBlockEmptyStart {
		for i := 3; i < len(doc.TextBlocks); i++ {
			prevBlock = currentBlock
			currentBlock = nextBlock
			nextBlock = doc.TextBlocks[i]
			hasChanged = classify(prevBlock, currentBlock, nextBlock) || hasChanged
		}
		prevBlock = currentBlock
		currentBlock = nextBlock
		nextBlock = boilerpipe.TextBlockEmptyEnd
		hasChanged = classify(prevBlock, currentBlock, nextBlock) || hasChanged
	}

	return hasChanged
}

func classify(prev, curr, next *boilerpipe.TextBlock) bool {
	isContent := false

	if curr.LinkDensity <= 0.333333 {
		if prev.LinkDensity <= 0.555556 {
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

func getNumFullTextWords(tb *boilerpipe.TextBlock) int {
	minTextDensity := 9.0

	if tb.TextDensity >= minTextDensity {
		return tb.NumWords
	} else {
		return 0
	}
}
