package filter

import (
	"strings"
	_ "unicode"

	"github.com/jlubawy/go-boilerpipe"
)

type terminatingBlocks struct{}

func TerminatingBlocks(doc *boilerpipe.TextDocument) bool {
	p := terminatingBlocks{}
	return p.Process(doc)
}

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

type trailingHeadlineToBoilerplate struct{}

func TrailingHeadlineToBoilerplate(doc *boilerpipe.TextDocument) bool {
	p := trailingHeadlineToBoilerplate{}
	return p.Process(doc)
}

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

type blockProximityFusionParams struct {
	maxBlocksDistance int
	contentOnly       bool
	sameTagLevelOnly  bool
}

var (
	BlockProximityFusionMaxDistanceOne                        = &blockProximityFusionParams{1, false, false}
	BlockProximityFusionMaxDistanceOneSameTagLevel            = &blockProximityFusionParams{1, false, true}
	BlockProximityFusionMaxDistanceOneContentOnly             = &blockProximityFusionParams{1, true, false}
	BlockProximityFusionMaxDistanceOneContentOnlySameTagLevel = &blockProximityFusionParams{1, true, true}
)

func (p *blockProximityFusionParams) BlockProximityFusion(doc *boilerpipe.TextDocument) bool {
	return p.Process(doc)
}

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
				i++
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

type keepLargestBlock struct {
	expandToSameLevelText bool
	minWords              int
}

const (
	ExpandToSameTagLevel             int = 0
	ExpandToSameTagLevelMinimumWords int = 150
)

func KeepLargestBlock(doc *boilerpipe.TextDocument) bool {
	p := keepLargestBlock{true, ExpandToSameTagLevelMinimumWords}
	return p.Process(doc)
}

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

type keepLargestFulltextBlock struct{}

func KeepLargestFulltextBlock(doc *boilerpipe.TextDocument) bool {
	p := keepLargestFulltextBlock{}
	return p.Process(doc)
}

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

type ignoreBlocksAfterContent struct{ minNumWords int }

const DefaultMinNumberOfWords = 60

func IgnoreBlocksAfterContent(doc *boilerpipe.TextDocument) bool {
	p := ignoreBlocksAfterContent{DefaultMinNumberOfWords}
	return p.Process(doc)
}

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

type numWordsRulesClassifier struct{}

func NumWordsRulesClassifier(doc *boilerpipe.TextDocument) bool {
	p := numWordsRulesClassifier{}
	return p.Process(doc)
}

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
