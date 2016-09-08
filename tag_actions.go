package boilerpipe

import (
	"errors"

	"golang.org/x/net/html/atom"
)

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
		panic(errors.New("input contains nested <a> elements"))
	}

	h.depthAnchor++

	if h.depthIgnoreable == 0 {
		h.addWhitespaceIfNecessary()
		h.tokenBuffer.WriteString(ANCHOR_TEXT_START)
		h.tokenBuffer.WriteRune(' ')
		h.sbLastWasWhitespace = true
	}

	return false
}
func (TagActionAnchor) End(h *ContentHandler) bool {
	h.depthAnchor--

	if h.depthAnchor == 0 {
		if h.depthIgnoreable == 0 {
			h.addWhitespaceIfNecessary()
			h.tokenBuffer.WriteString(ANCHOR_TEXT_END)
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

func (ta TagActionInlineNoWhitespace) Start(h *ContentHandler) bool {
	return false
}

func (TagActionInlineNoWhitespace) End(h *ContentHandler) bool { return false }

func (TagActionInlineNoWhitespace) ChangesTagLevel() bool { return false }

type TagActionIgnoreableVoid struct{}

func (ta TagActionIgnoreableVoid) Start(h *ContentHandler) bool { return false }
func (TagActionIgnoreableVoid) End(h *ContentHandler) bool      { return false }
func (TagActionIgnoreableVoid) ChangesTagLevel() bool           { return false }

// From DefaultTagActionMap.java
var TagActionMap = map[atom.Atom]TagAction{
	atom.Applet:   TagActionIgnorable{},
	atom.Noscript: TagActionIgnorable{},
	atom.Object:   TagActionIgnorable{},
	atom.Option:   TagActionIgnorable{},
	atom.Script:   TagActionIgnorable{},
	atom.Style:    TagActionIgnorable{},

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

	// TODO: New in 1.3
	//setTagAction("LI", new CommonTagActions.BlockTagLabelAction(new LabelAction(DefaultLabels.LI)));
	//setTagAction("H1", new CommonTagActions.BlockTagLabelAction(new LabelAction(DefaultLabels.H1,
	//    DefaultLabels.HEADING)));
	//setTagAction("H2", new CommonTagActions.BlockTagLabelAction(new LabelAction(DefaultLabels.H2,
	//    DefaultLabels.HEADING)));
	//setTagAction("H3", new CommonTagActions.BlockTagLabelAction(new LabelAction(DefaultLabels.H3,
	//    DefaultLabels.HEADING)));

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
}
