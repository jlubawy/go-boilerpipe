package boilerpipe

import (
	"bytes"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

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
	// lastEvent Event

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

func (h *ContentHandler) startElement(z *html.Tokenizer) {
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

	//lastEvent = Event.START_TAG
	h.lastStartTag = a.String()

}

func (h *ContentHandler) endElement(z *html.Tokenizer) {
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
		// TODO: h.flushBlock()
	}

	//lastEvent = Event.END_TAG
	h.lastEndTag = a.String()

	// TODO: labelStacks.removeLast()
}

func (h *ContentHandler) textToken(z *html.Tokenizer) {
	h.textElementIndex++

	if h.flush {
		// TODO: h.flushBlock();
		h.flush = false
	}

	if h.depthIgnoreable != 0 {
		return
	}

	ch := z.Text()
	// TODO: start := 0
	start := 0
	length := len(ch)

	var (
		c               rune
		startWhitespace bool
		endWhitespace   bool
	)

	if length == 0 {
		return
	}

	end := start + length

	for i := start; i < end; i++ {
		if unicode.IsSpace(rune(ch[i])) {
			ch[i] = ' '
		}
	}

	for start < end {
		c = rune(ch[start])

		if c == ' ' {
			startWhitespace = true
			start++
			length--
		} else {
			break
		}
	}

	for length > 0 {
		c = rune(ch[start+length-1])
		if c == ' ' {
			endWhitespace = true
			length--
		} else {
			break
		}
	}

	if length == 0 {
		if startWhitespace || endWhitespace {
			if h.sbLastWasWhitespace == false {
				h.textBuffer.WriteRune(' ')
				h.tokenBuffer.WriteRune(' ')
			}

			h.sbLastWasWhitespace = true
		} else {
			h.sbLastWasWhitespace = false
		}

		//lastEvent = Event.WHITESPACE;
		return
	}

	if startWhitespace {
		if h.sbLastWasWhitespace == false {
			h.textBuffer.WriteRune(' ')
			h.tokenBuffer.WriteRune(' ')
		}
	}

	if h.depthBlockTag == -1 {
		h.depthBlockTag = h.depthTag
	}

	h.textBuffer.Write(ch[start : start+length])
	h.tokenBuffer.Write(ch[start : start+length])

	if endWhitespace {
		h.textBuffer.WriteRune(' ')
		h.tokenBuffer.WriteRune(' ')
	}

	h.sbLastWasWhitespace = endWhitespace
	//lastEvent = Event.CHARACTERS;

	// TODO: currentContainedTextElements.set(h.textElementIndex);
}

func isWord(tok string) bool {
	return true
}

func (h *ContentHandler) flushBlock() {
	if h.depthBody == 0 {
		if h.lastStartTag == atom.Title.String() {
			h.title = strings.TrimSpace(h.tokenBuffer.String())
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

	tokens := make([]string, 0)
	//    final String[] tokens = UnicodeTokenizer.tokenize(tokenBuffer);

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
		if tok == "<a>" {
			h.inAnchorText = true
		} else if tok == "</a>" {
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

	tb := &TextBlock{
	// TODO: init
	}
	//TextBlock tb =
	//    new TextBlock(textBuffer.toString().trim(), currentContainedTextElements, numWords,
	//        numLinkedWords, numWordsInWrappedLines, numWrappedLines, offsetBlocks);

	// TODO: currentContainedTextElements = new BitSet();

	h.offsetBlocks++

	h.textBuffer.Reset()
	h.tokenBuffer.Reset()

	//tb.setTagLevel(h.depthBlockTag);
	h.textBlocks = append(h.textBlocks, tb)
	h.depthBlockTag = -1
}

func (h *ContentHandler) addWhitespaceIfNecessary() {
	if h.sbLastWasWhitespace == false {
		h.tokenBuffer.WriteString(" ")
		h.textBuffer.WriteString(" ")

		h.sbLastWasWhitespace = true
	}
}
