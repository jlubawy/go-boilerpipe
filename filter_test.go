package boilerpipe

import (
	"bytes"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
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

type extractJSON struct {
	Document []byte `json:"document"`
	URL      string `json:"url"`
	Results  struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Date    string `json:"date"`
		Content string `json:"content"`
	} `json:"results"`
}

func TestArticlePipeline(t *testing.T) {
	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil // skip directories
		}

		if filepath.Ext(path) != ".json" {
			t.Logf("Skipping file '%s'", path)
			return nil // skip non-html files
		}

		t.Logf("Testing file: '%s'", path)

		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		var testData extractJSON

		if err := json.NewDecoder(f).Decode(&testData); err != nil {
			t.Fatal(err)
		}

		u, err := url.Parse(testData.URL)
		if err != nil {
			t.Fatal(err)
		}

		doc, err := NewDocument(bytes.NewReader(testData.Document), u)
		if err != nil {
			t.Fatal(err)
		}
		ArticlePipeline().Process(doc)

		expected := testData.Results

		if doc.Title != expected.Title {
			errorf(t, "Title", doc.Title, expected.Title)
		}

		if doc.URL.String() != expected.URL {
			errorf(t, "URL", doc.URL, expected.URL)
		}

		if expected.Date != "" {
			expDate, err := time.Parse(time.RFC3339, expected.Date)
			if err != nil {
				t.Fatal(err)
			}

			if !doc.Date.Equal(expDate) {
				errorf(t, "Date", doc.Date, expDate)
			}
		} else {
			t.Logf("Skipping Date check...")
		}

		if doc.Content() != expected.Content {
			t.Errorf("Content mismatch")
		}

		return nil
	}

	if err := filepath.Walk("testdata", walkFn); err != nil {
		t.Error(err)
	}
}

func errorf(t *testing.T, name string, act, exp interface{}) {
	t.Errorf("%s mismatch (act=%s, exp=%s)", name, act, exp)
}
