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
		t.Error()
	}

	// True
	if !startsWithNumber("456 users responded in") {
		t.Error()
	}

	// False
	if startsWithNumber("abc comments") {
		t.Error()
	}

	// False
	if startsWithNumber("def users responded in") {
		t.Error()
	}
}
