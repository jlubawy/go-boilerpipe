package boilerpipe

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	url "github.com/jlubawy/go-boilerpipe/normurl"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type BoilerpipeVersion struct {
	Major uint
	Minor uint
	Build uint
}

func (v BoilerpipeVersion) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Build)
}

var Version = BoilerpipeVersion{
	Major: 0,
	Minor: 0,
	Build: 7,
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

	Labels map[Label]bool
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

		Labels: make(map[Label]bool),
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

type Label string

const (
	LabelIndicatesEndOfText Label = "IndicatesEndOfText"
	LabelMightBeContent           = "MightBeContent"
	LabelVeryLikelyContent        = "VeryLikelyContent"
	LabelTitle                    = "Title"
	LabelList                     = "List"
	LabelHeading                  = "Heading"
	LabelHeading1                 = "Heading1"
	LabelHeading2                 = "Heading2"
	LabelHeading3                 = "Heading3"
)

func (tb *TextBlock) AddLabel(label Label) *TextBlock {
	tb.Labels[label] = true
	return tb
}

func (tb *TextBlock) AddLabels(labels ...Label) *TextBlock {
	for _, label := range labels {
		tb.AddLabel(label)
	}
	return tb
}

func (tb *TextBlock) HasLabel(label Label) bool {
	_, hasLabel := tb.Labels[label]
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

	for k, v := range next.Labels {
		tb.Labels[k] = v
	}

	tb.TagLevel = int(math.Min(float64(tb.TagLevel), float64(next.TagLevel)))
}

type TextDocument struct {
	Title      string
	Time       time.Time
	TextBlocks []*TextBlock

	errs []error
}

func NewTextDocument(r io.Reader) (doc *TextDocument, err error) {
	z := html.NewTokenizer(r)

	h := NewContentHandler()

	doc = &TextDocument{}

	if v, ok := r.(*URLReader); ok {
		if d, exists := v.URL().Date(); exists {
			doc.Time = d
		}
	}

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

	// Set the rest of the document fields
	doc.Title = h.title
	if doc.Time.Equal(time.Time{}) {
		doc.Time = h.time
	}
	doc.TextBlocks = h.textBlocks

	// Save any errors we might have encountered
	doc.errs = h.Errors()

	return
}

func (doc *TextDocument) Errors() []error {
	return doc.errs
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

		fmt.Fprintln(buf, tb.Text)
	}

	return html.EscapeString(strings.Trim(buf.String(), " \n"))
}

type Processor interface {
	Name() string
	Process(*TextDocument) bool
}

var reMultiSpace = regexp.MustCompile(`[\s]+`)

func ExtractText(r io.Reader) (string, error) {
	z := html.NewTokenizer(r)
	buf := &bytes.Buffer{}

	h := NewContentHandler()

	for {
		tt := z.Next()

		switch tt {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				goto DONE
			} else {
				return "", z.Err()
			}

		case html.TextToken:
			if h.depthIgnoreable == 0 {
				var skipWhitespace bool

				if h.lastEndTag != "" {
					a := atom.Lookup([]byte(h.lastEndTag))
					ta, ok := TagActionMap[a]
					if ok {
						switch ta.(type) {
						case TagActionAnchor, TagActionInlineNoWhitespace:
							skipWhitespace = true
						}
					}
				}

				if !skipWhitespace {
					buf.WriteRune(' ')
				}

				buf.WriteString(string(z.Text()))
			}

		case html.StartTagToken:
			h.StartElement(z)

		case html.EndTagToken:
			h.EndElement(z)

		case html.SelfClosingTagToken, html.CommentToken, html.DoctypeToken:
			// do nothing
		}
	}

DONE:
	return strings.TrimSpace(reMultiSpace.ReplaceAllString(buf.String(), " ")), nil
}

type AtomStack struct {
	a []atom.Atom
}

func NewAtomStack() *AtomStack {
	return &AtomStack{
		a: make([]atom.Atom, 0),
	}
}

func (as *AtomStack) Push(a atom.Atom) *AtomStack {
	as.a = append(as.a, a)
	return as
}

func (as *AtomStack) Pop() atom.Atom {
	if len(as.a) == 0 {
		return atom.Atom(0)
	}
	a := as.a[len(as.a)-1]
	as.a = as.a[:len(as.a)-1]
	return a
}

type URLReader struct {
	client *http.Client
	r      io.Reader
	u      *url.URL
}

func NewURLReader(client *http.Client, u *url.URL) *URLReader {
	return &URLReader{
		client: client,
		u:      u,
	}
}

func (r *URLReader) URL() *url.URL {
	return r.u
}

func (r *URLReader) Read(p []byte) (n int, err error) {
	if r.r != nil {
		return r.r.Read(p)
	}

	resp, err := r.client.Get(r.u.String())
	if err != nil {
		return 0, err
	}
	r.r = resp.Body

	if resp.StatusCode >= 400 {
		return 0, &URLReaderErr{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}

	return r.r.Read(p)
}

func (r *URLReader) Close() error {
	if r.r == nil {
		return nil
	}
	return r.Close()
}

type URLReaderErr struct {
	StatusCode int
	Status     string
}

func (err *URLReaderErr) Error() string {
	return err.Status
}
