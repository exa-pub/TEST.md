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
	t.Run("no_state_block", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "TEST.md")
		os.WriteFile(f, []byte("# Test\n\nSome content.\n"), 0644)

		st, err := Load(f)
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

	t.Run("with_state_block", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "TEST.md")
		data := models.State{
			Version: 1,
			Tests: map[string]*models.TestRecord{
				"abc-def": {Status: "resolved"},
			},
		}
		body, _ := json.Marshal(data)
		block := "<!-- State\n```testmd\n" + string(body) + "\n```\n-->\n"
		os.WriteFile(f, []byte("# Test\n\n"+block), 0644)

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
}

func TestSave(t *testing.T) {
	t.Run("appends_to_file_without_block", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "TEST.md")
		os.WriteFile(f, []byte("# Test\n\nContent here."), 0644)

		st := &models.State{Version: 1, Tests: map[string]*models.TestRecord{}}
		err := Save(f, st)
		if err != nil {
			t.Fatal(err)
		}
		text, _ := os.ReadFile(f)
		s := string(text)
		if !strings.Contains(s, "<!-- State") {
			t.Error("expected '<!-- State' in output")
		}
		if !strings.Contains(s, "```testmd") {
			t.Error("expected '```testmd' in output")
		}
		if !strings.Contains(s, "```\n-->") {
			t.Error("expected '```\\n-->' in output")
		}
	})

	t.Run("replaces_existing_block", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "TEST.md")
		oldState := &models.State{
			Version: 1,
			Tests:   map[string]*models.TestRecord{"old": {}},
		}
		oldBody, _ := json.Marshal(oldState)
		block := "<!-- State\n```testmd\n" + string(oldBody) + "\n```\n-->\n"
		os.WriteFile(f, []byte("# Test\n\n"+block), 0644)

		newState := &models.State{
			Version: 1,
			Tests:   map[string]*models.TestRecord{"new": {}},
		}
		err := Save(f, newState)
		if err != nil {
			t.Fatal(err)
		}
		text, _ := os.ReadFile(f)
		s := string(text)
		if !strings.Contains(s, `"new"`) {
			t.Error("expected '\"new\"' in output")
		}
		if strings.Contains(s, `"old"`) {
			t.Error("should not contain '\"old\"' in output")
		}
		if strings.Count(s, "<!-- State") != 1 {
			t.Error("expected exactly one state block")
		}
	})

	t.Run("json_indent", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "TEST.md")
		os.WriteFile(f, []byte("# Test\n"), 0644)

		st := &models.State{Version: 1, Tests: map[string]*models.TestRecord{}}
		Save(f, st)
		text, _ := os.ReadFile(f)
		if !strings.Contains(string(text), `  "version": 1`) {
			t.Error("expected indented JSON")
		}
	})

	t.Run("block_format", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "TEST.md")
		os.WriteFile(f, []byte("# Test\n"), 0644)

		st := &models.State{Version: 1, Tests: map[string]*models.TestRecord{}}
		Save(f, st)
		text, _ := os.ReadFile(f)
		s := string(text)
		if !strings.Contains(s, "<!-- State\n```testmd\n") {
			t.Error("expected block start format")
		}
		if !strings.Contains(s, "\n```\n-->\n") {
			t.Error("expected block end format")
		}
	})
}

func TestStripBlock(t *testing.T) {
	t.Run("removes_block", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "TEST.md")
		block := "<!-- State\n```testmd\n{\"version\":1}\n```\n-->\n"
		os.WriteFile(f, []byte("# Test\n\n"+block), 0644)

		err := StripBlock(f)
		if err != nil {
			t.Fatal(err)
		}
		text, _ := os.ReadFile(f)
		s := string(text)
		if strings.Contains(s, "<!-- State") {
			t.Error("state block should be removed")
		}
		if !strings.Contains(s, "# Test") {
			t.Error("should preserve '# Test'")
		}
	})

	t.Run("no_block_is_noop", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "TEST.md")
		os.WriteFile(f, []byte("# Test\n\nNo state here.\n"), 0644)

		err := StripBlock(f)
		if err != nil {
			t.Fatal(err)
		}
		text, _ := os.ReadFile(f)
		s := string(text)
		if !strings.Contains(s, "# Test") {
			t.Error("should preserve '# Test'")
		}
		if !strings.Contains(s, "No state here.") {
			t.Error("should preserve content")
		}
	})
}
