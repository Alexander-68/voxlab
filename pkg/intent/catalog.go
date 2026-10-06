package intent

import (
	"encoding/json"
	"os"
	"strings"
)

// CommandEntry represents one canonical command in the catalog.
type CommandEntry struct {
	ID             string              `json:"id"`
	Examples       []string            `json:"examples"`
	ActionAliases  []string            `json:"action_aliases,omitempty"`
	RequiredSlots  []string            `json:"required_slots,omitempty"`
	Preconditions  []string            `json:"preconditions,omitempty"`
	Confirmation   string              `json:"confirmation,omitempty"` // "none", "physical", "voice"
}

// CommandCatalog defines the full command schema with commands, distractors, and entities.
type CommandCatalog struct {
	SchemaVersion int                            `json:"schema_version"`
	Commands      []CommandEntry                 `json:"commands"`
	Distractors   []string                       `json:"distractors"`
	Entities      map[string]map[string][]string `json:"entities,omitempty"`
}

// LoadCatalog loads a CommandCatalog from JSON file.
func LoadCatalog(filePath string) (*CommandCatalog, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return DefaultCatalog(), nil
	}
	var catalog CommandCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}
	return &catalog, nil
}

// DefaultCatalog provides a comprehensive built-in catalog if data/commands.json is missing.
func DefaultCatalog() *CommandCatalog {
	return &CommandCatalog{
		SchemaVersion: 1,
		Commands: []CommandEntry{
			{
				ID:            "NAV_SETTINGS",
				Examples:      []string{"open settings", "go to preferences", "show configuration", "system setup"},
				ActionAliases: []string{"open", "show", "go"},
			},
			{
				ID:            "START_RECORDING",
				Examples:      []string{"start recording", "begin recording", "record now", "start capture"},
				ActionAliases: []string{"start", "begin", "record"},
				Preconditions: []string{"recording_state == idle"},
			},
			{
				ID:            "STOP_RECORDING",
				Examples:      []string{"stop recording", "finish this recording", "end recording", "stop capture"},
				ActionAliases: []string{"stop", "finish", "end"},
				Confirmation:  "physical",
				Preconditions: []string{"recording_state == active"},
			},
			{
				ID:            "DISPLAY_DARK",
				Examples:      []string{"dim screen", "lower brightness", "make it darker", "turn down lights", "dark mode"},
				ActionAliases: []string{"dim", "dark", "lower"},
			},
			{
				ID:            "MEDIA_NEXT",
				Examples:      []string{"next song", "skip track", "play next", "skip forward"},
				ActionAliases: []string{"next", "skip"},
			},
			{
				ID:            "VOLUME_UP",
				Examples:      []string{"volume up", "increase volume", "make it louder", "turn up sound"},
				ActionAliases: []string{"increase", "louder", "up"},
			},
			{
				ID:            "VOLUME_DOWN",
				Examples:      []string{"volume down", "decrease volume", "make it quieter", "turn down sound"},
				ActionAliases: []string{"decrease", "quieter", "down"},
			},
			{
				ID:            "ROUTE_VIDEO",
				Examples:      []string{"show endoscope on the main monitor", "send camera two to monitor two", "route camera one to second screen"},
				ActionAliases: []string{"show", "send", "route", "switch"},
				RequiredSlots: []string{"source", "destination"},
			},
		},
		Distractors: []string{
			"how are you doing",
			"what are you eating",
			"see you later",
			"let us go outside",
			"what time is lunch",
			"great weather today",
			"did you hear that",
			"what did he say",
			"tell me a joke",
		},
		Entities: map[string]map[string][]string{
			"source": {
				"endoscope": {"endoscope", "scope camera", "scope"},
				"camera_1":  {"camera one", "camera 1", "cam one"},
				"camera_2":  {"camera two", "camera 2", "cam two"},
			},
			"destination": {
				"main_monitor": {"main monitor", "main screen", "first monitor", "display one"},
				"monitor_2":    {"monitor two", "second monitor", "second screen", "display two"},
			},
		},
	}
}

// Tokenize converts text to lowercase words stripped of punctuation.
func Tokenize(text string) []string {
	text = strings.ToLower(text)
	var b strings.Builder
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	fields := strings.Fields(b.String())
	return fields
}
