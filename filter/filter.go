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

type ignoreBlocksAfterContentFilter struct{ minNumWords int }

const DefaultMinNumberOfWords = 60

func IgnoreBlocksAfterContentFilter(doc *boilerpipe.TextDocument) bool {
	p := ignoreBlocksAfterContentFilter{DefaultMinNumberOfWords}
	return p.Process(doc)
}

func (p ignoreBlocksAfterContentFilter) Process(doc *boilerpipe.TextDocument) bool {
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

func getNumFullTextWords(tb *boilerpipe.TextBlock) int {
	minTextDensity := 9.0

	if tb.TextDensity >= minTextDensity {
		return tb.NumWords
	} else {
		return 0
	}
}
