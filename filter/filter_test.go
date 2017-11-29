package filter

import (
	"testing"
)

func TestStartsWithNumber(t *testing.T) {
	startsWithNumber := func(text string) bool {
		return StartsWithNumber(text, " comments", " users responded in")
	}

	// True
	if !startsWithNumber("123 comments") {
		t.Error("expected to start with number")
	}

	// True
	if !startsWithNumber("456 users responded in") {
		t.Error("expected to start with number")
	}

	// False
	if startsWithNumber("abc comments") {
		t.Error("not expected to start with number")
	}

	// False
	if startsWithNumber("def users responded in") {
		t.Error("not expected to start with number")
	}
}
