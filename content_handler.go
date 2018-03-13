package boilerpipe

import (
	"bytes"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type atomStack struct {
	s []atom.Atom
}

func newAtomStack() *atomStack {
	return &atomStack{
		s: make([]atom.Atom, 0),
	}
}

func (stack *atomStack) Push(a atom.Atom) *atomStack {
	stack.s = append(stack.s, a)
	return stack
}

func (stack *atomStack) Pop() atom.Atom {
	if len(stack.s) == 0 {
		return atom.Atom(0)
	}
	el := stack.s[len(stack.s)-1]
	stack.s = stack.s[:len(stack.s)-1]
	return el
}

const (
	anchorTextStart = "$\ue00a<"
	anchorTextEnd   = ">\ue00a$"
)

type contentHandler struct {
	title string
	time  time.Time

	tokenBuffer *bytes.Buffer
	textBuffer  *bytes.Buffer

	depthBody       int
	depthAnchor     int
	depthIgnoreable int

	depthTag      int
	depthBlockTag int

	lastWasWhitespace bool
	textElementIndex  int

	textBlocks []*TextBlock

	lastStartTag string
	lastEndTag   string

	offsetBlocks int
	//private BitSet currentContainedTextElements = new BitSet();

	flush        bool
	inAnchorText bool

	labelStack *LabelStack
	// TODO: LinkedList<Integer> fontSizeStack = new LinkedList<Integer>();

	atomStack *atomStack

	inLinkedDataJSON bool
	linkedDataJSON   []string
}

func newContentHandler() *contentHandler {
	return &contentHandler{
		tokenBuffer: &bytes.Buffer{},
		textBuffer:  &bytes.Buffer{},

		depthBlockTag: -1,

		textBlocks: make([]*TextBlock, 0),

		labelStack: NewLabelStack(),

		atomStack: newAtomStack(),

		linkedDataJSON: make([]string, 0),
	}
}

func (h *contentHandler) StartElement(tok *html.Token) {
	h.atomStack.Push(tok.DataAtom)

	ta, ok := tagActionMap[tok.DataAtom]
	if ok {
		switch ta.(type) {
		case *tagActionTime:
			for _, attr := range tok.Attr {
				if attr.Key == "datetime" {
					t, err := time.Parse(time.RFC3339, attr.Val)
					if err == nil {
						h.time = t
					}
					break
				}
			}
		}

		if ta.ChangesTagLevel() {
			h.depthTag++
		}
		h.flush = ta.Start(h) || h.flush
	} else {
		h.depthTag++
		h.flush = true
	}

	h.lastStartTag = tok.Data

}

func (h *contentHandler) EndElement(tok *html.Token) {
	pa := h.atomStack.Pop()
	if pa != tok.DataAtom {
		return // malformed HTML, missing closing tag
	}

	ta, ok := tagActionMap[tok.DataAtom]
	if ok {
		h.flush = ta.End(h) || h.flush
	} else {
		h.flush = true
	}

	if !ok || ta.ChangesTagLevel() {
		h.depthTag--
	}

	if h.flush {
		h.FlushBlock()
	}

	h.lastEndTag = tok.Data

	h.labelStack.Pop()
}

type spaceRemover struct {
	wasFirstWhitespace bool
	wasLastWhitespace  bool
}

func (sr *spaceRemover) getSpaceRemovalFunc() func(rune) rune {
	i := 0
	return func(r rune) rune {
		if unicode.IsSpace(r) {
			if i == 0 {
				sr.wasFirstWhitespace = true
			}
			i++
			if sr.wasLastWhitespace {
				return -1
			} else {
				sr.wasLastWhitespace = true
				return ' '
			}
		} else {
			i++
			sr.wasLastWhitespace = false
		}
		return r
	}
}

func (h *contentHandler) TextToken(tok *html.Token) {
	h.textElementIndex++

	if h.flush {
		h.FlushBlock()
		h.flush = false
	}

	if h.depthIgnoreable != 0 {
		return
	}

	if len(tok.Data) == 0 {
		return
	}

	sr := &spaceRemover{}
	ch := strings.TrimSpace(strings.Map(sr.getSpaceRemovalFunc(), tok.Data))
	if len(ch) == 0 {
		if sr.wasFirstWhitespace || sr.wasLastWhitespace {
			if h.lastWasWhitespace == false {
				h.textBuffer.WriteByte(' ')
				h.tokenBuffer.WriteByte(' ')
			}
			h.lastWasWhitespace = true
		} else {
			h.lastWasWhitespace = false
		}

		return
	}

	if sr.wasFirstWhitespace {
		if h.lastWasWhitespace == false {
			h.textBuffer.WriteByte(' ')
			h.tokenBuffer.WriteByte(' ')
		}
	}

	if h.depthBlockTag == -1 {
		h.depthBlockTag = h.depthTag
	}

	h.textBuffer.WriteString(ch)
	h.tokenBuffer.WriteString(ch)
	if sr.wasLastWhitespace {
		h.textBuffer.WriteByte(' ')
		h.tokenBuffer.WriteByte(' ')
	}

	h.lastWasWhitespace = sr.wasLastWhitespace

	// TODO: currentContainedTextElements.set(h.textElementIndex);
}

var reMultiSpace = regexp.MustCompile(`[\s]+`)

func tokenize(b *bytes.Buffer) []string {
	return reMultiSpace.Split(strings.TrimSpace(b.String()), -1)
}

var reValidWordCharacter = regexp.MustCompile(`[\w]`)

func isWord(tok string) bool {
	return reValidWordCharacter.MatchString(tok)
}

func (h *contentHandler) FlushBlock() {
	if h.depthBody == 0 {
		if h.lastStartTag == atom.Title.String() {
			title := strings.TrimSpace(h.tokenBuffer.String())
			if len(title) > 0 {
				h.title = title
			}
		}

		h.textBuffer.Reset()
		h.tokenBuffer.Reset()
		return
	}

	switch h.tokenBuffer.Len() {
	case 0:
		return
	case 1:
		if h.lastWasWhitespace {
			h.textBuffer.Reset()
			h.tokenBuffer.Reset()
			return
		}
	}

	tokens := tokenize(h.tokenBuffer)

	const maxLineLength = 80

	var (
		numWords            int
		numLinkedWords      int
		numWrappedLines     int
		numTokens           int
		numWordsCurrentLine int
	)
	currentLineLength := -1 // don't count the first space

	for _, tok := range tokens {
		if tok == anchorTextStart {
			h.inAnchorText = true
		} else if tok == anchorTextEnd {
			h.inAnchorText = false
		} else if isWord(tok) {
			numTokens++
			numWords++
			numWordsCurrentLine++

			if h.inAnchorText {
				numLinkedWords++
			}

			tokLength := len(tok)
			currentLineLength += tokLength + 1

			if currentLineLength > maxLineLength {
				numWrappedLines++
				currentLineLength = tokLength
				numWordsCurrentLine = 1
			}
		} else {
			numTokens++
		}
	}

	if numTokens == 0 {
		return
	}

	numWordsInWrappedLines := 0
	_ = numWordsInWrappedLines

	if numWrappedLines == 0 {
		numWordsInWrappedLines = numWords
		numWrappedLines = 1
	} else {
		numWordsInWrappedLines = numWords - numWordsCurrentLine
	}

	text := strings.TrimSpace(h.textBuffer.String())

	if len(text) > 0 {
		tb := NewTextBlock()

		tb.Text = text
		tb.NumWords = numWords
		tb.NumLinkedWords = numLinkedWords
		tb.NumWordsInWrappedLines = numWordsInWrappedLines
		tb.NumWrappedLines = numWrappedLines
		tb.OffsetBlocksStart = h.offsetBlocks
		tb.OffsetBlocksEnd = h.offsetBlocks
		tb.TagLevel = h.depthBlockTag

		if numWordsInWrappedLines == 0 {
			tb.NumWordsInWrappedLines = numWords
			tb.NumWrappedLines = 1
		}

		// TODO:
		//for (Integer l : fontSizeStack) {
		//  if (l != null) {
		//    tb.AddLabels("font-" + l);
		//    break;
		//  }
		//}

		tb.AddLabels(h.labelStack.PopAll()...)

		h.textBlocks = append(h.textBlocks, tb)

		// TODO: currentContainedTextElements = new BitSet();
		h.offsetBlocks++
	}

	h.textBuffer.Reset()
	h.tokenBuffer.Reset()

	h.depthBlockTag = -1
}

func (h *contentHandler) addWhitespaceIfNecessary() {
	if h.lastWasWhitespace == false {
		h.tokenBuffer.WriteByte(' ')
		h.textBuffer.WriteByte(' ')
		h.lastWasWhitespace = true
	}
}

type tagAction interface {
	Start(*contentHandler) bool
	End(*contentHandler) bool
	ChangesTagLevel() bool
}

type tagActionIgnorable struct{}

func (ta *tagActionIgnorable) Start(h *contentHandler) bool {
	h.depthIgnoreable++
	return true
}

func (*tagActionIgnorable) End(h *contentHandler) bool {
	h.depthIgnoreable--
	return true
}

func (*tagActionIgnorable) ChangesTagLevel() bool { return true }

type tagActionAnchor struct{}

func (ta *tagActionAnchor) Start(h *contentHandler) bool {
	h.depthAnchor++

	if h.depthIgnoreable == 0 {
		h.addWhitespaceIfNecessary()
		h.tokenBuffer.WriteString(anchorTextStart)
		h.tokenBuffer.WriteByte(' ')
		h.lastWasWhitespace = true
	}

	return false
}

func (*tagActionAnchor) End(h *contentHandler) bool {
	h.depthAnchor--

	if h.depthAnchor == 0 {
		if h.depthIgnoreable == 0 {
			h.addWhitespaceIfNecessary()
			h.tokenBuffer.WriteString(anchorTextEnd)
			h.tokenBuffer.WriteByte(' ')
			h.lastWasWhitespace = true
		}
	}

	return false
}

func (*tagActionAnchor) ChangesTagLevel() bool { return true }

type tagActionBody struct{}

func (ta *tagActionBody) Start(h *contentHandler) bool {
	h.FlushBlock()
	h.depthBody++
	return false
}
func (*tagActionBody) End(h *contentHandler) bool {
	h.FlushBlock()
	h.depthBody--
	return false
}

func (*tagActionBody) ChangesTagLevel() bool { return true }

type tagActionInlineWhitespace struct{}

func (ta *tagActionInlineWhitespace) Start(h *contentHandler) bool {
	h.addWhitespaceIfNecessary()
	return false
}

func (*tagActionInlineWhitespace) End(h *contentHandler) bool {
	h.addWhitespaceIfNecessary()
	return false
}

func (*tagActionInlineWhitespace) ChangesTagLevel() bool { return false }

type tagActionInlineNoWhitespace struct{}

func (*tagActionInlineNoWhitespace) Start(h *contentHandler) bool { return false }
func (*tagActionInlineNoWhitespace) End(h *contentHandler) bool   { return false }
func (*tagActionInlineNoWhitespace) ChangesTagLevel() bool        { return false }

type tagActionBlockTagLabel struct{ labels []Label }

func (ta *tagActionBlockTagLabel) Start(h *contentHandler) bool {
	h.labelStack.Push(ta.labels...)
	return true
}
func (*tagActionBlockTagLabel) End(h *contentHandler) bool { return true }
func (*tagActionBlockTagLabel) ChangesTagLevel() bool      { return true }

type tagActionIgnoreableVoid struct{}

func (*tagActionIgnoreableVoid) Start(h *contentHandler) bool { return false }
func (*tagActionIgnoreableVoid) End(h *contentHandler) bool   { return false }
func (*tagActionIgnoreableVoid) ChangesTagLevel() bool        { return false }

type tagActionTime struct{}

func (*tagActionTime) Start(h *contentHandler) bool { return true }
func (*tagActionTime) End(h *contentHandler) bool   { return true }
func (*tagActionTime) ChangesTagLevel() bool        { return true }

// From DefaulttagActionMap.java
var tagActionMap = map[atom.Atom]tagAction{
	atom.Applet:     &tagActionIgnorable{},
	atom.Figcaption: &tagActionIgnorable{},
	atom.Figure:     &tagActionIgnorable{},
	atom.Noscript:   &tagActionIgnorable{},
	atom.Object:     &tagActionIgnorable{},
	atom.Option:     &tagActionIgnorable{},
	atom.Script:     &tagActionIgnorable{},
	atom.Style:      &tagActionIgnorable{},

	atom.A: &tagActionAnchor{},

	atom.Body: &tagActionBody{},

	atom.Abbr: &tagActionInlineWhitespace{},
	// no atom.Acronym

	atom.B:      &tagActionInlineNoWhitespace{},
	atom.Code:   &tagActionInlineNoWhitespace{},
	atom.Em:     &tagActionInlineNoWhitespace{},
	atom.Font:   &tagActionInlineNoWhitespace{}, // can also use TA_FONT
	atom.I:      &tagActionInlineNoWhitespace{},
	atom.Span:   &tagActionInlineNoWhitespace{},
	atom.Strike: &tagActionInlineNoWhitespace{},
	atom.Strong: &tagActionInlineNoWhitespace{},
	atom.Sub:    &tagActionInlineNoWhitespace{},
	atom.Sup:    &tagActionInlineNoWhitespace{},
	atom.Tt:     &tagActionInlineNoWhitespace{},
	atom.U:      &tagActionInlineNoWhitespace{},
	atom.Var:    &tagActionInlineNoWhitespace{},

	atom.Li: &tagActionBlockTagLabel{[]Label{LabelList}},
	atom.H1: &tagActionBlockTagLabel{[]Label{LabelHeading, LabelHeading1}},
	atom.H2: &tagActionBlockTagLabel{[]Label{LabelHeading, LabelHeading2}},
	atom.H3: &tagActionBlockTagLabel{[]Label{LabelHeading, LabelHeading3}},

	atom.Area:     &tagActionIgnoreableVoid{},
	atom.Base:     &tagActionIgnoreableVoid{},
	atom.Br:       &tagActionIgnoreableVoid{},
	atom.Col:      &tagActionIgnoreableVoid{},
	atom.Embed:    &tagActionIgnoreableVoid{},
	atom.Hr:       &tagActionIgnoreableVoid{},
	atom.Img:      &tagActionIgnoreableVoid{},
	atom.Input:    &tagActionIgnoreableVoid{},
	atom.Link:     &tagActionIgnoreableVoid{},
	atom.Menuitem: &tagActionIgnoreableVoid{},
	atom.Meta:     &tagActionIgnoreableVoid{},
	atom.Param:    &tagActionIgnoreableVoid{},
	atom.Source:   &tagActionIgnoreableVoid{},
	atom.Track:    &tagActionIgnoreableVoid{},
	atom.Wbr:      &tagActionIgnoreableVoid{},

	atom.Time: &tagActionTime{},
}
