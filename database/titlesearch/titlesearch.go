// Package titlesearch matches search queries against the titles of games,
// bundles and collections, ignoring case, accents and punctuation.
package titlesearch

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Letters that are written without the mark or ligature when typed on
// another keyboard, and that NFKD leaves alone because they aren't a base
// letter with an accent.
var letterFolds = map[rune]string{
	'ß': "ss",
	'æ': "ae",
	'œ': "oe",
	'ø': "o",
	'ł': "l",
	'đ': "d",
	'ð': "d",
	'þ': "th",
	'ħ': "h",
	'ı': "i",
	// only written at the end of a word, and never what uppercase folds to
	'ς': "σ",
}

// searchWords folds a title or query down to what matching looks at:
// lowercase letters and digits without accents, each run of anything
// else being a word break. "X-Moon: Pokémon" gives x, moon, pokemon.
// Apostrophes are dropped without breaking the word, "Don't" gives dont.
func searchWords(s string) []string {
	var words []string
	var word strings.Builder
	flush := func() {
		if word.Len() > 0 {
			words = append(words, word.String())
			word.Reset()
		}
	}
	// NFKD splits marks from their letter. Those are accents to drop in
	// these scripts, but part of the letter or word in most others
	// (Japanese voicing marks, Devanagari vowel signs).
	accented := false
	for _, r := range norm.NFKD.String(s) {
		switch {
		case r == '\'' || r == '’' || r == '‘' || r == 'ʼ':
		case unicode.IsMark(r):
			if !accented && word.Len() > 0 {
				word.WriteRune(r)
			}
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			accented = unicode.In(r, unicode.Latin, unicode.Greek, unicode.Cyrillic)
			r = unicode.ToLower(r)
			if folded, ok := letterFolds[r]; ok {
				word.WriteString(folded)
			} else {
				word.WriteRune(r)
			}
		default:
			flush()
		}
	}
	flush()
	return words
}

// Title is a title folded for matching against queries.
type Title struct {
	// the title as it is, lowercased
	lower string
	// the title's words run together, so that "xmoon" finds "x-moon"
	key string
	// the title's words, each preceded by a space
	words string
	// Length of the title, in characters
	Length int
}

func NewTitle(title string) Title {
	words := searchWords(title)
	return Title{
		lower:  strings.ToLower(title),
		key:    strings.Join(words, ""),
		words:  " " + strings.Join(words, " "),
		Length: utf8.RuneCountInString(title),
	}
}

// Tier is how well a query matches a title, best first.
type Tier int

const (
	// the query is the title as it is, apart from case
	TierIdentical Tier = iota
	// the query is the whole title
	TierExact
	// the title starts with the query
	TierPrefix
	// every query word starts a word of the title, in any order
	TierWords
	// the query appears inside a word of the title, or in the title as
	// it was typed
	TierSubstring
	// as TierWords, with the title's words run together where they have
	// to be for the query to match: "xmoon" in "Super X-Moon"
	TierJoined
	TierNone
)

// Query is a search query, folded for matching against titles.
type Query struct {
	title string
	key   string
	// each word preceded by a space, to match against Title.words
	words []string
}

func NewQuery(query string) Query {
	words := searchWords(query)
	q := Query{
		title: strings.ToLower(strings.TrimSpace(query)),
		key:   strings.Join(words, ""),
	}
	for _, word := range words {
		// typed where the title may have an &, which isn't a word there
		if word == "and" {
			continue
		}
		q.words = append(q.words, " "+word)
	}
	if len(q.words) == 0 {
		// nothing but "and"
		for _, word := range words {
			q.words = append(q.words, " "+word)
		}
	}
	return q
}

// Empty tells if there is nothing in the query to match on.
func (q Query) Empty() bool {
	return q.title == ""
}

func (q Query) Tier(e *Title) Tier {
	// a query of punctuation only folds to nothing
	if q.key != "" {
		if tier := q.foldedTier(e); tier != TierNone {
			return tier
		}
	}
	// as typed, which is all there is to go by for punctuation, and what
	// finds a part of the title that starts or ends inside a word
	if e.lower == q.title {
		return TierIdentical
	}
	if strings.Contains(e.lower, q.title) {
		return TierSubstring
	}
	return TierNone
}

func (q Query) foldedTier(e *Title) Tier {
	if strings.HasPrefix(e.key, q.key) {
		if len(e.key) == len(q.key) {
			if e.lower == q.title {
				return TierIdentical
			}
			return TierExact
		}
		return TierPrefix
	}
	joined := false
	for _, word := range q.words {
		if strings.Contains(e.words, word) {
			continue
		}
		if !e.joinedStartsWith(word[1:]) {
			// the words don't each match, the whole query still might
			// words has no space inside a word, and key has none at all
			if strings.Contains(e.words, q.key) {
				return TierSubstring
			}
			if e.joinedStartsWith(q.key) {
				return TierJoined
			}
			return TierNone
		}
		joined = true
	}
	if joined {
		return TierJoined
	}
	return TierWords
}

// joinedStartsWith tells if the title, run together from one of its
// words on, starts with s.
func (e *Title) joinedStartsWith(s string) bool {
	if !strings.Contains(e.key, s) {
		return false
	}
	// each space in words precedes a word, and key is words without them
	spaces := 0
	for i := 0; i < len(e.words); i++ {
		if e.words[i] != ' ' {
			continue
		}
		spaces++
		if strings.HasPrefix(e.key[i+1-spaces:], s) {
			return true
		}
	}
	return false
}
