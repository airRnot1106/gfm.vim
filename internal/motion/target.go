package motion

import (
	"regexp"
	"slices"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/airRnot1106/gks.vim/migemo"

	"github.com/airRnot1106/gfm.vim/internal/fzf"
)

const kensakuScore = 1

// Pos is a one-based line and byte column.
type Pos struct {
	Line int `json:"line"`
	Col  int `json:"col"`
}

// Word is a candidate of the window. Text is what the word regular expressions
// matched at Pos.
type Word struct {
	Text string `json:"text"`
	Pos  Pos    `json:"pos"`
}

// Target is a match with its label. Start and End are byte offsets in Text.
type Target struct {
	Word
	Char  string `json:"char"`
	Start int    `json:"start"`
	End   int    `json:"end"`
	Score int    `json:"score"`
}

// Request is everything a query needs, so that no handler calls the editor.
// Previous are the targets of the last input, whose labels stay.
type Request struct {
	Input    string   `json:"input"`
	Lines    []string `json:"lines"`
	First    int      `json:"first"`
	Regexps  []string `json:"regexps"`
	Filters  []string `json:"filters"`
	Labels   []string `json:"labels"`
	Matchers []string `json:"matchers"`
	Previous []Target `json:"previous"`
}

func compileAll(patterns []string) ([]*regexp.Regexp, error) {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := compile(p)
		if err != nil {
			return nil, err
		}
		out = append(out, re)
	}
	return out, nil
}

// words returns the candidates line by line, and within a line regular
// expression by regular expression.
func words(r Request) ([]Word, error) {
	regexps, err := compileAll(r.Regexps)
	if err != nil {
		return nil, err
	}
	filters, err := compileAll(r.Filters)
	if err != nil {
		return nil, err
	}
	var out []Word
	for i, line := range r.Lines {
		for _, re := range regexps {
		next:
			for _, loc := range re.FindAllStringIndex(line, -1) {
				text := line[loc[0]:loc[1]]
				for _, filter := range filters {
					if !filter.MatchString(text) {
						continue next
					}
				}
				out = append(out, Word{text, Pos{r.First + i, loc[0] + 1}})
			}
		}
	}
	return out, nil
}

// byteOffset converts an offset in UTF-16 code units of text.
func byteOffset(text string, units int) int {
	offset := 0
	for units > 0 && offset < len(text) {
		r, size := utf8.DecodeRuneInString(text[offset:])
		offset += size
		units -= max(utf16.RuneLen(r), 1)
	}
	return offset
}

type result struct {
	word       Word
	start, end int
	score      int
}

func fzfResults(words []Word, input string) []result {
	texts := make([]string, len(words))
	for i, w := range words {
		texts[i] = w.Text
	}
	var out []result
	for _, r := range fzf.Find(texts, input) {
		text := texts[r.Index]
		out = append(out, result{words[r.Index], byteOffset(text, max(r.Start, 0)), byteOffset(text, max(r.End, 0)), r.Score})
	}
	return out
}

func kensakuResults(m *migemo.Migemo, words []Word, input string) ([]result, error) {
	pattern, err := m.Query(input, migemo.JavaScript)
	if err != nil {
		return nil, err
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	var out []result
	for _, w := range words {
		for _, loc := range re.FindAllStringIndex(w.Text, -1) {
			out = append(out, result{w, loc[0], loc[1], kensakuScore})
		}
	}
	return out, nil
}

// label gives each result the label its position had, then the free labels in
// order. Results past the last label are dropped.
func label(results []result, labels []string, previous []Target) []Target {
	type position struct {
		pos   Pos
		start int
	}
	kept := make(map[position]string, len(previous))
	used := make(map[string]bool, len(previous))
	for _, t := range previous {
		kept[position{t.Pos, t.Start}] = t.Char
		used[t.Char] = true
	}
	next := 0
	targets := []Target{}
	for _, r := range results {
		char, ok := kept[position{r.word.Pos, r.start}]
		if !ok {
			for next < len(labels) && used[labels[next]] {
				next++
			}
			if next >= len(labels) {
				continue
			}
			char = labels[next]
			used[char] = true
		}
		targets = append(targets, Target{r.word, char, r.start, r.end, r.score})
	}
	return targets
}

// Answer is the outcome of an input. Narrowed is the target of an input that
// left no other target among those labeled before it, which is what
// g:fuzzy_motion_auto_jump jumps to.
type Answer struct {
	Targets  []Target `json:"targets"`
	Narrowed *Target  `json:"narrowed"`
}

// answer runs the matchers on the words of r. m is nil when kensaku is not
// among them. fzf results come first, and a position keeps its first result.
func answer(r Request, m *migemo.Migemo) (Answer, error) {
	if r.Input == "" {
		return Answer{Targets: []Target{}}, nil
	}
	candidates, err := words(r)
	if err != nil {
		return Answer{}, err
	}
	var results []result
	if slices.Contains(r.Matchers, "fzf") {
		results = fzfResults(candidates, r.Input)
	}
	if m != nil && len(utf16.Encode([]rune(r.Input))) >= 2 {
		found, err := kensakuResults(m, candidates, r.Input)
		if err != nil {
			return Answer{}, err
		}
		results = append(results, found...)
	}
	seen := make(map[Pos]bool, len(results))
	unique := results[:0]
	for _, c := range results {
		at := Pos{c.word.Pos.Line, c.word.Pos.Col + c.start}
		if !seen[at] {
			seen[at] = true
			unique = append(unique, c)
		}
	}
	// The first pass keeps the labels of the targets that stay. The second gives
	// the labels of the targets that left to the new ones.
	kept := label(unique, r.Labels, r.Previous)
	a := Answer{Targets: label(unique, r.Labels, kept)}
	if len(kept) == 1 {
		a.Narrowed = &kept[0]
	}
	return a, nil
}
