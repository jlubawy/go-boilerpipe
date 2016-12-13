package extractor

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jlubawy/go-boilerpipe"
)

func TestArticleExtractor(t *testing.T) {
	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil // skip directories
		}

		if filepath.Ext(path) != ".html" {
			return nil // skip non-html files
		}

		t.Logf("Opening test file: '%s'", path)

		dir := filepath.Dir(path)
		htmlFilename := filepath.Base(path)
		txtFilename := htmlFilename[:strings.LastIndex(htmlFilename, ".")] + ".txt"

		// Open the input html document
		f, err := os.Open(path)
		if err != nil {
			t.Error(err)
		}
		defer f.Close()

		// Open and read the expected article text
		expText, err := ioutil.ReadFile(filepath.Join(dir, txtFilename))
		if err != nil {
			t.Error(err)
		}

		doc, err := boilerpipe.NewTextDocument(f)
		if err != nil {
			t.Error(err)
		}

		// Process the HTML document
		EnableLogging("testresults", false)
		Article().Process(doc)
		actualContent := doc.Content()

		// Write output to test results file
		if err := ioutil.WriteFile(filepath.Join("testresults", txtFilename), []byte(actualContent), 0644); err != nil {
			t.Error(err)
		}

		// Compare to the expected
		if actualContent != string(expText) {
			t.Errorf("expected does not match actual")
		}

		return nil
	}

	if err := filepath.Walk("testdata", walkFn); err != nil {
		t.Error(err)
	}
}
