package noosexit_test

import (
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/cmd/staticlint/noosexit"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()

	analysistest.Run(t, testdata, noosexit.Analyzer,
		"badmain",
		"helperexit",
		"notmainpkg",
	)
}
