package patterns

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	ignore "github.com/sabhiram/go-gitignore"

	"github.com/testmd/testmd/internal/models"
)

// LoadIgnorefile loads a gitignore-format file from root.
func LoadIgnorefile(root, filename string) *ignore.GitIgnore {
	path := filepath.Join(root, filename)
	ig, err := ignore.CompileIgnoreFile(path)
	if err != nil {
		return nil
	}
	return ig
}

// DiscoverValues resolves an EachSource to a sorted, deduplicated list of values.
func DiscoverValues(root string, src models.EachSource, ig *ignore.GitIgnore) []string {
	if len(src.Values) > 0 {
		sorted := make([]string, len(src.Values))
		copy(sorted, src.Values)
		sort.Strings(sorted)
		return sorted
	}

	if src.Glob == "" {
		return nil
	}

	pat := strings.TrimPrefix(src.Glob, "./")
	dirsOnly := strings.HasSuffix(pat, "/")
	pat = strings.TrimSuffix(pat, "/")

	// Detect extension to strip: only if the last segment has a literal extension like *.yaml
	lastSeg := filepath.Base(pat)
	stripExt := ""
	if idx := strings.LastIndex(lastSeg, "."); idx > 0 && strings.Contains(lastSeg[:idx], "*") {
		stripExt = lastSeg[idx:]
	}

	fsys := os.DirFS(root)
	matches, err := doublestar.Glob(fsys, pat)
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	var result []string
	for _, m := range matches {
		rel := filepath.ToSlash(m)

		// Filter: directories only if trailing /
		full := filepath.Join(root, m)
		info, err := os.Stat(full)
		if err != nil {
			continue
		}
		if dirsOnly && !info.IsDir() {
			continue
		}

		// Filter: hidden files
		base := filepath.Base(m)
		if strings.HasPrefix(base, ".") {
			continue
		}

		// Filter: ignorefile
		if ig != nil {
			checkPath := rel
			if info.IsDir() {
				checkPath += "/"
			}
			if ig.MatchesPath(checkPath) {
				continue
			}
		}

		name := base
		if stripExt != "" && strings.HasSuffix(name, stripExt) {
			name = strings.TrimSuffix(name, stripExt)
		}

		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}

	sort.Strings(result)
	return result
}

// ExpandEach computes the cartesian product of all each-sources.
func ExpandEach(root string, each map[string]models.EachSource, ig *ignore.GitIgnore) []map[string]string {
	if len(each) == 0 {
		return []map[string]string{{}}
	}

	// Sort keys for determinism
	keys := make([]string, 0, len(each))
	for k := range each {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := []map[string]string{{}}
	for _, key := range keys {
		values := DiscoverValues(root, each[key], ig)
		if len(values) == 0 {
			return nil
		}
		var next []map[string]string
		for _, combo := range result {
			for _, val := range values {
				newCombo := copyMap(combo)
				newCombo[key] = val
				next = append(next, newCombo)
			}
		}
		result = next
	}
	return result
}

// ExpandCombinations computes the union of entries, each entry is a cartesian product.
func ExpandCombinations(root string, combos []map[string]models.EachSource, ig *ignore.GitIgnore) []map[string]string {
	var all []map[string]string
	for _, entry := range combos {
		for _, combo := range ExpandEach(root, entry, ig) {
			if !containsLabels(all, combo) {
				all = append(all, combo)
			}
		}
	}
	if len(all) == 0 {
		return []map[string]string{{}}
	}
	return all
}

// ResolveFiles substitutes labels and globs for matching files.
func ResolveFiles(root, pattern string, labels map[string]string, ig *ignore.GitIgnore) ([]string, error) {
	resolved := pattern
	for k, v := range labels {
		resolved = strings.ReplaceAll(resolved, "{"+k+"}", v)
	}
	resolved = strings.TrimPrefix(resolved, "./")

	fsys := os.DirFS(root)
	matches, err := doublestar.Glob(fsys, resolved)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, m := range matches {
		full := filepath.Join(root, m)
		info, err := os.Stat(full)
		if err != nil || info.IsDir() {
			continue
		}
		rel := filepath.ToSlash(m)
		if ig != nil && ig.MatchesPath(rel) {
			continue
		}
		files = append(files, rel)
	}
	sort.Strings(files)
	return files, nil
}

func copyMap(m map[string]string) map[string]string {
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

func containsLabels(list []map[string]string, labels map[string]string) bool {
	for _, existing := range list {
		if len(existing) != len(labels) {
			continue
		}
		match := true
		for k, v := range labels {
			if existing[k] != v {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
