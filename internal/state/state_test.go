package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/testmd/testmd/internal/models"
)

func TestLoad(t *testing.T) {
	t.Run("file_not_found", func(t *testing.T) {
		st, err := Load(filepath.Join(t.TempDir(), "TEST.md.lock"))
		if err != nil {
			t.Fatal(err)
		}
		if st.Version != 1 {
			t.Errorf("expected version 1, got %d", st.Version)
		}
		if len(st.Tests) != 0 {
			t.Errorf("expected empty tests, got %v", st.Tests)
		}
	})

	t.Run("reads_json", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "TEST.md.lock")
		data := models.State{
			Version: 1,
			Tests: map[string]*models.TestRecord{
				"abc-def": {Status: "resolved"},
			},
		}
		body, _ := json.MarshalIndent(data, "", "  ")
		os.WriteFile(f, body, 0644)

		st, err := Load(f)
		if err != nil {
			t.Fatal(err)
		}
		if st.Tests["abc-def"] == nil {
			t.Fatal("expected test record for abc-def")
		}
		if st.Tests["abc-def"].Status != "resolved" {
			t.Errorf("expected status 'resolved', got %q", st.Tests["abc-def"].Status)
		}
	})

	t.Run("nil_tests_map", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "TEST.md.lock")
		os.WriteFile(f, []byte(`{"version":1}`), 0644)

		st, err := Load(f)
		if err != nil {
			t.Fatal(err)
		}
		if st.Tests == nil {
			t.Error("expected non-nil tests map")
		}
	})
}

func TestSave(t *testing.T) {
	t.Run("writes_json", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "TEST.md.lock")

		st := &models.State{
			Version: 1,
			Tests: map[string]*models.TestRecord{
				"abc-def": {Status: "resolved"},
			},
		}
		if err := Save(f, st); err != nil {
			t.Fatal(err)
		}
		text, _ := os.ReadFile(f)
		s := string(text)
		if !strings.Contains(s, `"abc-def"`) {
			t.Error("expected test id in output")
		}
		if !strings.Contains(s, `  "version": 1`) {
			t.Error("expected indented JSON")
		}
		if !strings.HasSuffix(s, "\n") {
			t.Error("expected trailing newline")
		}
	})

	t.Run("overwrites_existing", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "TEST.md.lock")

		old := &models.State{Version: 1, Tests: map[string]*models.TestRecord{"old": {Status: "resolved"}}}
		Save(f, old)

		new := &models.State{Version: 1, Tests: map[string]*models.TestRecord{"new": {Status: "pending"}}}
		Save(f, new)

		text, _ := os.ReadFile(f)
		s := string(text)
		if strings.Contains(s, `"old"`) {
			t.Error("should not contain old record")
		}
		if !strings.Contains(s, `"new"`) {
			t.Error("expected new record")
		}
	})

	t.Run("empty_state_deletes_file", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "TEST.md.lock")

		st := &models.State{Version: 1, Tests: map[string]*models.TestRecord{"x": {Status: "resolved"}}}
		Save(f, st)
		if _, err := os.Stat(f); err != nil {
			t.Fatal("file should exist after save with tests")
		}

		empty := &models.State{Version: 1, Tests: map[string]*models.TestRecord{}}
		if err := Save(f, empty); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Error("file should be deleted when state is empty")
		}
	})

	t.Run("empty_state_no_file_is_noop", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "TEST.md.lock")

		empty := &models.State{Version: 1, Tests: map[string]*models.TestRecord{}}
		if err := Save(f, empty); err != nil {
			t.Fatal(err)
		}
	})
}
