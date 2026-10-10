package server

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	rePeriod       = regexp.MustCompile(`(?i)\b(period|full stop)\b`)
	reComma        = regexp.MustCompile(`(?i)\bcomma\b`)
	reQuestion     = regexp.MustCompile(`(?i)\bquestion mark\b`)
	reExclamation  = regexp.MustCompile(`(?i)\bexclamation (?:mark|point)\b`)
	reSemicolon    = regexp.MustCompile(`(?i)\bsemicolon\b`)
	reColon        = regexp.MustCompile(`(?i)\bcolon\b`)
	reNewParagraph = regexp.MustCompile(`(?i)\bnew paragraph\b`)
	reNewLine      = regexp.MustCompile(`(?i)\bnew line\b`)

	rePrePunctSpace  = regexp.MustCompile(`[ \t]+([.,;:?!])`)
	rePostPunctSpace = regexp.MustCompile(`([.,;:?!])([a-zA-Z0-9])`)
	reNewlineSpaces  = regexp.MustCompile(`[ \t]*\n[ \t]*`)
	reMultiNewlines  = regexp.MustCompile(`\n{3,}`)
	reMultiSpaces    = regexp.MustCompile(`[ \t]{2,}`)
	reEndsWithPunct  = regexp.MustCompile(`[.;,!?:\n]$`)
	reLeadingChar    = regexp.MustCompile(`^(\s*)([a-z])`)
	reAfterSentence  = regexp.MustCompile(`([.?!]\s+)([a-z])`)
	reAfterNewline   = regexp.MustCompile(`(\n+\s*)([a-z])`)
)

var reSentenceEnd = regexp.MustCompile(`[.?!:\n]$`)

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
	text = rePeriod.ReplaceAllString(text, ".")
	text = reComma.ReplaceAllString(text, ",")
	text = reQuestion.ReplaceAllString(text, "?")
	text = reExclamation.ReplaceAllString(text, "!")
	text = reSemicolon.ReplaceAllString(text, ";")
	text = reColon.ReplaceAllString(text, ":")
	text = reNewParagraph.ReplaceAllString(text, "\n\n")
	text = reNewLine.ReplaceAllString(text, "\n")

	// 2. Adjust spacing around punctuation and clean newlines
	text = rePrePunctSpace.ReplaceAllString(text, "$1")
	text = rePostPunctSpace.ReplaceAllString(text, "$1 $2")
	text = reNewlineSpaces.ReplaceAllString(text, "\n")
	text = reMultiNewlines.ReplaceAllString(text, "\n\n")
	text = reMultiSpaces.ReplaceAllString(text, " ")
	text = strings.Trim(text, " \t")

	if text == "" {
		return ""
	}

	// 3. Capitalization
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
