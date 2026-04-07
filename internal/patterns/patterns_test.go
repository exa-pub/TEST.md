package patterns

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/testmd/testmd/internal/models"
)

func TestDiscoverValues(t *testing.T) {
	t.Run("explicit_list", func(t *testing.T) {
		result := DiscoverValues("/tmp", models.EachSource{Values: []string{"b", "a", "c"}}, nil)
		if len(result) != 3 || result[0] != "a" || result[1] != "b" || result[2] != "c" {
			t.Errorf("expected sorted [a b c], got %v", result)
		}
	})

	t.Run("glob_dirs", func(t *testing.T) {
		tmp := t.TempDir()
		os.MkdirAll(filepath.Join(tmp, "services", "auth"), 0755)
		os.MkdirAll(filepath.Join(tmp, "services", "billing"), 0755)
		os.WriteFile(filepath.Join(tmp, "services", "README.md"), []byte("x"), 0644)

		result := DiscoverValues(tmp, models.EachSource{Glob: "./services/*/"}, nil)
		if len(result) != 2 {
			t.Fatalf("expected 2 dirs, got %d: %v", len(result), result)
		}
		if result[0] != "auth" || result[1] != "billing" {
			t.Errorf("expected [auth billing], got %v", result)
		}
	})

	t.Run("glob_files_strip_ext", func(t *testing.T) {
		tmp := t.TempDir()
		os.MkdirAll(filepath.Join(tmp, "configs"), 0755)
		os.WriteFile(filepath.Join(tmp, "configs", "app.yaml"), []byte("x"), 0644)
		os.WriteFile(filepath.Join(tmp, "configs", "db.yaml"), []byte("x"), 0644)

		result := DiscoverValues(tmp, models.EachSource{Glob: "./configs/*.yaml"}, nil)
		if len(result) != 2 || result[0] != "app" || result[1] != "db" {
			t.Errorf("expected [app db], got %v", result)
		}
	})

	t.Run("glob_files_no_strip", func(t *testing.T) {
		tmp := t.TempDir()
		os.MkdirAll(filepath.Join(tmp, "data"), 0755)
		os.WriteFile(filepath.Join(tmp, "data", "foo.txt"), []byte("x"), 0644)

		result := DiscoverValues(tmp, models.EachSource{Glob: "./data/*"}, nil)
		if len(result) != 1 || result[0] != "foo.txt" {
			t.Errorf("expected [foo.txt], got %v", result)
		}
	})

	t.Run("hidden_excluded", func(t *testing.T) {
		tmp := t.TempDir()
		os.MkdirAll(filepath.Join(tmp, ".hidden"), 0755)
		os.MkdirAll(filepath.Join(tmp, "visible"), 0755)

		result := DiscoverValues(tmp, models.EachSource{Glob: "./*/"}, nil)
		if len(result) != 1 || result[0] != "visible" {
			t.Errorf("expected [visible], got %v", result)
		}
	})

	t.Run("empty_glob", func(t *testing.T) {
		result := DiscoverValues("/tmp", models.EachSource{Glob: ""}, nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

func TestExpandEach(t *testing.T) {
	t.Run("single_var_explicit", func(t *testing.T) {
		each := map[string]models.EachSource{
			"env": {Values: []string{"prod", "staging"}},
		}
		result := ExpandEach("/tmp", each, nil)
		if len(result) != 2 {
			t.Fatalf("expected 2 combos, got %d: %v", len(result), result)
		}
		assertContainsLabels(t, result, map[string]string{"env": "prod"})
		assertContainsLabels(t, result, map[string]string{"env": "staging"})
	})

	t.Run("cartesian_product", func(t *testing.T) {
		each := map[string]models.EachSource{
			"env":    {Values: []string{"prod", "dev"}},
			"region": {Values: []string{"us", "eu"}},
		}
		result := ExpandEach("/tmp", each, nil)
		if len(result) != 4 {
			t.Fatalf("expected 4 combos, got %d: %v", len(result), result)
		}
		assertContainsLabels(t, result, map[string]string{"env": "prod", "region": "us"})
		assertContainsLabels(t, result, map[string]string{"env": "dev", "region": "eu"})
	})

	t.Run("with_glob", func(t *testing.T) {
		tmp := t.TempDir()
		os.MkdirAll(filepath.Join(tmp, "svcA"), 0755)
		os.MkdirAll(filepath.Join(tmp, "svcB"), 0755)

		each := map[string]models.EachSource{
			"svc": {Glob: "./*/"},
		}
		result := ExpandEach(tmp, each, nil)
		if len(result) != 2 {
			t.Fatalf("expected 2 combos, got %d: %v", len(result), result)
		}
		assertContainsLabels(t, result, map[string]string{"svc": "svcA"})
		assertContainsLabels(t, result, map[string]string{"svc": "svcB"})
	})

	t.Run("empty_each", func(t *testing.T) {
		result := ExpandEach("/tmp", map[string]models.EachSource{}, nil)
		if len(result) != 1 || len(result[0]) != 0 {
			t.Errorf("expected [{}], got %v", result)
		}
	})
}

func TestExpandCombinations(t *testing.T) {
	t.Run("single_entry", func(t *testing.T) {
		combos := []map[string]models.EachSource{
			{"db": {Values: []string{"postgres", "mysql"}}, "suite": {Values: []string{"full"}}},
		}
		result := ExpandCombinations("/tmp", combos, nil)
		if len(result) != 2 {
			t.Fatalf("expected 2 combos, got %d: %v", len(result), result)
		}
		assertContainsLabels(t, result, map[string]string{"db": "postgres", "suite": "full"})
		assertContainsLabels(t, result, map[string]string{"db": "mysql", "suite": "full"})
	})

	t.Run("union_of_entries", func(t *testing.T) {
		combos := []map[string]models.EachSource{
			{"db": {Values: []string{"postgres"}}, "suite": {Values: []string{"full"}}},
			{"db": {Values: []string{"sqlite"}}, "suite": {Values: []string{"basic"}}},
		}
		result := ExpandCombinations("/tmp", combos, nil)
		if len(result) != 2 {
			t.Fatalf("expected 2 combos, got %d: %v", len(result), result)
		}
		assertContainsLabels(t, result, map[string]string{"db": "postgres", "suite": "full"})
		assertContainsLabels(t, result, map[string]string{"db": "sqlite", "suite": "basic"})
	})

	t.Run("empty_returns_empty_dict", func(t *testing.T) {
		result := ExpandCombinations("/tmp", []map[string]models.EachSource{}, nil)
		if len(result) != 1 || len(result[0]) != 0 {
			t.Errorf("expected [{}], got %v", result)
		}
	})
}

func TestResolveFiles(t *testing.T) {
	t.Run("substitutes_labels_and_globs", func(t *testing.T) {
		tmp := t.TempDir()
		os.MkdirAll(filepath.Join(tmp, "web", "src"), 0755)
		os.WriteFile(filepath.Join(tmp, "web", "src", "main.py"), []byte("x"), 0644)

		result, err := ResolveFiles(tmp, "{svc}/src/*.py", map[string]string{"svc": "web"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 || result[0] != "web/src/main.py" {
			t.Errorf("expected [web/src/main.py], got %v", result)
		}
	})

	t.Run("double_star", func(t *testing.T) {
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
