package intent

import (
	"sort"
	"strings"
)

// MatchCandidate represents a scored intent match.
type MatchCandidate struct {
	IntentID   string  `json:"intent_id"`
	Score      float64 `json:"score"`
	BestPhrase string  `json:"best_phrase"`
}

// IntentDecision represents the final evaluation of an utterance.
type IntentDecision struct {
	Decision       string            `json:"decision"` // "ACCEPT", "REJECT", "CLARIFY"
	IntentID       string            `json:"intent_id,omitempty"`
	Confidence     float64           `json:"confidence"`
	Margin         float64           `json:"margin"`
	Reason         string            `json:"reason,omitempty"`
	RawTranscript  string            `json:"raw_transcript"`
	Slots          map[string]string `json:"slots,omitempty"`
	MissingSlots   []string          `json:"missing_slots,omitempty"`
	TopCandidates  []MatchCandidate  `json:"top_candidates"`
}

// IntentMatcher performs multi-stage matching against the command catalog.
type IntentMatcher struct {
	catalog             *CommandCatalog
	confidenceThreshold float64
	marginThreshold     float64
}

// NewIntentMatcher creates a new matcher with given thresholds.
func NewIntentMatcher(catalog *CommandCatalog, confThreshold, marginThreshold float64) *IntentMatcher {
	if catalog == nil {
		catalog = DefaultCatalog()
	}
	return &IntentMatcher{
		catalog:             catalog,
		confidenceThreshold: confThreshold,
		marginThreshold:     marginThreshold,
	}
}

// HasNegation checks if the transcript contains explicit negations.
func HasNegation(text string) bool {
	tokens := Tokenize(text)
	negations := map[string]bool{"not": true, "dont": true, "don't": true, "never": true, "no": true, "stop": false}
	for i, t := range tokens {
		if negations[t] {
			// Check if followed by an action verb
			if i+1 < len(tokens) {
				return true
			}
		}
	}
	return false
}

// calculateSimilarity computes a normalized lexical/token overlap score in [0.0, 1.0].
func calculateSimilarity(inputTokens, targetTokens []string, rawInput, rawTarget string) float64 {
	if len(inputTokens) == 0 || len(targetTokens) == 0 {
		return 0.0
	}

	// Exact string match
	if strings.TrimSpace(rawInput) == strings.TrimSpace(rawTarget) {
		return 1.0
	}

	inSet := make(map[string]bool, len(inputTokens))
	for _, t := range inputTokens {
		inSet[t] = true
	}

	var matchCount int
	for _, t := range targetTokens {
		if inSet[t] {
			matchCount++
		}
	}

	// Jaccard token overlap
	unionCount := len(inputTokens) + len(targetTokens) - matchCount
	if unionCount <= 0 {
		return 0.0
	}
	jaccard := float64(matchCount) / float64(unionCount)

	// Recall over target tokens (helps when user says "please open settings now")
	targetCoverage := float64(matchCount) / float64(len(targetTokens))

	// Weighted blend
	score := 0.6*targetCoverage + 0.4*jaccard

	// Bonus for exact substring match
	if strings.Contains(strings.ToLower(rawInput), strings.ToLower(rawTarget)) {
		score += 0.25
		if score > 1.0 {
			score = 1.0
		}
	}

	return score
}

// Match evaluates an utterance against all catalog intents and distractors.
func (m *IntentMatcher) Match(transcript string) IntentDecision {
	cleanText := strings.TrimSpace(transcript)
	if cleanText == "" {
		return IntentDecision{
			Decision:      "REJECT",
			Reason:        "EMPTY_INPUT",
			RawTranscript: transcript,
		}
	}

	// Check negation
	if HasNegation(cleanText) {
		return IntentDecision{
			Decision:      "REJECT",
			Reason:        "NEGATION_DETECTED",
			RawTranscript: transcript,
		}
	}

	inputTokens := Tokenize(cleanText)

	// Step 1: Score distractors
	bestDistractorScore := 0.0
	for _, dist := range m.catalog.Distractors {
		distTokens := Tokenize(dist)
		sim := calculateSimilarity(inputTokens, distTokens, cleanText, dist)
		if sim > bestDistractorScore {
			bestDistractorScore = sim
		}
	}

	// Step 2: Score command candidates
	candidates := make([]MatchCandidate, 0, len(m.catalog.Commands))
	for _, cmd := range m.catalog.Commands {
		bestCmdScore := 0.0
		bestPhrase := ""
		for _, ex := range cmd.Examples {
			exTokens := Tokenize(ex)
			sim := calculateSimilarity(inputTokens, exTokens, cleanText, ex)
			if sim > bestCmdScore {
				bestCmdScore = sim
				bestPhrase = ex
			}
		}
		candidates = append(candidates, MatchCandidate{
			IntentID:   cmd.ID,
			Score:      bestCmdScore,
			BestPhrase: bestPhrase,
		})
	}

	// Sort candidates by score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	if len(candidates) == 0 {
		return IntentDecision{
			Decision:      "REJECT",
			Reason:        "NO_CANDIDATES",
			RawTranscript: transcript,
		}
	}

	top1 := candidates[0]
	top2Score := 0.0
	if len(candidates) > 1 {
		top2Score = candidates[1].Score
	}
	margin := top1.Score - top2Score

	// Step 3: Distractor rejection
	if bestDistractorScore >= top1.Score && bestDistractorScore >= 0.50 {
		return IntentDecision{
			Decision:      "REJECT",
			Confidence:    top1.Score,
			Margin:        margin,
			Reason:        "OUT_OF_DOMAIN_CHATTER",
			RawTranscript: transcript,
			TopCandidates: candidates,
		}
	}

	// Step 4: Confidence & Margin decision boundary
	if top1.Score < m.confidenceThreshold {
		return IntentDecision{
			Decision:      "REJECT",
			Confidence:    top1.Score,
			Margin:        margin,
			Reason:        "LOW_CONFIDENCE",
			RawTranscript: transcript,
			TopCandidates: candidates,
		}
	}

	if margin < m.marginThreshold && len(candidates) > 1 {
		return IntentDecision{
			Decision:      "CLARIFY",
			IntentID:      top1.IntentID,
			Confidence:    top1.Score,
			Margin:        margin,
			Reason:        "AMBIGUOUS_SEPARATION",
			RawTranscript: transcript,
			TopCandidates: candidates,
		}
	}

	// Step 5: Slot Extraction
	slots, missing := m.extractSlots(top1.IntentID, cleanText)
	if len(missing) > 0 {
		return IntentDecision{
			Decision:      "CLARIFY",
			IntentID:      top1.IntentID,
			Confidence:    top1.Score,
			Margin:        margin,
			Reason:        "MISSING_REQUIRED_SLOTS",
			RawTranscript: transcript,
			Slots:         slots,
			MissingSlots:  missing,
			TopCandidates: candidates,
		}
	}

	return IntentDecision{
		Decision:      "ACCEPT",
		IntentID:      top1.IntentID,
		Confidence:    top1.Score,
		Margin:        margin,
		RawTranscript: transcript,
		Slots:         slots,
		TopCandidates: candidates,
	}
}

// extractSlots extracts required slot entities based on the entity catalog.
func (m *IntentMatcher) extractSlots(intentID, text string) (map[string]string, []string) {
	var cmd *CommandEntry
	for _, c := range m.catalog.Commands {
		if c.ID == intentID {
			cmd = &c
			break
		}
	}
	if cmd == nil || len(cmd.RequiredSlots) == 0 {
		return nil, nil
	}

	resolved := make(map[string]string)
	textLower := strings.ToLower(text)

	for _, slotName := range cmd.RequiredSlots {
		slotEntities, exists := m.catalog.Entities[slotName]
		if !exists {
			continue
		}
		for canonicalID, aliases := range slotEntities {
			for _, alias := range aliases {
				if strings.Contains(textLower, strings.ToLower(alias)) {
					resolved[slotName] = canonicalID
					break
				}
			}
			if _, found := resolved[slotName]; found {
				break
			}
		}
	}

	var missing []string
	for _, req := range cmd.RequiredSlots {
		if _, ok := resolved[req]; !ok {
			missing = append(missing, req)
		}
	}

	return resolved, missing
}
