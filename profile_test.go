package dnsmill

import (
	"os"
	"strings"
	"testing"

	"github.com/hexops/autogold/v2"
)

// testResult tuples the result and error of a test.
// It is used for autogold assertions.
type testResult[T any] struct {
	Result T
	Error  error
}

func TestParseProfileAsYAML(t *testing.T) {
	parseTestFile := mustReadFile(t, "testdata/parse_test.yml")
	parseTests := strings.Split(string(parseTestFile), "---")

	for i, parseTest := range parseTests {
		parseTest = strings.TrimSpace(parseTest)

		name, data, ok := strings.Cut(parseTest, "\n")
		if !ok {
			t.Fatalf("test %d is invalid: no test name", i+1)
			continue
		}
		name = strings.TrimSpace(strings.TrimPrefix(name, "#"))
		data = strings.TrimSpace(data)

		t.Run(name, func(t *testing.T) {
			p, err := parseProfileAsYAML(strings.NewReader(data))
			t.Logf("%#v", p)
			autogold.ExpectFile(t, testResult[*Profile]{p, err})
		})
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	return b
}
