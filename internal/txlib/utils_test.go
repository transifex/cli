package txlib

import (
	"reflect"
	"sort"
	"testing"

	"github.com/transifex/cli/internal/txlib/config"
)

func TestFigureOutResources(t *testing.T) {
	resources := []config.Resource{
		{ProjectSlug: "abc", ResourceSlug: "def"},
		{ProjectSlug: "abc", ResourceSlug: "dfg"},
		{ProjectSlug: "oab", ResourceSlug: "def"},
	}
	cfg := config.Config{Local: &config.LocalConfig{Resources: resources}}

	test := func(pattern string, expectedStrings [][]string) {
		result, err := figureOutResources([]string{pattern}, &cfg)
		if err != nil {
			t.Error(err)
		}
		sort.SliceStable(result, func(i, j int) bool {
			if result[i].ProjectSlug == result[j].ProjectSlug {
				return result[i].ResourceSlug < result[j].ResourceSlug
			} else {
				return result[i].ProjectSlug < result[j].ProjectSlug
			}
		})
		var expected []*config.Resource
		for _, row := range expectedStrings {
			expected = append(
				expected,
				&config.Resource{ProjectSlug: row[0], ResourceSlug: row[1]},
			)
		}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Got wrong result with pattern %s\n", pattern)
		}
	}

	test("abc.def", [][]string{{"abc", "def"}})
	test("a*", [][]string{{"abc", "def"}, {"abc", "dfg"}})
	test("ab*", [][]string{{"abc", "def"}, {"abc", "dfg"}})
	test("abc*", [][]string{{"abc", "def"}, {"abc", "dfg"}})
	test("abc.*", [][]string{{"abc", "def"}, {"abc", "dfg"}})
	test("abc.d*", [][]string{{"abc", "def"}, {"abc", "dfg"}})
	test("abc.de*", [][]string{{"abc", "def"}})
	test("abc*def", [][]string{{"abc", "def"}})
	test("ab*def", [][]string{{"abc", "def"}})
	test("a*def", [][]string{{"abc", "def"}})
	test("*def", [][]string{{"abc", "def"}, {"oab", "def"}})
	test("abc*ef", [][]string{{"abc", "def"}})
	test("abc*f", [][]string{{"abc", "def"}})
	test("*bc.def", [][]string{{"abc", "def"}})
	test("*c.def", [][]string{{"abc", "def"}})
	test("*.def", [][]string{{"abc", "def"}, {"oab", "def"}})
	test("*bc.de*", [][]string{{"abc", "def"}})
	test("*bc*de*", [][]string{{"abc", "def"}})

	result, err := figureOutResources([]string{"foo*"}, &cfg)
	if result != nil || err == nil {
		t.Error("Did not get error with unfound pattern")
	}
}
