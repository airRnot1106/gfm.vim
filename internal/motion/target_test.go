package motion

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/airRnot1106/gks.vim/migemo"
)

var (
	defaultRegexps = []string{`[0-9a-zA-Z_-]+`, `([0-9a-zA-Z_-]|[.])+`, `([0-9a-zA-Z_-]|[().#])+`, `\P{Script_Extensions=Latin}+`}
	defaultLabels  = strings.Split("ASDFGHJKLQWERTYUIOPZXCVBNM", "")
)

func fixture(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile("testdata/fixture.txt")
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

// The dictionary is GPL, so it is not in this repository. GKS_TEST_DICT
// names it; nix flake check sets it.
func dictionary(t *testing.T) *migemo.Migemo {
	t.Helper()
	path := os.Getenv("GKS_TEST_DICT")
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".cache/kensaku.vim/migemo-compact-dict")
	}
	if _, err := os.Stat(path); err != nil {
		t.Skip("no dictionary: ", err)
	}
	dict, err := migemo.Load(path, "")
	if err != nil {
		t.Fatal(err)
	}
	return migemo.New(dict)
}

// targets.json is what fuzzy_motion#targets() of fuzzy-motion.vim returned on
// fixture.txt with g:fuzzy_motion_matchers = ['kensaku', 'fzf'].
func TestTargetsEqualFuzzyMotion(t *testing.T) {
	data, err := os.ReadFile("testdata/targets.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden []struct {
		Query   string
		Targets []Target
	}
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatal(err)
	}
	m := dictionary(t)
	lines := fixture(t)
	for _, g := range golden {
		got, err := targets(Request{
			Input:    g.Query,
			Lines:    lines,
			First:    1,
			Regexps:  defaultRegexps,
			Labels:   defaultLabels,
			Matchers: []string{"kensaku", "fzf"},
		}, m)
		if err != nil {
			t.Fatalf("%q: %v", g.Query, err)
		}
		// fuzzy-motion.vim keeps fzf offsets in UTF-16 code units, and all
		// others in bytes. Only kensaku results have the score 1.
		want := make([]Target, len(g.Targets))
		for i, target := range g.Targets {
			want[i] = target
			if target.Score != kensakuScore {
				want[i].Start = byteOffset(target.Text, max(target.Start, 0))
				want[i].End = byteOffset(target.Text, max(target.End, 0))
			}
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%q:\n got %+v\nwant %+v", g.Query, got, want)
		}
	}
}

func TestLabelsStayWhileTheInputGrows(t *testing.T) {
	r := Request{
		Lines:    []string{"alpha", "alps", "beta"},
		First:    1,
		Regexps:  defaultRegexps[:1],
		Labels:   []string{"A", "S"},
		Matchers: []string{"fzf"},
	}
	r.Input = "a"
	first, err := targets(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Input, r.Previous = "alps", first
	second, err := targets(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || len(second) != 1 {
		t.Fatalf("%+v %+v", first, second)
	}
	if second[0].Pos.Line != 2 || second[0].Char != "S" {
		t.Errorf("got %+v", second[0])
	}
}

func TestResultsPastTheLastLabelAreDropped(t *testing.T) {
	got, err := targets(Request{
		Input:    "a",
		Lines:    []string{"a a a a"},
		First:    1,
		Regexps:  defaultRegexps[:1],
		Labels:   []string{"A", "S"},
		Matchers: []string{"fzf"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("got %+v", got)
	}
}

func TestKensakuNeedsTwoCharacters(t *testing.T) {
	m := dictionary(t)
	r := Request{Lines: []string{"日本語"}, First: 1, Regexps: defaultRegexps, Labels: defaultLabels, Matchers: []string{"kensaku"}}
	r.Input = "n"
	if got, _ := targets(r, m); len(got) != 0 {
		t.Errorf("got %+v", got)
	}
	r.Input = "ni"
	if got, _ := targets(r, m); len(got) != 1 || got[0].Start != 0 || got[0].End != 3 {
		t.Errorf("got %+v", got)
	}
}

func TestFzfOffsetsAreBytes(t *testing.T) {
	got, err := targets(Request{
		Input:    "x",
		Lines:    []string{"日本 x"},
		First:    1,
		Regexps:  defaultRegexps[3:],
		Labels:   defaultLabels,
		Matchers: []string{"fzf"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("x is Latin, so it is outside the word: %+v", got)
	}
	got, err = targets(Request{
		Input:    "1",
		Lines:    []string{"日本 1"},
		First:    1,
		Regexps:  defaultRegexps[3:],
		Labels:   defaultLabels,
		Matchers: []string{"fzf"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Start != len("日本 ") {
		t.Errorf("got %+v", got)
	}
}

func TestFilters(t *testing.T) {
	got, err := targets(Request{
		Input:    "a",
		Lines:    []string{"ab ac"},
		First:    1,
		Regexps:  defaultRegexps[:1],
		Filters:  []string{"c$"},
		Labels:   defaultLabels,
		Matchers: []string{"fzf"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Text != "ac" {
		t.Errorf("got %+v", got)
	}
}

func TestScriptExtensionsLatin(t *testing.T) {
	re, err := compile(`\P{Script_Extensions=Latin}+`)
	if err != nil {
		t.Fatal(err)
	}
	if got := re.FindString("abc日本語·def"); got != "日本語" {
		t.Errorf("got %q", got)
	}
}
