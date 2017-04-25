package boilerpipe

import (
	"bytes"
	"container/list"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const ANCHOR_TEXT_START = "$\ue00a<"
const ANCHOR_TEXT_END = ">\ue00a$"

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
		if tok == ANCHOR_TEXT_START {
			h.inAnchorText = true
		} else if tok == ANCHOR_TEXT_END {
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
