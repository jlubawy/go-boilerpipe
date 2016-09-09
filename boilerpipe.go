package boilerpipe

import (
	"bytes"
	"io"
	"math"

	"golang.org/x/net/html"
)

func init() {
	//log.SetFlags(log.Lshortfile)
}

type TextBlock struct {
	Text string

	OffsetBlocksStart int
	OffsetBlocksEnd   int

	NumWords               int
	NumLinkedWords         int
	NumWordsInWrappedLines int
	NumWrappedLines        int
	TagLevel               int

	TextDensity float64
	LinkDensity float64

	IsContent bool

	labels map[int]bool
}

var TextBlockEmptyStart = NewTextBlock("", 0, 0, 0, 0, math.MinInt32, 0)
var TextBlockEmptyEnd = NewTextBlock("", 0, 0, 0, 0, math.MaxInt32, 0)

func NewTextBlock(text string, numWords int, numLinkedWords int, numWordsInWrappedLines int, numWrappedLines int, offsetBlocks int, tagLevel int) *TextBlock {
	tb := &TextBlock{
		Text: text,
		// TODO: currentContainedTextElements,
		NumWords:               numWords,
		NumLinkedWords:         numLinkedWords,
		NumWordsInWrappedLines: numWordsInWrappedLines,
		NumWrappedLines:        numWrappedLines,
		OffsetBlocksStart:      offsetBlocks,
		OffsetBlocksEnd:        offsetBlocks,
		TagLevel:               tagLevel,

		labels: make(map[int]bool),
	}

	if numWordsInWrappedLines == 0 {
		tb.NumWordsInWrappedLines = numWords
		tb.NumWrappedLines = 1
	}

	initDensities(tb)

	return tb
}

func initDensities(tb *TextBlock) {
	tb.TextDensity = float64(tb.NumWordsInWrappedLines) / float64(tb.NumWrappedLines)
	if tb.NumWords == 0 {
		tb.LinkDensity = 0.0
	} else {
		tb.LinkDensity = float64(tb.NumLinkedWords) / float64(tb.NumWords)
	}
}

const (
	LabelIndicatesEndOfText int = iota
	LabelHeading
	LabelMightBeContent
	LabelVeryLikelyContent
	LabelTitle
)

func (tb *TextBlock) AddLabel(label int) *TextBlock {
	tb.labels[label] = true
	return tb
}

func (tb *TextBlock) HasLabel(label int) bool {
	_, hasLabel := tb.labels[label]
	return hasLabel
}

func (tb *TextBlock) MergeNext(next *TextBlock) {
	buf := bytes.NewBufferString(tb.Text)
	buf.WriteRune('\n')
	buf.WriteString(next.Text)
	tb.Text = buf.String()

	tb.NumWords += next.NumWords
	tb.NumLinkedWords += next.NumLinkedWords

	tb.NumWordsInWrappedLines += next.NumWordsInWrappedLines
	tb.NumWrappedLines += next.NumWrappedLines

	tb.OffsetBlocksStart = int(math.Min(float64(tb.OffsetBlocksStart), float64(next.OffsetBlocksStart)))
	tb.OffsetBlocksEnd = int(math.Min(float64(tb.OffsetBlocksEnd), float64(next.OffsetBlocksEnd)))

	initDensities(tb)

	tb.IsContent = tb.IsContent || next.IsContent

	// TODO
	//if (containedTextElements == null) {
	//  containedTextElements = (BitSet) next.containedTextElements.clone();
	//} else {
	//  containedTextElements.or(next.containedTextElements);
	//}

	for k, v := range next.labels {
		tb.labels[k] = v
	}

	tb.TagLevel = int(math.Min(float64(tb.TagLevel), float64(next.TagLevel)))
}

type TextDocument struct {
	Title      string
	TextBlocks []*TextBlock
}

func NewTextDocument(r io.Reader) (doc *TextDocument, err error) {
	z := html.NewTokenizer(r)

	h := NewContentHandler()

	for {
		tt := z.Next()
		//log.Printf("TokenType: %-14s : %s\n", tt, h)

		switch tt {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				goto DONE
			} else {
				err = z.Err()
				return
			}

		case html.TextToken:
			h.TextToken(z)

		case html.StartTagToken:
			h.StartElement(z)

		case html.EndTagToken:
			h.EndElement(z)

		case html.SelfClosingTagToken, html.CommentToken, html.DoctypeToken:
			// do nothing
		}
	}

DONE:
	h.FlushBlock()

	doc = &TextDocument{
		Title:      h.title,
		TextBlocks: h.textBlocks,
	}

	return
}

func (doc *TextDocument) Content() string {
	return doc.Text(true, false)
}

func (doc *TextDocument) Text(includeContent, includeNonContent bool) string {
	buf := &bytes.Buffer{}

	for _, tb := range doc.TextBlocks {
		if tb.IsContent {
			if includeContent == false {
				continue
			}
		} else {
			if includeNonContent == false {
				continue
			}
		}

		if _, err := buf.WriteString(tb.Text); err != nil {
			panic(err)
		}
		if _, err := buf.WriteRune('\n'); err != nil {
			panic(err)
		}
	}

	return buf.String()
}

type Processor interface {
	Name() string
	Process(*TextDocument) bool
}
