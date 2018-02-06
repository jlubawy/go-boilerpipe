package boilerpipe

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type AtomStack struct {
	s []atom.Atom
}

func NewAtomStack() *AtomStack {
	return &AtomStack{
		s: make([]atom.Atom, 0),
	}
}

func (stack *AtomStack) Push(a atom.Atom) *AtomStack {
	stack.s = append(stack.s, a)
	return stack
}

func (stack *AtomStack) Pop() atom.Atom {
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

type ContentHandler struct {
	title string
	time  time.Time

	tokenBuffer *bytes.Buffer
	textBuffer  *bytes.Buffer

	depthBody       int
	depthAnchor     int
	depthIgnoreable int

	depthTag      int
	depthBlockTag int

	sbLastWasWhitespace bool
	textElementIndex    int

	textBlocks []*TextBlock

	lastStartTag string
	lastEndTag   string

	offsetBlocks int
	//private BitSet currentContainedTextElements = new BitSet();

	flush        bool
	inAnchorText bool

	labelStacks *list.List
	// TODO: LinkedList<Integer> fontSizeStack = new LinkedList<Integer>();

	errs []error

	atomStack *AtomStack
}

func NewContentHandler() *ContentHandler {
	return &ContentHandler{
		tokenBuffer: &bytes.Buffer{},
		textBuffer:  &bytes.Buffer{},

		depthBlockTag: -1,

		textBlocks: make([]*TextBlock, 0),

		labelStacks: list.New(),

		errs: make([]error, 0),

		atomStack: NewAtomStack(),
	}
}

func (h *ContentHandler) Errors() []error {
	return h.errs
}

func (h *ContentHandler) String() string {
	return fmt.Sprintf("ContentHandler{ len(textBlocks): %d, tokenBuffer.Len(): %d, textBuffer.Len(): %d, depthBody: %d, depthAnchor: %d, depthIgnoreable: %d, depthTag: %d, depthBlockTag: %d, sbLastWasWhitespace: %t, textElementIndex: %d, lastStartTag: %s, lastEndTag: %s, offsetBlocks: %d, flush: %t, inAnchorText: %t }",
		len(h.textBlocks),
		h.tokenBuffer.Len(),
		h.textBuffer.Len(),
		h.depthBody,
		h.depthAnchor,
		h.depthIgnoreable,
		h.depthTag,
		h.depthBlockTag,
		h.sbLastWasWhitespace,
		h.textElementIndex,
		h.lastStartTag,
		h.lastEndTag,
		h.offsetBlocks,
		h.flush,
		h.inAnchorText)
}

func (h *ContentHandler) StartElement(z *html.Tokenizer) {
	h.labelStacks.PushBack(nil)

	tn, _ := z.TagName()
	a := atom.Lookup(tn)

	h.atomStack.Push(a)

	ta, ok := TagActionMap[a]
	if ok {
		switch ta.(type) {
		case TagActionTime:
			for {
				key, val, _ := z.TagAttr()
				if key == nil {
					break
				} else {
					keyS := string(key)
					if keyS == "datetime" {
						t, err := time.Parse(time.RFC3339, string(val))
						if err == nil {
							h.time = t
						}
						break
					}
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

	h.lastStartTag = a.String()

}

func (h *ContentHandler) EndElement(z *html.Tokenizer) {
	tn, _ := z.TagName()
	a := atom.Lookup(tn)

	pa := h.atomStack.Pop()
	if pa != a {
		return // malformed HTML, missing closing tag
	}

	ta, ok := TagActionMap[a]
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

	h.lastEndTag = a.String()

	h.labelStacks.Remove(h.labelStacks.Back())
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

func (h *ContentHandler) TextToken(z *html.Tokenizer) {
	h.textElementIndex++

	if h.flush {
		h.FlushBlock()
		h.flush = false
	}

	if h.depthIgnoreable != 0 {
		return
	}

	t := string(z.Text())
	if len(t) == 0 {
		return
	}

	sr := &spaceRemover{}
	ch := strings.TrimSpace(strings.Map(sr.getSpaceRemovalFunc(), t))
	if len(ch) == 0 {
		if sr.wasFirstWhitespace || sr.wasLastWhitespace {
			if h.sbLastWasWhitespace == false {
				h.textBuffer.WriteRune(' ')
				h.tokenBuffer.WriteRune(' ')
			}
			h.sbLastWasWhitespace = true
		} else {
			h.sbLastWasWhitespace = false
		}

		return
	}

	if sr.wasFirstWhitespace {
		if h.sbLastWasWhitespace == false {
			h.textBuffer.WriteRune(' ')
			h.tokenBuffer.WriteRune(' ')
		}
	}

	if h.depthBlockTag == -1 {
		h.depthBlockTag = h.depthTag
	}

	h.textBuffer.WriteString(ch)
	h.tokenBuffer.WriteString(ch)
	if sr.wasLastWhitespace {
		h.textBuffer.WriteRune(' ')
		h.tokenBuffer.WriteRune(' ')
	}

	h.sbLastWasWhitespace = sr.wasLastWhitespace

	// TODO: currentContainedTextElements.set(h.textElementIndex);
}

func tokenize(b *bytes.Buffer) []string {
	return reMultiSpace.Split(b.String(), -1)
}

var reValidWordCharacter = regexp.MustCompile(`[\w]`)

func isWord(tok string) bool {
	return reValidWordCharacter.MatchString(tok)
}

func (h *ContentHandler) FlushBlock() {
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
		if h.sbLastWasWhitespace {
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
		h.addTextBlock(NewTextBlock(
			text,
			numWords,
			numLinkedWords,
			numWordsInWrappedLines,
			numWrappedLines,
			h.offsetBlocks,
			h.depthBlockTag,
		))
		// TODO: currentContainedTextElements = new BitSet();
		h.offsetBlocks++
	}

	h.textBuffer.Reset()
	h.tokenBuffer.Reset()

	h.depthBlockTag = -1
}

func (h *ContentHandler) addTextBlock(tb *TextBlock) {
	// TODO:
	//for (Integer l : fontSizeStack) {
	//  if (l != null) {
	//    tb.addLabel("font-" + l);
	//    break;
	//  }
	//}

	for e := h.labelStacks.Back(); e != nil; e = e.Prev() {
		if e.Value != nil {
			labelStack := e.Value.(*list.List)

			for e1 := labelStack.Back(); e1 != nil; e1 = e1.Prev() {
				if e1.Value != nil {
					labelActions := e1.Value.(*LabelAction)
					labelActions.AddTo(tb)
				}
			}
		}
	}

	h.textBlocks = append(h.textBlocks, tb)
}

func (h *ContentHandler) addWhitespaceIfNecessary() {
	if h.sbLastWasWhitespace == false {
		h.tokenBuffer.WriteRune(' ')
		h.textBuffer.WriteRune(' ')
		h.sbLastWasWhitespace = true
	}
}

func (h *ContentHandler) addLabelAction(la *LabelAction) {
	var labelStack *list.List
	el := h.labelStacks.Back()

	if el.Value == nil {
		labelStack = list.New()
		h.labelStacks.Remove(h.labelStacks.Back())
		h.labelStacks.PushBack(labelStack)
	} else {
		labelStack = el.Value.(*list.List)
	}

	labelStack.PushBack(la)
}

type TagAction interface {
	Start(*ContentHandler) bool
	End(*ContentHandler) bool
	ChangesTagLevel() bool
}

type TagActionIgnorable struct{}

func (ta TagActionIgnorable) Start(h *ContentHandler) bool {
	h.depthIgnoreable++
	return true
}

func (TagActionIgnorable) End(h *ContentHandler) bool {
	h.depthIgnoreable--
	return true
}

func (TagActionIgnorable) ChangesTagLevel() bool { return true }

type TagActionAnchor struct{}

func (ta TagActionAnchor) Start(h *ContentHandler) bool {
	if h.depthAnchor > 0 {
		h.errs = append(h.errs, errors.New("input contains nested <a> elements"))
		return false
	}

	h.depthAnchor++

	if h.depthIgnoreable == 0 {
		h.addWhitespaceIfNecessary()
		h.tokenBuffer.WriteString(anchorTextStart)
		h.tokenBuffer.WriteRune(' ')
		h.sbLastWasWhitespace = true
	}

	return false
}

func (TagActionAnchor) End(h *ContentHandler) bool {
	if h.depthAnchor == 0 {
		h.errs = append(h.errs, errors.New("input contains unopened </a> element"))
		return false
	}

	h.depthAnchor--

	if h.depthAnchor == 0 {
		if h.depthIgnoreable == 0 {
			h.addWhitespaceIfNecessary()
			h.tokenBuffer.WriteString(anchorTextEnd)
			h.tokenBuffer.WriteRune(' ')
			h.sbLastWasWhitespace = true
		}
	}

	return false
}

func (TagActionAnchor) ChangesTagLevel() bool { return true }

type TagActionBody struct{}

func (ta TagActionBody) Start(h *ContentHandler) bool {
	h.FlushBlock()
	h.depthBody++
	return false
}
func (TagActionBody) End(h *ContentHandler) bool {
	h.FlushBlock()
	h.depthBody--
	return false
}

func (TagActionBody) ChangesTagLevel() bool { return true }

type TagActionInlineWhitespace struct{}

func (ta TagActionInlineWhitespace) Start(h *ContentHandler) bool {
	h.addWhitespaceIfNecessary()
	return false
}

func (TagActionInlineWhitespace) End(h *ContentHandler) bool {
	h.addWhitespaceIfNecessary()
	return false
}

func (TagActionInlineWhitespace) ChangesTagLevel() bool { return false }

type TagActionInlineNoWhitespace struct{}

func (TagActionInlineNoWhitespace) Start(h *ContentHandler) bool { return false }
func (TagActionInlineNoWhitespace) End(h *ContentHandler) bool   { return false }
func (TagActionInlineNoWhitespace) ChangesTagLevel() bool        { return false }

type TagActionBlockTagLabel struct{ labelAction *LabelAction }

func (ta TagActionBlockTagLabel) Start(h *ContentHandler) bool {
	h.addLabelAction(ta.labelAction)
	return true
}
func (TagActionBlockTagLabel) End(h *ContentHandler) bool { return true }
func (TagActionBlockTagLabel) ChangesTagLevel() bool      { return true }

type TagActionIgnoreableVoid struct{}

func (TagActionIgnoreableVoid) Start(h *ContentHandler) bool { return false }
func (TagActionIgnoreableVoid) End(h *ContentHandler) bool   { return false }
func (TagActionIgnoreableVoid) ChangesTagLevel() bool        { return false }

type TagActionTime struct{}

func (TagActionTime) Start(h *ContentHandler) bool { return true }
func (TagActionTime) End(h *ContentHandler) bool   { return true }
func (TagActionTime) ChangesTagLevel() bool        { return true }

// From DefaultTagActionMap.java
var TagActionMap = map[atom.Atom]TagAction{
	atom.Applet:     TagActionIgnorable{},
	atom.Figcaption: TagActionIgnorable{},
	atom.Figure:     TagActionIgnorable{},
	atom.Noscript:   TagActionIgnorable{},
	atom.Object:     TagActionIgnorable{},
	atom.Option:     TagActionIgnorable{},
	atom.Script:     TagActionIgnorable{},
	atom.Style:      TagActionIgnorable{},

	atom.A: TagActionAnchor{},

	atom.Body: TagActionBody{},

	atom.Abbr: TagActionInlineWhitespace{},
	// no atom.Acronym

	atom.B:      TagActionInlineNoWhitespace{},
	atom.Code:   TagActionInlineNoWhitespace{},
	atom.Em:     TagActionInlineNoWhitespace{},
	atom.Font:   TagActionInlineNoWhitespace{}, // can also use TA_FONT
	atom.I:      TagActionInlineNoWhitespace{},
	atom.Span:   TagActionInlineNoWhitespace{},
	atom.Strike: TagActionInlineNoWhitespace{},
	atom.Strong: TagActionInlineNoWhitespace{},
	atom.Sub:    TagActionInlineNoWhitespace{},
	atom.Sup:    TagActionInlineNoWhitespace{},
	atom.Tt:     TagActionInlineNoWhitespace{},
	atom.U:      TagActionInlineNoWhitespace{},
	atom.Var:    TagActionInlineNoWhitespace{},

	atom.Li: TagActionBlockTagLabel{NewLabelAction(LabelList)},
	atom.H1: TagActionBlockTagLabel{NewLabelAction(LabelHeading, LabelHeading1)},
	atom.H2: TagActionBlockTagLabel{NewLabelAction(LabelHeading, LabelHeading2)},
	atom.H3: TagActionBlockTagLabel{NewLabelAction(LabelHeading, LabelHeading3)},

	atom.Area:     TagActionIgnoreableVoid{},
	atom.Base:     TagActionIgnoreableVoid{},
	atom.Br:       TagActionIgnoreableVoid{},
	atom.Col:      TagActionIgnoreableVoid{},
	atom.Embed:    TagActionIgnoreableVoid{},
	atom.Hr:       TagActionIgnoreableVoid{},
	atom.Img:      TagActionIgnoreableVoid{},
	atom.Input:    TagActionIgnoreableVoid{},
	atom.Link:     TagActionIgnoreableVoid{},
	atom.Menuitem: TagActionIgnoreableVoid{},
	atom.Meta:     TagActionIgnoreableVoid{},
	atom.Param:    TagActionIgnoreableVoid{},
	atom.Source:   TagActionIgnoreableVoid{},
	atom.Track:    TagActionIgnoreableVoid{},
	atom.Wbr:      TagActionIgnoreableVoid{},

	atom.Time: TagActionTime{},
}

type LabelAction struct{ labels []Label }

func NewLabelAction(labels ...Label) *LabelAction {
	la := &LabelAction{
		labels: make([]Label, 0),
	}
	la.labels = append(la.labels, labels...)
	return la
}

func (la *LabelAction) AddTo(tb *TextBlock) {
	tb.AddLabels(la.labels...)
}
