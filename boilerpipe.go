package boilerpipe

import (
	"bytes"
	"io"

	"golang.org/x/net/html"
)

type TextDocument struct {
	Title      string
	TextBlocks []*TextBlock
}

type TextBlock struct {
	Text string

	numWords               int
	numLinkedWords         int
	numWordsInWrappedLines int
	numWrappedLines        int
	offsetBlocks           int
	tagLevel               int

	isContent bool
}

func NewTextDocument(r io.Reader) (doc *TextDocument, err error) {
	z := html.NewTokenizer(r)

	h := NewContentHandler()

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				goto DONE
			} else {
				err = z.Err()
				return
			}

		case html.TextToken:
			h.textToken(z)

		case html.StartTagToken:
			h.startElement(z)

		case html.EndTagToken:
			h.endElement(z)

		case html.SelfClosingTagToken, html.CommentToken, html.DoctypeToken:
			// do nothing
		}
	}

DONE:
	doc = &TextDocument{
		Title: h.title,
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
