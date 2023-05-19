package boilerpipe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Version is the version of the boilerpipe package.
const Version = "0.5.0"

type Document struct {
	// Title is the title of the document.
	Title string

	Author string

	// Date is the date the document was created.
	Date time.Time

	TextBlocks []*TextBlock

	linkedDataArticle linkedDataArticle
}

// ParseDocument parses an HTML document and returns a Document for further
// processing through filters.
func ParseDocument(r io.Reader) (*Document, error) {
	var h *contentHandler
	h, err := parse(r, func(tok *html.Token, h *contentHandler) {
		h.TextToken(tok)
	})
	if err != nil {
		return nil, err
	}

	h.FlushBlock()

	doc := &Document{}

	// Parse linked-data JSON
	for _, s := range h.linkedDataJSON {
		if err := json.Unmarshal([]byte(s), &doc.linkedDataArticle); err != nil {
			continue // try the next if multiple
		}
		if doc.linkedDataArticle.Type == "Article" {
			break
		}
	}

	if doc.linkedDataArticle.Headline != "" {
		doc.Title = doc.linkedDataArticle.Headline
	} else {
		doc.Title = h.title
	}

	doc.Author = doc.linkedDataArticle.Author.Name

	if !doc.linkedDataArticle.DatePublished.IsZero() {
		doc.Date = doc.linkedDataArticle.DatePublished
	} else {
		doc.Date = h.time
	}

	doc.TextBlocks = h.textBlocks

	return doc, nil
}

func (doc *Document) Content() string {
	if doc.linkedDataArticle.Body != "" {
		return doc.linkedDataArticle.Body
	}
	return doc.Text(true, false)
}

// HasTitle returns true if the document date is not the zero time.Time value.
func (doc *Document) HasTitle() bool {
	return !doc.Date.IsZero()
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

func parse(r io.Reader, fn func(tok *html.Token, h *contentHandler)) (h *contentHandler, err error) {
	h = newContentHandler()

	z := html.NewTokenizer(r)
	for {
		tt := z.Next()
		tok := z.Token()

		switch tt {
		case html.ErrorToken:
			if z.Err() != io.EOF {
				err = z.Err()
			}
			goto DONE

		case html.TextToken:
			if h.inLinkedDataJSON {
				h.linkedDataJSON = append(h.linkedDataJSON, tok.Data)
			}
			fn(&tok, h)

		case html.StartTagToken:
			// If the token is start tag, but should be a self-closing tag,
			// then the token is malformed and should be skipped.
			if shouldBeSelfClosingTag(tok.DataAtom) {
				continue
			}

			if tok.DataAtom == atom.Script {
				for _, attr := range tok.Attr {
					if attr.Key == "type" && attr.Val == "application/ld+json" {
						h.inLinkedDataJSON = true
					}
				}
			}
			h.StartElement(&tok)

		case html.EndTagToken:
			if h.inLinkedDataJSON {
				h.inLinkedDataJSON = false
			}
			h.EndElement(&tok)

		case html.SelfClosingTagToken, html.CommentToken, html.DoctypeToken:
			// do nothing
		}
	}

DONE:
	return
}

type linkedDataArticle struct {
	Type          string           `json:"@type"`
	Headline      string           `json:"headline"`
	DatePublished time.Time        `json:"datePublished"`
	Author        linkedDataAuthor `json:"author"`
	Body          string           `json:"articleBody"`
}

type linkedDataAuthor struct {
	Type string `json:"@type"`
	Name string `json:"name"`
}
