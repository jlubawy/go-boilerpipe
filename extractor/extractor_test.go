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

		// Open the input html document
		f, err := os.Open(path)
		if err != nil {
			t.Error(err)
		}
		defer f.Close()

		// Open and read the expected article text
		expPath := path[:strings.LastIndex(path, ".")] + ".txt"
		expText, err := ioutil.ReadFile(expPath)
		if err != nil {
			t.Error(err)
		}

		doc, err := boilerpipe.NewTextDocument(f)
		if err != nil {
			t.Error(err)
		}

		//EnableLogging(true)
		Article().Process(doc)

		actualContent := doc.Content()
		//t.Logf(`expected='%s'\n`, expText)
		t.Logf(`actual='%s'\n`, actualContent)

		if actualContent != string(expText) {
			t.Fail()
		}

		return nil
	}

	if err := filepath.Walk("testdata", walkFn); err != nil {
		t.Error(err)
	}
}
