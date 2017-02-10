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
		h.errs = append(h.errs, errors.New("input contains nested <a> elements"))
		return false
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
	if h.depthAnchor == 0 {
		h.errs = append(h.errs, errors.New("input contains unopened </a> element"))
		return false
	}

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
