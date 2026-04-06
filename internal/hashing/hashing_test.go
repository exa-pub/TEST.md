package hashing

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestHashFile(t *testing.T) {
	t.Run("consistent", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "f.txt"), []byte("hello"), 0644)

		h1, err := HashFile(tmp, "f.txt")
		if err != nil {
			t.Fatal(err)
		}
		h2, err := HashFile(tmp, "f.txt")
		if err != nil {
			t.Fatal(err)
		}
		if h1 != h2 {
			t.Errorf("hashes should be consistent: %s != %s", h1, h2)
		}
	})

	t.Run("includes_path_in_hash", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "a.txt"), []byte("same"), 0644)
		os.WriteFile(filepath.Join(tmp, "b.txt"), []byte("same"), 0644)

		ha, _ := HashFile(tmp, "a.txt")
		hb, _ := HashFile(tmp, "b.txt")
		if ha == hb {
			t.Errorf("different paths with same content should produce different hashes")
		}
	})

	t.Run("known_value", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "x.txt"), []byte("data"), 0644)

		h := sha256.New()
		h.Write([]byte("x.txt"))
		h.Write([]byte{0})
		h.Write([]byte("data"))
		expected := fmt.Sprintf("%x", h.Sum(nil))

		got, err := HashFile(tmp, "x.txt")
		if err != nil {
			t.Fatal(err)
		}
		if got != expected {
			t.Errorf("expected %s, got %s", expected, got)
		}
	})
}

func TestHashFiles(t *testing.T) {
	t.Run("empty_list", func(t *testing.T) {
		tmp := t.TempDir()
		contentHash, fileHashes, err := HashFiles(tmp, []string{})
		if err != nil {
			t.Fatal(err)
		}
		if len(fileHashes) != 0 {
			t.Errorf("expected empty file_hashes, got %v", fileHashes)
		}
		expected := fmt.Sprintf("%x", sha256.Sum256([]byte("")))
		if contentHash != expected {
			t.Errorf("expected hash of empty string %s, got %s", expected, contentHash)
		}
	})

	t.Run("multiple_files", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "a.txt"), []byte("aaa"), 0644)
		os.WriteFile(filepath.Join(tmp, "b.txt"), []byte("bbb"), 0644)

		contentHash, fileHashes, err := HashFiles(tmp, []string{"a.txt", "b.txt"})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := fileHashes["a.txt"]; !ok {
			t.Error("missing a.txt in file_hashes")
		}
		if _, ok := fileHashes["b.txt"]; !ok {
			t.Error("missing b.txt in file_hashes")
		}
		if len(contentHash) != 64 {
			t.Errorf("expected 64-char sha256 hex, got %d chars", len(contentHash))
		}
	})

	t.Run("content_hash_changes_with_file_content", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "f.txt"), []byte("v1"), 0644)
		h1, _, err := HashFiles(tmp, []string{"f.txt"})
		if err != nil {
			t.Fatal(err)
		}

		os.WriteFile(filepath.Join(tmp, "f.txt"), []byte("v2"), 0644)
		h2, _, err := HashFiles(tmp, []string{"f.txt"})
		if err != nil {
			t.Fatal(err)
		}
		if h1 == h2 {
			t.Error("content hash should change when file content changes")
		}
	})
}

func TestMakeID(t *testing.T) {
	t.Run("with_title_only", func(t *testing.T) {
		tid := MakeID("My Test", "", map[string]string{})
		if len(tid) != 13 {
			t.Errorf("expected length 13 (6+1+6), got %d: %q", len(tid), tid)
		}
		if tid[6] != '-' {
			t.Errorf("expected dash at position 6, got %q", tid)
		}
	})

	t.Run("explicit_id_overrides_title", func(t *testing.T) {
		id1 := MakeID("Title A", "custom", map[string]string{})
		id2 := MakeID("Title B", "custom", map[string]string{})
		parts1 := splitID(id1)
		parts2 := splitID(id2)
		if parts1[0] != parts2[0] {
			t.Errorf("same explicit_id should produce same first part: %s vs %s", parts1[0], parts2[0])
		}
	})

	t.Run("labels_affect_second_part", func(t *testing.T) {
		id1 := MakeID("T", "", map[string]string{})
		id2 := MakeID("T", "", map[string]string{"svc": "web"})
		parts1 := splitID(id1)
		parts2 := splitID(id2)
		if parts1[0] != parts2[0] {
			t.Errorf("same title should produce same first part: %s vs %s", parts1[0], parts2[0])
		}
		if parts1[1] == parts2[1] {
			t.Errorf("different labels should produce different second part: %s vs %s", parts1[1], parts2[1])
		}
	})

	t.Run("different_titles_different_first_part", func(t *testing.T) {
		id1 := MakeID("Alpha", "", map[string]string{})
		id2 := MakeID("Beta", "", map[string]string{})
		parts1 := splitID(id1)
		parts2 := splitID(id2)
		if parts1[0] == parts2[0] {
			t.Errorf("different titles should produce different first parts: %s vs %s", parts1[0], parts2[0])
		}
	})

	t.Run("deterministic", func(t *testing.T) {
		id1 := MakeID("X", "", map[string]string{"a": "1"})
		id2 := MakeID("X", "", map[string]string{"a": "1"})
		if id1 != id2 {
			t.Errorf("should be deterministic: %s != %s", id1, id2)
		}
	})
}

func splitID(id string) [2]string {
	for i, c := range id {
		if c == '-' {
			return [2]string{id[:i], id[i+1:]}
		}
	}
	return [2]string{id, ""}
}
