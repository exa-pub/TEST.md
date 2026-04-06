package state

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"

	"github.com/testmd/testmd/internal/models"
)

var stateRE = regexp.MustCompile(`(?s)<!-- State\n` + "```testmd\n" + `(.*?)` + "```\n" + `-->` + "\n?")

// Load extracts state from the <!-- State --> block in a TEST.md file.
func Load(testFile string) (*models.State, error) {
	data, err := os.ReadFile(testFile)
	if err != nil {
		return &models.State{Version: 1, Tests: map[string]*models.TestRecord{}}, nil
	}

	m := stateRE.FindSubmatch(data)
	if m == nil {
		return &models.State{Version: 1, Tests: map[string]*models.TestRecord{}}, nil
	}

	var st models.State
	if err := json.Unmarshal(m[1], &st); err != nil {
		return nil, err
	}
	if st.Tests == nil {
		st.Tests = map[string]*models.TestRecord{}
	}
	return &st, nil
}

// Save writes state as formatted JSON into the <!-- State --> block.
func Save(testFile string, st *models.State) error {
	data, err := os.ReadFile(testFile)
	if err != nil {
		return err
	}

	body, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	block := "<!-- State\n```testmd\n" + string(body) + "\n```\n-->\n"

	text := string(data)
	loc := stateRE.FindStringIndex(text)
	if loc != nil {
		text = text[:loc[0]] + block
	} else {
		text = strings.TrimRight(text, "\n") + "\n\n" + block
	}

	return os.WriteFile(testFile, []byte(text), 0644)
}

// StripBlock removes the state block from a TEST.md file if present.
func StripBlock(testFile string) error {
	data, err := os.ReadFile(testFile)
	if err != nil {
		return nil // file doesn't exist — nothing to strip
	}

	text := string(data)
	loc := stateRE.FindStringIndex(text)
	if loc == nil {
		return nil
	}

	text = strings.TrimRight(text[:loc[0]], "\n") + "\n"
	return os.WriteFile(testFile, []byte(text), 0644)
}
