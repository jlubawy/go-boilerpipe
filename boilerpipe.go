package boilerpipe

import (
	"fmt"
	"io"

	"golang.org/x/net/html"
)

type TextDocument struct {
	Title      string
	TextBlocks []*TextBlock
}

type TextBlock struct {
}

func NewTextDocument(r io.Reader) (doc *TextDocument, err error) {
	z := html.NewTokenizer(r)

	h := NewContentHandler()

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				return
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

	doc = &TextDocument{
		Title: h.title,
	}

	fmt.Print(h.textBuffer.String())

	return
}
