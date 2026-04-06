package patterns

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindLabelVars(t *testing.T) {
	t.Run("single_var", func(t *testing.T) {
		result := FindLabelVars("$svc/src/**")
		if len(result) != 1 || result[0] != "svc" {
			t.Errorf("expected [svc], got %v", result)
		}
	})

	t.Run("multiple_vars", func(t *testing.T) {
		result := FindLabelVars("$org/$repo/file")
		if len(result) != 2 || result[0] != "org" || result[1] != "repo" {
			t.Errorf("expected [org repo], got %v", result)
		}
	})

	t.Run("no_vars", func(t *testing.T) {
		result := FindLabelVars("src/**/*")
		if len(result) != 0 {
			t.Errorf("expected [], got %v", result)
		}
	})

	t.Run("underscore_var", func(t *testing.T) {
		result := FindLabelVars("$my_var/x")
		if len(result) != 1 || result[0] != "my_var" {
			t.Errorf("expected [my_var], got %v", result)
		}
	})
}

func TestEnumerateLabels(t *testing.T) {
	t.Run("no_vars_returns_empty_dict", func(t *testing.T) {
		result := EnumerateLabels("/tmp", []string{"src/**/*"}, nil)
		if len(result) != 1 || len(result[0]) != 0 {
			t.Errorf("expected [{}], got %v", result)
		}
	})

	t.Run("discovers_dirs", func(t *testing.T) {
		tmp := t.TempDir()
		os.MkdirAll(filepath.Join(tmp, "alpha", "src"), 0755)
		os.MkdirAll(filepath.Join(tmp, "beta", "src"), 0755)
		os.WriteFile(filepath.Join(tmp, "alpha", "src", "a.py"), []byte("a"), 0644)
		os.WriteFile(filepath.Join(tmp, "beta", "src", "b.py"), []byte("b"), 0644)

		result := EnumerateLabels(tmp, []string{"$svc/src"}, nil)
		if len(result) != 2 {
			t.Fatalf("expected 2 results, got %d: %v", len(result), result)
		}
		found := map[string]bool{}
		for _, r := range result {
			found[r["svc"]] = true
		}
		if !found["alpha"] || !found["beta"] {
			t.Errorf("expected alpha and beta, got %v", result)
		}
	})

	t.Run("hidden_dirs_excluded", func(t *testing.T) {
		tmp := t.TempDir()
		os.MkdirAll(filepath.Join(tmp, ".hidden", "src"), 0755)
		os.MkdirAll(filepath.Join(tmp, "visible", "src"), 0755)

		result := EnumerateLabels(tmp, []string{"$svc/src"}, nil)
		if len(result) != 1 {
			t.Fatalf("expected 1 result, got %d: %v", len(result), result)
		}
		if result[0]["svc"] != "visible" {
			t.Errorf("expected svc=visible, got %v", result[0])
		}
	})

	t.Run("deduplicates", func(t *testing.T) {
		tmp := t.TempDir()
		os.MkdirAll(filepath.Join(tmp, "foo", "a"), 0755)
		os.MkdirAll(filepath.Join(tmp, "foo", "b"), 0755)

		result := EnumerateLabels(tmp, []string{"$svc/a", "$svc/b"}, nil)
		count := 0
		for _, r := range result {
			if r["svc"] == "foo" {
				count++
			}
		}
		if count != 1 {
			t.Errorf("expected foo to appear once, got %d times in %v", count, result)
		}
	})
}

func TestExpandMatrix(t *testing.T) {
	t.Run("const_only", func(t *testing.T) {
		matrix := []ExpandableEntry{
			{Const: map[string][]string{"env": {"dev", "prod"}, "region": {"us", "eu"}}},
		}
		result := ExpandMatrix("/tmp", matrix, nil)
		if len(result) != 4 {
			t.Fatalf("expected 4 combos, got %d: %v", len(result), result)
		}
		assertContainsLabels(t, result, map[string]string{"env": "dev", "region": "us"})
		assertContainsLabels(t, result, map[string]string{"env": "prod", "region": "eu"})
	})

	t.Run("match_only", func(t *testing.T) {
		tmp := t.TempDir()
		os.Mkdir(filepath.Join(tmp, "svcA"), 0755)
		os.Mkdir(filepath.Join(tmp, "svcB"), 0755)

		matrix := []ExpandableEntry{
			{Match: []string{"$svc/"}},
		}
		result := ExpandMatrix(tmp, matrix, nil)
		assertContainsLabels(t, result, map[string]string{"svc": "svcA"})
		assertContainsLabels(t, result, map[string]string{"svc": "svcB"})
	})

	t.Run("match_plus_const", func(t *testing.T) {
		tmp := t.TempDir()
		os.Mkdir(filepath.Join(tmp, "web"), 0755)

		matrix := []ExpandableEntry{
			{Match: []string{"$svc/"}, Const: map[string][]string{"env": {"dev", "prod"}}},
		}
		result := ExpandMatrix(tmp, matrix, nil)
		if len(result) != 2 {
			t.Fatalf("expected 2 combos, got %d: %v", len(result), result)
		}
		assertContainsLabels(t, result, map[string]string{"svc": "web", "env": "dev"})
		assertContainsLabels(t, result, map[string]string{"svc": "web", "env": "prod"})
	})

	t.Run("union_of_entries", func(t *testing.T) {
		matrix := []ExpandableEntry{
			{Const: map[string][]string{"x": {"1"}}},
			{Const: map[string][]string{"x": {"2"}}},
		}
		result := ExpandMatrix("/tmp", matrix, nil)
		if len(result) != 2 {
			t.Fatalf("expected 2 combos, got %d: %v", len(result), result)
		}
		assertContainsLabels(t, result, map[string]string{"x": "1"})
		assertContainsLabels(t, result, map[string]string{"x": "2"})
	})

	t.Run("empty_matrix_returns_empty_dict", func(t *testing.T) {
		result := ExpandMatrix("/tmp", []ExpandableEntry{}, nil)
		if len(result) != 1 || len(result[0]) != 0 {
			t.Errorf("expected [{}], got %v", result)
		}
	})

	t.Run("const_scalar_value", func(t *testing.T) {
		// In Go, the const map always has []string values, so a scalar is represented as a single-element slice
		matrix := []ExpandableEntry{
			{Const: map[string][]string{"env": {"prod"}}},
		}
		result := ExpandMatrix("/tmp", matrix, nil)
		if len(result) != 1 {
			t.Fatalf("expected 1 combo, got %d: %v", len(result), result)
		}
		if result[0]["env"] != "prod" {
			t.Errorf("expected env=prod, got %v", result[0])
		}
	})
}

func TestResolveFiles(t *testing.T) {
	t.Run("substitutes_labels_and_globs", func(t *testing.T) {
		tmp := t.TempDir()
		os.MkdirAll(filepath.Join(tmp, "web", "src"), 0755)
		os.WriteFile(filepath.Join(tmp, "web", "src", "main.py"), []byte("x"), 0644)

		result, err := ResolveFiles(tmp, "$svc/src/*.py", map[string]string{"svc": "web"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 || result[0] != "web/src/main.py" {
			t.Errorf("expected [web/src/main.py], got %v", result)
		}
	})

	t.Run("double_star_normalization", func(t *testing.T) {
		tmp := t.TempDir()
		os.Mkdir(filepath.Join(tmp, "lib"), 0755)
		os.WriteFile(filepath.Join(tmp, "lib", "a.py"), []byte("a"), 0644)

		result, err := ResolveFiles(tmp, "lib/**", map[string]string{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, f := range result {
			if f == "lib/a.py" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected lib/a.py in results, got %v", result)
		}
	})

	t.Run("dot_slash_stripped", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "foo.txt"), []byte("hello"), 0644)

		result, err := ResolveFiles(tmp, "./foo.txt", map[string]string{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 || result[0] != "foo.txt" {
			t.Errorf("expected [foo.txt], got %v", result)
		}
	})

	t.Run("no_matches", func(t *testing.T) {
		tmp := t.TempDir()
		result, err := ResolveFiles(tmp, "nonexistent/*.py", map[string]string{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty, got %v", result)
		}
	})
}

func assertContainsLabels(t *testing.T, list []map[string]string, want map[string]string) {
	t.Helper()
	for _, m := range list {
		if len(m) != len(want) {
			continue
		}
		match := true
		for k, v := range want {
			if m[k] != v {
				match = false
				break
			}
		}
		if match {
			return
		}
	}
	t.Errorf("expected %v to be in %v", want, list)
}
