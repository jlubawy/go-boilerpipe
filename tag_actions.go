package boilerpipe

import (
	"errors"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type TagAction interface {
	Start(*ContentHandler, html.Token) bool
	End(*ContentHandler, html.Token) bool
	ChangesTagLevel() bool
}

type TagActionIgnorable struct{}

func (TagActionIgnorable) Start(h *ContentHandler, t html.Token) bool {
	h.depthIgnoreable++
	return true
}

func (TagActionIgnorable) End(h *ContentHandler, t html.Token) bool {
	h.depthIgnoreable--
	return true
}

func (TagActionIgnorable) ChangesTagLevel() bool { return true }

type TagActionAnchor struct{}

func (ta TagActionAnchor) Start(h *ContentHandler, t html.Token) bool {
	if h.depthAnchor > 0 {
		panic(errors.New("input contains nested <a> elements"))
		ta.End(h, t)
	}

	h.depthAnchor++

	if h.depthIgnoreable == 0 {
		h.addWhitespaceIfNecessary()
		h.tokenBuffer.WriteString(atom.A.String())
		h.tokenBuffer.WriteRune(' ')
		h.sbLastWasWhitespace = true
	}

	return false
}
func (TagActionAnchor) End(h *ContentHandler, t html.Token) bool {
	h.depthAnchor--

	if h.depthAnchor == 0 {
		if h.depthIgnoreable == 0 {
			h.addWhitespaceIfNecessary()
			h.tokenBuffer.WriteString(atom.A.String())
			h.tokenBuffer.WriteRune(' ')
			h.sbLastWasWhitespace = true
		}
	}

	return false
}

func (TagActionAnchor) ChangesTagLevel() bool { return true }

type TagActionBody struct{}

func (TagActionBody) Start(h *ContentHandler, t html.Token) bool {
	h.flushBlock()
	h.depthBody++
	return false
}
func (TagActionBody) End(h *ContentHandler, t html.Token) bool {
	h.flushBlock()
	h.depthBody--
	return false
}

func (TagActionBody) ChangesTagLevel() bool { return true }

type TagActionInlineWhitespace struct{}

func (TagActionInlineWhitespace) Start(h *ContentHandler, t html.Token) bool {
	h.addWhitespaceIfNecessary()
	return false
}

func (TagActionInlineWhitespace) End(h *ContentHandler, t html.Token) bool {
	h.addWhitespaceIfNecessary()
	return false
}

func (TagActionInlineWhitespace) ChangesTagLevel() bool { return false }

type TagActionInlineNoWhitespace struct{}

func (TagActionInlineNoWhitespace) Start(h *ContentHandler, t html.Token) bool { return false }
func (TagActionInlineNoWhitespace) End(h *ContentHandler, t html.Token) bool   { return false }

func (TagActionInlineNoWhitespace) ChangesTagLevel() bool { return false }

// From DefaultTagActionMap.java
var TagActionMap = map[atom.Atom]TagAction{
	atom.Style:    TagActionIgnorable{},
	atom.Script:   TagActionIgnorable{},
	atom.Option:   TagActionIgnorable{},
	atom.Object:   TagActionIgnorable{},
	atom.Embed:    TagActionIgnorable{},
	atom.Applet:   TagActionIgnorable{},
	atom.Link:     TagActionIgnorable{},
	atom.Noscript: TagActionIgnorable{},

	atom.A: TagActionAnchor{},

	atom.Body: TagActionBody{},

	atom.Abbr: TagActionInlineWhitespace{},
	// no atom.Acronym

	atom.Strike: TagActionInlineNoWhitespace{},
	atom.U:      TagActionInlineNoWhitespace{},
	atom.B:      TagActionInlineNoWhitespace{},
	atom.I:      TagActionInlineNoWhitespace{},
	atom.Em:     TagActionInlineNoWhitespace{},
	atom.Strong: TagActionInlineNoWhitespace{},
	atom.Span:   TagActionInlineNoWhitespace{},
	atom.Sup:    TagActionInlineNoWhitespace{},
	atom.Code:   TagActionInlineNoWhitespace{},
	atom.Tt:     TagActionInlineNoWhitespace{},
	atom.Sub:    TagActionInlineNoWhitespace{},
	atom.Var:    TagActionInlineNoWhitespace{},
	atom.Font:   TagActionInlineNoWhitespace{}, // can also use TA_FONT

	// New in 1.3
	//setTagAction("LI", new CommonTagActions.BlockTagLabelAction(new LabelAction(DefaultLabels.LI)));
	//setTagAction("H1", new CommonTagActions.BlockTagLabelAction(new LabelAction(DefaultLabels.H1,
	//    DefaultLabels.HEADING)));
	//setTagAction("H2", new CommonTagActions.BlockTagLabelAction(new LabelAction(DefaultLabels.H2,
	//    DefaultLabels.HEADING)));
	//setTagAction("H3", new CommonTagActions.BlockTagLabelAction(new LabelAction(DefaultLabels.H3,
	//    DefaultLabels.HEADING)));
}
