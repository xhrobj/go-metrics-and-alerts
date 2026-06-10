package noosexit_test

import (
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/cmd/staticlint/noosexit"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	fixtures := []string{
		"badmain",
		"helperexit",
		"notmainpkg",
	}

	analysistest.Run(t, testdata, noosexit.Analyzer, fixtures...)
}
