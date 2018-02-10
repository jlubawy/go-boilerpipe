package boilerpipe

import (
	"errors"
	"testing"
)

func TestErrorSlice(t *testing.T) {
	es := ErrorSlice([]error{
		errors.New("a"),
		errors.New("b"),
		errors.New("c"),
	})

	if es.Error() != "a\nb\nc\n" {
		t.Error("unexpected error output")
	}
}
