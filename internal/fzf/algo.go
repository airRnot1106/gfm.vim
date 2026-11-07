// Package fzf is a port of the npm package fzf 0.5.2, the matcher of
// fuzzy-motion.vim. It finds with the extended syntax and the v2 algorithm.
//
// Text is a slice of UTF-16 code units, as in that package, and offsets count
// them.
package fzf

import (
	"slices"
	"unicode"
)

const (
	scoreMatch        = 16
	scoreGapStart     = -3
	scoreGapExtension = -1
	bonusBoundary     = scoreMatch / 2
	bonusNonWord      = scoreMatch / 2
	bonusCamel123     = bonusBoundary + scoreGapExtension
	bonusConsecutive  = -(scoreGapStart + scoreGapExtension)
	bonusFirstChar    = 2

	// v2Limit is the size of the score matrix past which v1 takes over.
	v2Limit  = 100 * 1024
	maxASCII = 127
)

type charClass int

const (
	classNonWord charClass = iota
	classLower
	classUpper
	classLetter
	classNumber
)

type match struct {
	start, end, score int
}

var noMatch = match{start: -1, end: -1}

// algorithm finds pattern in text and returns the matched offsets when it knows
// them one by one.
type algorithm func(caseSensitive, normalize bool, text, pattern []int32) (match, []int)

func normalizeRune(r int32) int32 {
	if r < 192 || r > 8580 {
		return r
	}
	if to, ok := normalized[r]; ok {
		return to
	}
	return r
}

func isWhitespace(r int32) bool {
	switch {
	case 0x2000 <= r && r <= 0x200a:
		return true
	case r == ' ', r == '\f', r == '\n', r == '\r', r == '\t', r == '\v', r == 0xa0, r == 0x1680,
		r == 0x2028, r == 0x2029, r == 0x202f, r == 0x205f, r == 0x3000, r == 0xfeff:
		return true
	}
	return false
}

func whitespacesAtStart(text []int32) int {
	n := 0
	for n < len(text) && isWhitespace(text[n]) {
		n++
	}
	return n
}

func whitespacesAtEnd(text []int32) int {
	n := 0
	for n < len(text) && isWhitespace(text[len(text)-1-n]) {
		n++
	}
	return n
}

func classOfASCII(r int32) charClass {
	switch {
	case 'a' <= r && r <= 'z':
		return classLower
	case 'A' <= r && r <= 'Z':
		return classUpper
	case '0' <= r && r <= '9':
		return classNumber
	}
	return classNonWord
}

func classOfNonASCII(r int32) charClass {
	switch {
	case unicode.ToUpper(r) != r:
		return classLower
	case unicode.ToLower(r) != r:
		return classUpper
	case unicode.IsNumber(r):
		return classNumber
	case unicode.IsLetter(r):
		return classLetter
	}
	return classNonWord
}

func classOf(r int32) charClass {
	if r <= maxASCII {
		return classOfASCII(r)
	}
	return classOfNonASCII(r)
}

func bonusFor(prev, curr charClass) int {
	switch {
	case prev == classNonWord && curr != classNonWord:
		return bonusBoundary
	case prev == classLower && curr == classUpper, prev != classNumber && curr == classNumber:
		return bonusCamel123
	case curr == classNonWord:
		return bonusNonWord
	}
	return 0
}

func bonusAt(text []int32, i int) int {
	if i == 0 {
		return bonusBoundary
	}
	return bonusFor(classOf(text[i-1]), classOf(text[i]))
}

func lower(r int32, caseSensitive bool) int32 {
	if caseSensitive {
		return r
	}
	if 'A' <= r && r <= 'Z' {
		return r + 32
	}
	if r > maxASCII {
		return unicode.ToLower(r)
	}
	return r
}

func trySkip(text []int32, caseSensitive bool, c int32, from int) int {
	rest := text[from:]
	i := slices.Index(rest, c)
	if i == 0 {
		return from
	}
	if !caseSensitive && 'a' <= c && c <= 'z' {
		if i > 0 {
			rest = rest[:i]
		}
		if u := slices.Index(rest, c-32); u >= 0 {
			i = u
		}
	}
	if i < 0 {
		return -1
	}
	return from + i
}

func isASCII(text []int32) bool {
	for _, r := range text {
		if r >= 128 {
			return false
		}
	}
	return true
}

// asciiFuzzyIndex returns where a match may start, or -1 when there is none.
func asciiFuzzyIndex(text, pattern []int32, caseSensitive bool) int {
	if !isASCII(text) {
		return 0
	}
	if !isASCII(pattern) {
		return -1
	}
	first, i := 0, 0
	for p, c := range pattern {
		if i > len(text) {
			return -1
		}
		i = trySkip(text, caseSensitive, c, i)
		if i < 0 {
			return -1
		}
		if p == 0 && i > 0 {
			first = i - 1
		}
		i++
	}
	return first
}

func fuzzyMatchV2(caseSensitive, normalize bool, input, pattern []int32) (match, []int) {
	M := len(pattern)
	if M == 0 {
		return match{}, nil
	}
	N := len(input)
	if N*M > v2Limit {
		return fuzzyMatchV1(caseSensitive, normalize, input, pattern)
	}
	idx := asciiFuzzyIndex(input, pattern, caseSensitive)
	if idx < 0 {
		return noMatch, nil
	}
	H0 := make([]int16, N)
	C0 := make([]int16, N)
	B := make([]int16, N)
	F := make([]int, M)
	T := slices.Clone(input)

	maxScore, maxScorePos := 0, 0
	pidx, lastIdx := 0, 0
	pchar0, pchar := pattern[0], pattern[0]
	prevH0 := 0
	prevClass := classNonWord
	inGap := false
	for i := idx; i < N; i++ {
		c := T[i]
		var class charClass
		if c <= maxASCII {
			class = classOfASCII(c)
			if !caseSensitive && class == classUpper {
				c += 32
			}
		} else {
			class = classOfNonASCII(c)
			if !caseSensitive && class == classUpper {
				c = unicode.ToLower(c)
			}
			if normalize {
				c = normalizeRune(c)
			}
		}
		T[i] = c
		bonus := bonusFor(prevClass, class)
		B[i] = int16(bonus)
		prevClass = class
		if c == pchar {
			if pidx < M {
				F[pidx] = i
				pidx++
				pchar = pattern[min(pidx, M-1)]
			}
			lastIdx = i
		}
		if c == pchar0 {
			score := scoreMatch + bonus*bonusFirstChar
			H0[i] = int16(score)
			C0[i] = 1
			if M == 1 && score > maxScore {
				maxScore = score
				maxScorePos = i
				if bonus == bonusBoundary {
					break
				}
			}
			inGap = false
		} else {
			if inGap {
				H0[i] = int16(max(prevH0+scoreGapExtension, 0))
			} else {
				H0[i] = int16(max(prevH0+scoreGapStart, 0))
			}
			C0[i] = 0
			inGap = true
		}
		prevH0 = int(H0[i])
	}
	if pidx != M {
		return noMatch, nil
	}
	if M == 1 {
		return match{maxScorePos, maxScorePos + 1, maxScore}, []int{maxScorePos}
	}

	f0 := F[0]
	width := lastIdx - f0 + 1
	H := make([]int16, width*M)
	copy(H, H0[f0:lastIdx+1])
	C := make([]int16, width*M)
	copy(C, C0[f0:lastIdx+1])

	for p := 1; p < M; p++ {
		f := F[p]
		c := pattern[p]
		row := p * width
		gap := false
		H[row+f-f0-1] = 0
		for col := f; col <= lastIdx; col++ {
			at := row + col - f0
			var s1, s2, consecutive int
			if gap {
				s2 = int(H[at-1]) + scoreGapExtension
			} else {
				s2 = int(H[at-1]) + scoreGapStart
			}
			if c == T[col] {
				s1 = int(H[at-1-width]) + scoreMatch
				b := int(B[col])
				consecutive = int(C[at-1-width]) + 1
				if b == bonusBoundary {
					consecutive = 1
				} else if consecutive > 1 {
					b = max(b, max(bonusConsecutive, int(B[col-consecutive+1])))
				}
				if s1+b < s2 {
					s1 += int(B[col])
					consecutive = 0
				} else {
					s1 += b
				}
			}
			C[at] = int16(consecutive)
			gap = s1 < s2
			score := max(s1, s2, 0)
			if p == M-1 && score > maxScore {
				maxScore = score
				maxScorePos = col
			}
			H[at] = int16(score)
		}
	}

	var pos []int
	i, j := M-1, maxScorePos
	preferMatch := true
	for {
		I := i * width
		j0 := j - f0
		s := int(H[I+j0])
		var s1, s2 int
		if i > 0 && j >= F[i] {
			s1 = int(H[I-width+j0-1])
		}
		if j > F[i] {
			s2 = int(H[I+j0-1])
		}
		if s > s1 && (s > s2 || s == s2 && preferMatch) {
			pos = append(pos, j)
			if i == 0 {
				break
			}
			i--
		}
		preferMatch = C[I+j0] > 1 || I+width+j0+1 < len(C) && C[I+width+j0+1] > 0
		j--
	}
	return match{j, maxScorePos + 1, maxScore}, pos
}

func calculateScore(caseSensitive, normalize bool, text, pattern []int32, sidx, eidx int) (int, []int) {
	pidx, score, consecutive, firstBonus := 0, 0, 0, 0
	inGap := false
	var pos []int
	prevClass := classNonWord
	if sidx > 0 {
		prevClass = classOf(text[sidx-1])
	}
	for i := sidx; i < eidx; i++ {
		r := text[i]
		class := classOf(r)
		r = lower(r, caseSensitive)
		if normalize {
			r = normalizeRune(r)
		}
		if pidx < len(pattern) && r == pattern[pidx] {
			pos = append(pos, i)
			score += scoreMatch
			bonus := bonusFor(prevClass, class)
			if consecutive == 0 {
				firstBonus = bonus
			} else {
				if bonus == bonusBoundary {
					firstBonus = bonus
				}
				bonus = max(bonus, firstBonus, bonusConsecutive)
			}
			if pidx == 0 {
				score += bonus * bonusFirstChar
			} else {
				score += bonus
			}
			inGap = false
			consecutive++
			pidx++
		} else {
			if inGap {
				score += scoreGapExtension
			} else {
				score += scoreGapStart
			}
			inGap = true
			consecutive = 0
			firstBonus = 0
		}
		prevClass = class
	}
	return score, pos
}

func fuzzyMatchV1(caseSensitive, normalize bool, text, pattern []int32) (match, []int) {
	if len(pattern) == 0 {
		return match{}, nil
	}
	if asciiFuzzyIndex(text, pattern, caseSensitive) < 0 {
		return noMatch, nil
	}
	pidx, sidx, eidx := 0, -1, -1
	for i, r := range text {
		r = lower(r, caseSensitive)
		if normalize {
			r = normalizeRune(r)
		}
		if r == pattern[pidx] {
			if sidx < 0 {
				sidx = i
			}
			pidx++
			if pidx == len(pattern) {
				eidx = i + 1
				break
			}
		}
	}
	if sidx < 0 || eidx < 0 {
		return noMatch, nil
	}
	pidx--
	for i := eidx - 1; i >= sidx; i-- {
		if lower(text[i], caseSensitive) == pattern[pidx] {
			pidx--
			if pidx < 0 {
				sidx = i
				break
			}
		}
	}
	score, pos := calculateScore(caseSensitive, normalize, text, pattern, sidx, eidx)
	return match{sidx, eidx, score}, pos
}

func exactMatch(caseSensitive, normalize bool, text, pattern []int32) (match, []int) {
	if len(pattern) == 0 {
		return match{}, nil
	}
	if len(text) < len(pattern) || asciiFuzzyIndex(text, pattern, caseSensitive) < 0 {
		return noMatch, nil
	}
	pidx, bestPos, bonus, bestBonus := 0, -1, 0, -1
	for i := 0; i < len(text); i++ {
		r := lower(text[i], caseSensitive)
		if normalize {
			r = normalizeRune(r)
		}
		if pattern[pidx] == r {
			if pidx == 0 {
				bonus = bonusAt(text, i)
			}
			pidx++
			if pidx == len(pattern) {
				if bonus > bestBonus {
					bestPos = i
					bestBonus = bonus
				}
				if bonus == bonusBoundary {
					break
				}
				i -= pidx - 1
				pidx = 0
				bonus = 0
			}
		} else {
			i -= pidx
			pidx = 0
			bonus = 0
		}
	}
	if bestPos < 0 {
		return noMatch, nil
	}
	sidx, eidx := bestPos-len(pattern)+1, bestPos+1
	score, _ := calculateScore(caseSensitive, normalize, text, pattern, sidx, eidx)
	return match{sidx, eidx, score}, nil
}

// foldEqual compares text from offset with pattern as prefix and suffix matches do.
func foldEqual(caseSensitive, normalize bool, text []int32, offset int, pattern []int32) bool {
	for i, p := range pattern {
		r := text[offset+i]
		if !caseSensitive {
			r = unicode.ToLower(r)
		}
		if normalize {
			r = normalizeRune(r)
		}
		if r != p {
			return false
		}
	}
	return true
}

func prefixMatch(caseSensitive, normalize bool, text, pattern []int32) (match, []int) {
	if len(pattern) == 0 {
		return match{}, nil
	}
	trimmed := 0
	if !isWhitespace(pattern[0]) {
		trimmed = whitespacesAtStart(text)
	}
	if len(text)-trimmed < len(pattern) || !foldEqual(caseSensitive, normalize, text, trimmed, pattern) {
		return noMatch, nil
	}
	end := trimmed + len(pattern)
	score, _ := calculateScore(caseSensitive, normalize, text, pattern, trimmed, end)
	return match{trimmed, end, score}, nil
}

func suffixMatch(caseSensitive, normalize bool, text, pattern []int32) (match, []int) {
	trimmed := len(text)
	if len(pattern) == 0 || !isWhitespace(pattern[len(pattern)-1]) {
		trimmed -= whitespacesAtEnd(text)
	}
	if len(pattern) == 0 {
		return match{trimmed, trimmed, 0}, nil
	}
	diff := trimmed - len(pattern)
	if diff < 0 || !foldEqual(caseSensitive, normalize, text, diff, pattern) {
		return noMatch, nil
	}
	score, _ := calculateScore(caseSensitive, normalize, text, pattern, diff, trimmed)
	return match{diff, trimmed, score}, nil
}

func equalMatch(caseSensitive, normalize bool, text, pattern []int32) (match, []int) {
	n := len(pattern)
	if n == 0 {
		return noMatch, nil
	}
	start, end := 0, 0
	if !isWhitespace(pattern[0]) {
		start = whitespacesAtStart(text)
	}
	if !isWhitespace(pattern[n-1]) {
		end = whitespacesAtEnd(text)
	}
	if len(text)-start-end != n {
		return noMatch, nil
	}
	for i, p := range pattern {
		r := text[start+i]
		if !caseSensitive {
			r = unicode.ToLower(r)
		}
		if normalize {
			r, p = normalizeRune(r), normalizeRune(p)
		}
		if r != p {
			return noMatch, nil
		}
	}
	return match{start, start + n, (scoreMatch+bonusBoundary)*n + (bonusFirstChar-1)*bonusBoundary}, nil
}
