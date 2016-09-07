package boilerpipe

import (
	"bytes"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const ANCHOR_TEXT_START = "$\ue00a<"
const ANCHOR_TEXT_END = ">\ue00a$"

type ContentHandler struct {
	title string

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

	//LinkedList<LinkedList<LabelAction>> labelStacks = new LinkedList<LinkedList<LabelAction>>();
	//LinkedList<Integer> fontSizeStack = new LinkedList<Integer>();
}

func NewContentHandler() *ContentHandler {
	return &ContentHandler{
		tokenBuffer: &bytes.Buffer{},
		textBuffer:  &bytes.Buffer{},

		depthBlockTag: -1,

		textBlocks: make([]*TextBlock, 0),
	}
}

func (h *ContentHandler) StartElement(z *html.Tokenizer) {
	// TODO: labelStacks.add(null);

	tn, _ := z.TagName()
	a := atom.Lookup(tn)

	ta, ok := TagActionMap[a]
	if ok {
		if ta.ChangesTagLevel() {
			h.depthTag++
		}
		h.flush = ta.Start(h, z.Token()) || h.flush
	} else {
		h.depthTag++
		h.flush = true
	}

	h.lastStartTag = a.String()

}

func (h *ContentHandler) EndElement(z *html.Tokenizer) {
	tn, _ := z.TagName()
	a := atom.Lookup(tn)

	ta, ok := TagActionMap[a]
	if ok {
		h.flush = ta.End(h, z.Token()) || h.flush
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

	// TODO: labelStacks.removeLast()
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

var (
	reWordBoundary       = regexp.MustCompile("\\b")
	reNotWordBoundary    = regexp.MustCompile("[\u2063]*([\\\"'\\.,\\!\\@\\-\\:\\;\\$\\?\\(\\)/])[\u2063]*")
	reValidWordCharacter = regexp.MustCompile("[\\p{L}\\p{Nd}\\p{Nl}\\p{No}]")
)

func tokenize(s string) []string {
	return []string{} // TODO
}

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

	length := h.tokenBuffer.Len()

	switch length {
	case 0:
		return
	case 1:
		if h.sbLastWasWhitespace {
			h.textBuffer.Reset()
			h.tokenBuffer.Reset()
			return
		}
	}

	tokens := strings.Split(h.tokenBuffer.String(), " ")

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

	h.textBlocks = append(h.textBlocks, &TextBlock{
		Text: strings.TrimSpace(h.textBuffer.String()),
		// TODO: currentContainedTextElements,
		numWords:               numWords,
		numLinkedWords:         numLinkedWords,
		numWordsInWrappedLines: numWordsInWrappedLines,
		numWrappedLines:        numWrappedLines,
		offsetBlocks:           h.offsetBlocks,
		tagLevel:               h.depthBlockTag,
	})

	// TODO: currentContainedTextElements = new BitSet();
	h.offsetBlocks++

	h.textBuffer.Reset()
	h.tokenBuffer.Reset()

	h.depthBlockTag = -1
}

func (h *ContentHandler) addWhitespaceIfNecessary() {
	if h.sbLastWasWhitespace == false {
		h.tokenBuffer.WriteRune(' ')
		h.textBuffer.WriteRune(' ')
		h.sbLastWasWhitespace = true
	}
}
