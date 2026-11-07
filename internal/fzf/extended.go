package fzf

import (
	"cmp"
	"regexp"
	"slices"
	"strings"
	"unicode/utf16"
)

type termType int

const (
	termFuzzy termType = iota
	termExact
	termPrefix
	termSuffix
	termEqual
)

var algorithms = map[termType]algorithm{
	termFuzzy:  fuzzyMatchV2,
	termExact:  exactMatch,
	termPrefix: prefixMatch,
	termSuffix: suffixMatch,
	termEqual:  equalMatch,
}

type term struct {
	typ           termType
	inverse       bool
	text          []int32
	caseSensitive bool
	normalize     bool
}

func units(s string) []int32 {
	encoded := utf16.Encode([]rune(s))
	out := make([]int32, len(encoded))
	for i, u := range encoded {
		out[i] = int32(u)
	}
	return out
}

func isJSSpace(r rune) bool { return isWhitespace(r) }

var spaces = regexp.MustCompile(` +`)

// parse reads the extended syntax into sets of alternatives. Every set has to
// match.
func parse(query string) [][]term {
	query = strings.TrimLeftFunc(query, isJSSpace)
	trimmed := strings.TrimRightFunc(query, isJSSpace)
	if strings.HasSuffix(trimmed, `\`) && strings.HasPrefix(query[len(trimmed):], " ") {
		query = trimmed + " "
	} else {
		query = trimmed
	}

	var sets [][]term
	var set []term
	switchSet, afterBar := false, false
	for _, token := range spaces.Split(strings.ReplaceAll(query, `\ `, "\t"), -1) {
		text := strings.ReplaceAll(token, "\t", " ")
		lowerText := strings.ToLower(text)
		caseSensitive := text != lowerText
		normalizeTerm := true
		for _, r := range units(lowerText) {
			if normalizeRune(r) != r {
				normalizeTerm = false
			}
		}
		if !caseSensitive {
			text = lowerText
		}
		if len(set) > 0 && !afterBar && text == "|" {
			switchSet = false
			afterBar = true
			continue
		}
		afterBar = false

		typ, inverse := termFuzzy, false
		if strings.HasPrefix(text, "!") {
			inverse = true
			typ = termExact
			text = text[1:]
		}
		if text != "$" && strings.HasSuffix(text, "$") {
			typ = termSuffix
			text = text[:len(text)-1]
		}
		if strings.HasPrefix(text, "'") {
			if inverse {
				typ = termFuzzy
			} else {
				typ = termExact
			}
			text = text[1:]
		} else if strings.HasPrefix(text, "^") {
			if typ == termSuffix {
				typ = termEqual
			} else {
				typ = termPrefix
			}
			text = text[1:]
		}
		if text == "" {
			continue
		}
		if switchSet {
			sets = append(sets, set)
			set = nil
		}
		runes := units(text)
		if normalizeTerm {
			for i, r := range runes {
				runes[i] = normalizeRune(r)
			}
		}
		set = append(set, term{typ, inverse, runes, caseSensitive, normalizeTerm})
		switchSet = true
	}
	if len(set) > 0 {
		sets = append(sets, set)
	}
	return sets
}

// Result is one matched text. Start and End are offsets in UTF-16 code units,
// and are -1 when the query matched without a position, as an inverse term does.
type Result struct {
	Index, Start, End, Score int
}

func matchSets(text []int32, sets [][]term) (Result, bool) {
	result := Result{Start: -1, End: -1}
	for _, set := range sets {
		matched, score := false, 0
		for _, t := range set {
			m, pos := algorithms[t.typ](t.caseSensitive, t.normalize, text, t.text)
			if m.start < 0 {
				if t.inverse {
					matched, score = true, 0
				}
				continue
			}
			if t.inverse {
				continue
			}
			matched, score = true, m.score
			if pos == nil {
				for i := m.start; i < m.end; i++ {
					pos = append(pos, i)
				}
			}
			for _, p := range pos {
				if result.Start < 0 || p < result.Start {
					result.Start = p
				}
				result.End = max(result.End, p+1)
			}
			break
		}
		if !matched {
			return result, false
		}
		result.Score += score
	}
	return result, true
}

// Find returns the texts that match query, the best score first and in the
// order of texts within a score.
func Find(texts []string, query string) []Result {
	sets := parse(query)
	var results []Result
	for i, text := range texts {
		if r, ok := matchSets(units(text), sets); ok {
			r.Index = i
			results = append(results, r)
		}
	}
	slices.SortStableFunc(results, func(a, b Result) int { return cmp.Compare(b.Score, a.Score) })
	return results
}
