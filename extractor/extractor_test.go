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

func getFilename(p string) string {
	return p[:strings.LastIndex(p, ".")]
}

func replaceExtension(p string, ext string) string {
	return getFilename(p) + "." + ext
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

		doc, err := boilerpipe.NewTextDocument(f, nil)
		if err != nil {
			t.Error(err)
		}

		// Process the HTML document
		Article().Process(doc)
		actStr := doc.Content()

		expTimeStr, ok := expTimes[htmlFilename]
		if !ok {
			t.Errorf("missing expected time for article '%s'", htmlFilename)
		}

		if expTimeStr != "" {
			expTime, _ := time.Parse(time.RFC3339, expTimeStr)
			if !doc.Date.Equal(expTime) {
				t.Errorf("expected time %s does not match actual time %s", expTime, doc.Date)
			}
		} else {
			t.Logf("Skipping time check for article '%s'", htmlFilename)
		}

		// Write output to test results file
		if err := ioutil.WriteFile(filepath.Join("testresults", txtFilename), []byte(actStr), 0644); err != nil {
			t.Error(err)
		}

		actStr = strings.Replace(actStr, "\r\n", "\n", -1)
		expStr := strings.Replace(string(expText), "\r\n", "\n", -1)

		// Compare to the expected
		if actStr != expStr {
			t.Errorf("expected does not match actual")
		}

		return nil
	}

	if err := filepath.Walk("testdata", walkFn); err != nil {
		t.Error(err)
	}
}
