package boilerpipe

import (
	"bytes"
	"io"
	_ "log"

	"golang.org/x/net/html"
)

func init() {
	//log.SetFlags(log.Lshortfile)
}

type TextBlock struct {
	Text string

	NumWords               int
	NumLinkedWords         int
	NumWordsInWrappedLines int
	NumWrappedLines        int
	OffsetBlocks           int
	TagLevel               int

	TextDensity float64
	LinkDensity float64

	IsContent bool

	labels map[int]bool
}

func NewTextBlock(text string, numWords int, numLinkedWords int, numWordsInWrappedLines int, numWrappedLines int, offsetBlocks int, tagLevel int) *TextBlock {
	tb := &TextBlock{
		Text: text,
		// TODO: currentContainedTextElements,
		NumWords:               numWords,
		NumLinkedWords:         numLinkedWords,
		NumWordsInWrappedLines: numWordsInWrappedLines,
		NumWrappedLines:        numWrappedLines,
		OffsetBlocks:           offsetBlocks,
		TagLevel:               tagLevel,

		labels: make(map[int]bool),
	}

	if numWordsInWrappedLines == 0 {
		tb.NumWordsInWrappedLines = numWords
		tb.NumWrappedLines = 1
	}

	tb.TextDensity = float64(numWordsInWrappedLines) / float64(numWrappedLines)
	if numWords == 0 {
		tb.LinkDensity = 0.0
	} else {
		tb.LinkDensity = float64(numLinkedWords) / float64(numWords)
	}

	return tb
}

const (
	LabelIndicatesEndOfText int = iota
)

func (tb *TextBlock) AddLabel(label int) *TextBlock {
	tb.labels[label] = true
	return tb
}

func (tb *TextBlock) HasLabel(label int) bool {
	_, hasLabel := tb.labels[label]
	return hasLabel
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

		if _, err := buf.WriteString("!!!"); err != nil {
			panic(err)
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
	Process(*TextDocument) bool
}
