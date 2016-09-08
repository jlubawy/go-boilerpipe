package boilerpipe

import (
	"bytes"
	"io"
	"log"

	"golang.org/x/net/html"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type TextBlock struct {
	Text string

	numWords               int
	numLinkedWords         int
	numWordsInWrappedLines int
	numWrappedLines        int
	offsetBlocks           int
	tagLevel               int

	textDensity float64
	linkDensity float64

	isContent bool
}

func NewTextBlock(text string, numWords int, numLinkedWords int, numWordsInWrappedLines int, numWrappedLines int, offsetBlocks int, tagLevel int) *TextBlock {
	tb := &TextBlock{
		Text: text,
		// TODO: currentContainedTextElements,
		numWords:               numWords,
		numLinkedWords:         numLinkedWords,
		numWordsInWrappedLines: numWordsInWrappedLines,
		numWrappedLines:        numWrappedLines,
		offsetBlocks:           offsetBlocks,
		tagLevel:               tagLevel,
	}

	if tb.numWordsInWrappedLines == 0 {
		tb.numWordsInWrappedLines = numWords
		tb.numWrappedLines = 1
	}

	tb.textDensity = float64(numWordsInWrappedLines) / float64(numWrappedLines)
	if numWords == 0 {
		tb.linkDensity = 0.0
	} else {
		tb.linkDensity = float64(numLinkedWords) / float64(numWords)
	}

	return tb
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
		log.Printf("TokenType: %-14s : %s\n", tt, h)

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
		if tb.isContent {
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
