package state

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/testmd/testmd/internal/models"
)

// Load reads state from a lock file. Returns empty state if file does not exist.
func Load(lockFile string) (*models.State, error) {
	data, err := os.ReadFile(lockFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &models.State{Version: 1, Tests: map[string]*models.TestRecord{}}, nil
		}
		return nil, err
	}

	var st models.State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	if st.Tests == nil {
		st.Tests = map[string]*models.TestRecord{}
	}
	return &st, nil
}

// Save writes state as formatted JSON to a lock file.
// If state has no tests, the lock file is deleted.
func Save(lockFile string, st *models.State) error {
	if len(st.Tests) == 0 {
		err := os.Remove(lockFile)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	body, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(lockFile, body, 0644)
}
