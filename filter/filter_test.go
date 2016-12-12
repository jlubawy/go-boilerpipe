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

var removeFirstData = map[string][]struct {
	TestStr  string
	Expected string
}{
	" - [^-]+$": {
		{
			TestStr:  "Day 18: Boilerpipe--Article Extraction for Java Developers &ndash; OpenShift Blog - www.example.com",
			Expected: "Day 18: Boilerpipe--Article Extraction for Java Developers &ndash; OpenShift Blog",
		},
		{
			TestStr:  "Day 18: Boilerpipe--Article Extraction for Java Developers &ndash; OpenShift Blog",
			Expected: "Day 18: Boilerpipe--Article Extraction for Java Developers &ndash; OpenShift Blog",
		},
	},

	"^[^-]+ - ": {
		{
			TestStr:  "Day 18: Boilerpipe--Article Extraction for Java Developers &ndash; OpenShift Blog ",
			Expected: "This is the titles",
		},
		{
			TestStr:  "Day 18: Boilerpipe--Article Extraction for Java Developers &ndash; OpenShift Blog",
			Expected: "Day 18: Boilerpipe--Article Extraction for Java Developers &ndash; OpenShift Blog",
		},
	},
}

func TestRemoveFirst(t *testing.T) {
	for pattern, patternData := range removeFirstData {
		t.Log(pattern)

		for i, td := range patternData {
			actual := RemoveFirst(td.TestStr, pattern)
			if td.Expected != actual {
				t.Errorf("[%d] expected='%s' != actual='%s'", i, td.Expected, actual)
			}
		}
	}
}
