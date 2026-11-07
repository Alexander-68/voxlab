package server

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	rePeriod       = regexp.MustCompile(`(?i)\b(?:period|full stop)\b`)
	reComma        = regexp.MustCompile(`(?i)\bcomma\b`)
	reQuestion     = regexp.MustCompile(`(?i)\bquestion mark\b`)
	reExclamation  = regexp.MustCompile(`(?i)\bexclamation (?:mark|point)\b`)
	reSemicolon    = regexp.MustCompile(`(?i)\bsemicolon\b`)
	reColon        = regexp.MustCompile(`(?i)\bcolon\b`)
	reNewParagraph = regexp.MustCompile(`(?i)\bnew paragraph\b`)
	reNewLine      = regexp.MustCompile(`(?i)\bnew line\b`)

	rePrePunctSpace = regexp.MustCompile(`[ \t]+([.,;:?!])`)
	reNewlineSpaces = regexp.MustCompile(`[ \t]*\n[ \t]*`)
	reMultiNewlines = regexp.MustCompile(`\n{3,}`)
	reMultiSpaces   = regexp.MustCompile(`[ \t]{2,}`)
	reEndsWithPunct = regexp.MustCompile(`[.;,!?:\n。！？；：，]$`)
	reLeadingChar   = regexp.MustCompile(`^(\s*)([a-z])`)
	reAfterSentence = regexp.MustCompile(`([.?!]\s+)([a-z])`)
	reAfterNewline  = regexp.MustCompile(`(\n+\s*)([a-z])`)
)

var reSentenceEnd = regexp.MustCompile(`[.?!:\n。！？：]$`)

// rePunctRun matches the doubled punctuation left when the ASR punctuates around a spoken
// keyword ("stable period, the wound"). reNewlinePunct matches the same after a line break.
var (
	rePunctRun     = regexp.MustCompile(`[.,;:?!]{2,}`)
	reNewlinePunct = regexp.MustCompile(`\n[ \t]*[.,;:]+[ \t]*`)
)

// "period" and "colon" are also clinical nouns, and the word before them often shows which is meant.
// ASR output is punctuated either way, so context is the only signal. The "" entry keeps "colon"
// as a word at the start of a phrase; a dictated period can follow a pause, so it has none.
// ponytail: word lists, so an unlisted modifier still becomes punctuation.
var (
	periodModifiers = map[string]bool{
		"the": true, "a": true, "an": true, "this": true, "that": true, "each": true, "same": true,
		"long": true, "short": true, "brief": true, "time": true,
		"postoperative": true, "postop": true, "preoperative": true, "perioperative": true,
		"recovery": true, "observation": true, "washout": true, "menstrual": true,
		"refractory": true, "incubation": true, "latency": true, "neonatal": true,
	}
	colonModifiers = map[string]bool{
		"": true, "the": true, "a": true, "an": true, "this": true,
		"ascending": true, "descending": true, "transverse": true, "sigmoid": true,
		"right": true, "left": true, "distal": true, "proximal": true, "spastic": true,
	}
)

// replaceSpokenWord converts an ambiguous keyword to its symbol, keeping it where it reads as a noun.
func replaceSpokenWord(text string, re *regexp.Regexp, symbol string, modifiers map[string]bool) string {
	var b strings.Builder
	last := 0
	for _, m := range re.FindAllStringIndex(text, -1) {
		before, after := strings.Fields(text[:m[0]]), strings.Fields(text[m[1]:])
		prev := ""
		if len(before) > 0 {
			prev = strings.ToLower(before[len(before)-1])
		}
		isNoun := (modifiers[prev] && (prev != "" || len(after) > 0)) ||
			(len(after) > 0 && strings.EqualFold(after[0], "of"))
		b.WriteString(text[last:m[0]])
		if isNoun {
			b.WriteString(text[m[0]:m[1]])
		} else {
			b.WriteString(symbol)
		}
		last = m[1]
	}
	b.WriteString(text[last:])
	return b.String()
}

// collapsePunctuation keeps one mark from a run, preferring whichever ends a sentence.
func collapsePunctuation(run string) string {
	if i := strings.IndexAny(run, ".?!"); i >= 0 {
		return run[i : i+1]
	}
	return run[:1]
}

// FormatDictationPhrase formats a completed dictation phrase by converting spoken punctuation
// words to symbols and ensuring proper spacing and casing without forcing artificial semicolons.
func FormatDictationPhrase(rawText string) string {
	return formatDictationPhraseInternal(rawText, false)
}

func formatDictationPhraseInternal(rawText string, isContinuation bool) string {
	text := strings.Trim(rawText, " \t\r\n")
	if text == "" {
		return ""
	}

	// 1. Convert spoken keywords to punctuation symbols
	text = replaceSpokenWord(text, rePeriod, ".", periodModifiers)
	text = replaceSpokenWord(text, reColon, ":", colonModifiers)
	text = reComma.ReplaceAllString(text, ",")
	text = reQuestion.ReplaceAllString(text, "?")
	text = reExclamation.ReplaceAllString(text, "!")
	text = reSemicolon.ReplaceAllString(text, ";")
	text = reNewParagraph.ReplaceAllString(text, "\n\n")
	text = reNewLine.ReplaceAllString(text, "\n")

	// 2. Adjust spacing around punctuation and clean newlines
	text = rePrePunctSpace.ReplaceAllString(text, "$1")
	text = rePunctRun.ReplaceAllStringFunc(text, collapsePunctuation)
	text = reNewlinePunct.ReplaceAllString(text, "\n")
	text = reNewlineSpaces.ReplaceAllString(text, "\n")
	text = reMultiNewlines.ReplaceAllString(text, "\n\n")
	text = reMultiSpaces.ReplaceAllString(text, " ")
	text = strings.Trim(text, " \t")

	if text == "" {
		return ""
	}

	// 3. Automatic punctuation '.' on pause if phrase does not already end with punctuation
	if !reEndsWithPunct.MatchString(text) {
		text += "."
	}

	// 4. Capitalization
	if !isContinuation {
		text = reLeadingChar.ReplaceAllStringFunc(text, func(s string) string {
			r := []rune(s)
			for i, c := range r {
				if unicode.IsLetter(c) {
					r[i] = unicode.ToUpper(c)
					break
				}
			}
			return string(r)
		})
	}

	text = reAfterSentence.ReplaceAllStringFunc(text, func(s string) string {
		r := []rune(s)
		for i := len(r) - 1; i >= 0; i-- {
			if unicode.IsLetter(r[i]) {
				r[i] = unicode.ToUpper(r[i])
				break
			}
		}
		return string(r)
	})

	text = reAfterNewline.ReplaceAllStringFunc(text, func(s string) string {
		r := []rune(s)
		for i := len(r) - 1; i >= 0; i-- {
			if unicode.IsLetter(r[i]) {
				r[i] = unicode.ToUpper(r[i])
				break
			}
		}
		return string(r)
	})

	return text
}

// AppendDictationPhrase appends a newly formatted phrase to the existing draft.
func AppendDictationPhrase(currentText, newPhrase string) string {
	current := strings.TrimRight(currentText, " \t\r\n")
	isContinuation := len(current) > 0 && !reSentenceEnd.MatchString(current)
	formatted := formatDictationPhraseInternal(newPhrase, isContinuation)
	if formatted == "" {
		return current
	}
	if current == "" {
		return formatted
	}
	if strings.HasPrefix(formatted, "\n") {
		return current + formatted
	}
	return current + " " + formatted
}
