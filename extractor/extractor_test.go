package extractor

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jlubawy/go-boilerpipe"
)

var expTimes = map[string]string{
	"0.html": "2013-11-15T00:00:00+00:00",
	"1.html": "",
}

func replaceExtension(s string, ext string) string {
	return s[:strings.LastIndex(s, ".")] + "." + ext
}

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
		txtFilename := replaceExtension(htmlFilename, "txt")

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

		expTimeStr, ok := expTimes[htmlFilename]
		if !ok {
			t.Errorf("missing expected time for article '%s'", htmlFilename)
		}

		if expTimeStr != "" {
			expTime, _ := time.Parse(time.RFC3339, expTimeStr)
			if !doc.Time.Equal(expTime) {
				t.Errorf("expected time %s does not match actual time %s", expTime, doc.Time)
			}
		} else {
			t.Logf("Skipping time check for article '%s'", htmlFilename)
		}

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
