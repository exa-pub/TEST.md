package parser

import (
	"strings"
	"testing"
)

func makeTest(title, onChange string) string {
	return "# " + title + "\n\nSome description.\n\n```yaml\non_change: " + onChange + "\n```\n"
}

func TestParseSingleTest_Basic(t *testing.T) {
	text := makeTest("My Test", "src/**/*")
	fm, tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(fm.Include) != 0 {
		t.Errorf("expected empty include, got %v", fm.Include)
	}
	if len(tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(tests))
	}
	tt := tests[0]
	if tt.Title != "My Test" {
		t.Errorf("expected title 'My Test', got %q", tt.Title)
	}
	if len(tt.OnChange) != 1 || tt.OnChange[0] != "src/**/*" {
		t.Errorf("expected on_change [src/**/*], got %v", tt.OnChange)
	}
	if tt.ExplicitID != "" {
		t.Errorf("expected empty explicit_id, got %q", tt.ExplicitID)
	}
	if tt.Matrix != nil {
		t.Errorf("expected nil matrix, got %v", tt.Matrix)
	}
	if !strings.Contains(tt.Description, "Some description.") {
		t.Errorf("expected description to contain 'Some description.', got %q", tt.Description)
	}
}

func TestParseSingleTest_SourceLineNoFrontmatter(t *testing.T) {
	text := makeTest("My Test", "src/**/*")
	_, tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if tests[0].SourceLine != 1 {
		t.Errorf("expected source_line 1, got %d", tests[0].SourceLine)
	}
}

func TestParseSingleTest_OnChangeAsList(t *testing.T) {
	text := "# T\n\n```yaml\non_change:\n  - a.txt\n  - b.txt\n```\n"
	_, tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(tests[0].OnChange) != 2 || tests[0].OnChange[0] != "a.txt" || tests[0].OnChange[1] != "b.txt" {
		t.Errorf("expected [a.txt b.txt], got %v", tests[0].OnChange)
	}
}

func TestParseSingleTest_OnChangeAsString(t *testing.T) {
	text := "# T\n\n```yaml\non_change: foo.py\n```\n"
	_, tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(tests[0].OnChange) != 1 || tests[0].OnChange[0] != "foo.py" {
		t.Errorf("expected [foo.py], got %v", tests[0].OnChange)
	}
}

func TestParseSingleTest_ExplicitID(t *testing.T) {
	text := "# T\n\n```yaml\nid: my-id\non_change: x\n```\n"
	_, tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if tests[0].ExplicitID != "my-id" {
		t.Errorf("expected explicit_id 'my-id', got %q", tests[0].ExplicitID)
	}
}

func TestParseSingleTest_MatrixParsed(t *testing.T) {
	text := "# T\n\n```yaml\non_change: $svc/**/*\nmatrix:\n  - match: $svc/\n```\n"
	_, tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(tests[0].Matrix) != 1 {
		t.Fatalf("expected 1 matrix entry, got %d", len(tests[0].Matrix))
	}
	if len(tests[0].Matrix[0].Match) != 1 || tests[0].Matrix[0].Match[0] != "$svc/" {
		t.Errorf("expected match ['$svc/'], got %v", tests[0].Matrix[0].Match)
	}
}

func TestParseMultipleTests_TwoTests(t *testing.T) {
	text := makeTest("First", "a.txt") + "\n" + makeTest("Second", "b.txt")
	_, tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(tests) != 2 {
		t.Fatalf("expected 2 tests, got %d", len(tests))
	}
	if tests[0].Title != "First" {
		t.Errorf("expected first test title 'First', got %q", tests[0].Title)
	}
	if tests[1].Title != "Second" {
		t.Errorf("expected second test title 'Second', got %q", tests[1].Title)
	}
}

func TestFrontmatter_Extracted(t *testing.T) {
	text := "---\ninclude:\n  - other.md\n---\n" + makeTest("My Test", "src/**/*")
	fm, tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(fm.Include) != 1 || fm.Include[0] != "other.md" {
		t.Errorf("expected include [other.md], got %v", fm.Include)
	}
	if len(tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(tests))
	}
}

func TestFrontmatter_SourceLineWithFrontmatter(t *testing.T) {
	frontmatter := "---\ninclude:\n  - other.md\n---\n"
	text := frontmatter + makeTest("My Test", "src/**/*")
	_, tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	// Frontmatter is 4 lines (---, include:, - other.md, ---), so offset is 4
	if tests[0].SourceLine != 1+4 {
		t.Errorf("expected source_line 5, got %d", tests[0].SourceLine)
	}
}

func TestErrors_MissingYamlBlock(t *testing.T) {
	text := "# T\n\nNo yaml here.\n"
	_, _, err := Parse(text, "TEST.md")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "missing yaml config block") {
		t.Errorf("expected 'missing yaml config block' in error, got %q", err.Error())
	}
}

func TestErrors_MissingOnChange(t *testing.T) {
	text := "# T\n\n```yaml\nid: foo\n```\n"
	_, _, err := Parse(text, "TEST.md")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "missing on_change") {
		t.Errorf("expected 'missing on_change' in error, got %q", err.Error())
	}
}

func TestDescriptionContent_YamlBlockExcluded(t *testing.T) {
	text := "# T\n\nBefore yaml.\n\n```yaml\non_change: x\n```\n\nAfter yaml.\n"
	_, tests, err := Parse(text, "TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	desc := tests[0].Description
	if !strings.Contains(desc, "Before yaml.") {
		t.Errorf("expected 'Before yaml.' in description, got %q", desc)
	}
	if !strings.Contains(desc, "After yaml.") {
		t.Errorf("expected 'After yaml.' in description, got %q", desc)
	}
	if strings.Contains(desc, "on_change") {
		t.Errorf("description should not contain 'on_change', got %q", desc)
	}
}
