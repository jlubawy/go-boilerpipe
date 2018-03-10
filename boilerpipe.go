package boilerpipe

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Version is the version of the boilerpipe package.
const Version = "0.2.4"

type Document struct {
	// Title is the title of the document.
	Title string

	// Date is the date the document was created.
	Date time.Time

	TextBlocks []*TextBlock
}

// ParseDocument parses an HTML document and returns a Document for further
// processing through filters.
func ParseDocument(r io.Reader) (*Document, error) {
	var h *contentHandler
	h, err := parse(r, func(z *html.Tokenizer, h *contentHandler) {
		h.TextToken(z)
	})
	if err != nil {
		return nil, err
	}

	h.FlushBlock()

	doc := &Document{}

	// Set the rest of the document fields
	doc.Title = h.title
	if doc.Date.Equal(time.Time{}) {
		doc.Date = h.time
	}
	doc.TextBlocks = h.textBlocks

	return doc, nil
}

func (doc *Document) Content() string {
	return doc.Text(true, false)
}

func (doc *Document) Text(includeContent, includeNonContent bool) string {
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

		fmt.Fprintln(buf, tb.Text)
	}

	return html.EscapeString(strings.Trim(buf.String(), " \n"))
}

func parse(r io.Reader, fn func(z *html.Tokenizer, h *contentHandler)) (h *contentHandler, err error) {
	h = newContentHandler()

	z := html.NewTokenizer(r)
	for {
		tt := z.Next()

		switch tt {
		case html.ErrorToken:
			if z.Err() != io.EOF {
				err = z.Err()
			}
			goto DONE

		case html.TextToken:
			fn(z, h)

		case html.StartTagToken:
			h.StartElement(z)

		case html.EndTagToken:
			h.EndElement(z)

		case html.SelfClosingTagToken, html.CommentToken, html.DoctypeToken:
			// do nothing
		}
	}

DONE:
	return
}
