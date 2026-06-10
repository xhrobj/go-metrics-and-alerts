package main

import (
	"strings"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/cmd/staticlint/noosexit"
)

// TestAnalyzersIncludeRequiredChecks фиксирует наличие обязательных
// представителей основных групп multichecker'а, как того требует описание И20.
func TestAnalyzersIncludeRequiredChecks(t *testing.T) {
	checks := analyzers()

	names := make(map[string]bool, len(checks))
	for _, check := range checks {
		names[check.Name] = true
	}

	for _, want := range []string{
		"printf",    // стандартный анализатор Go
		"copylocks", // стандартный анализатор Go
		"bodyclose", // публичный анализатор
		"nilerr",    // публичный анализатор
		"noosexit",  // собственный анализатор
	} {
		if !names[want] {
			t.Fatalf("got no analyzer %q", want)
		}
	}
}

// TestAnalyzersIncludeStaticcheckGroups фиксирует, что staticlint включает
// обязательный класс SA* и дополнительный класс S* из Staticcheck.
func TestAnalyzersIncludeStaticcheckGroups(t *testing.T) {
	checks := analyzers()

	var hasSA bool
	var hasSimple bool

	for _, check := range checks {
		if strings.HasPrefix(check.Name, "SA") {
			hasSA = true
		}

		if strings.HasPrefix(check.Name, "S") && !strings.HasPrefix(check.Name, "SA") {
			hasSimple = true
		}
	}

	if !hasSA {
		t.Fatal("got no SA analyzers")
	}

	if !hasSimple {
		t.Fatal("got no S analyzers")
	}
}

// TestAnalyzersIncludeNoosexitAnalyzer фиксирует, что в multichecker
// подключен собственный анализатор noosexit.
func TestAnalyzersIncludeNoosexitAnalyzer(t *testing.T) {
	checks := analyzers()

	for _, check := range checks {
		if check == noosexit.Analyzer {
			return
		}
	}

	t.Fatal("got no noosexit analyzer")
}
