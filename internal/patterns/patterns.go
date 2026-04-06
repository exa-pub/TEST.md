package patterns

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	ignore "github.com/sabhiram/go-gitignore"
)

var labelVarRE = regexp.MustCompile(`\$([a-zA-Z_]\w*)`)

// FindLabelVars extracts $variable names from a pattern.
func FindLabelVars(pattern string) []string {
	matches := labelVarRE.FindAllStringSubmatch(pattern, -1)
	result := make([]string, len(matches))
	for i, m := range matches {
		result[i] = m[1]
	}
	return result
}

// LoadIgnorefile loads a gitignore-format file from root.
func LoadIgnorefile(root, filename string) *ignore.GitIgnore {
	path := filepath.Join(root, filename)
	ig, err := ignore.CompileIgnoreFile(path)
	if err != nil {
		return nil
	}
	return ig
}

// EnumerateLabels discovers label combinations from on_change patterns.
func EnumerateLabels(root string, patterns []string, ig *ignore.GitIgnore) []map[string]string {
	var all []map[string]string
	for _, pat := range patterns {
		if len(FindLabelVars(pat)) == 0 {
			continue
		}
		for _, combo := range enumeratePattern(root, pat, ig) {
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

// ExpandMatrix expands matrix entries into label combinations (union).
func ExpandMatrix(root string, matrix []ExpandableEntry, ig *ignore.GitIgnore) []map[string]string {
	var all []map[string]string
	for _, entry := range matrix {
		for _, combo := range expandEntry(root, entry, ig) {
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

// ExpandableEntry mirrors models.MatrixEntry for decoupling.
type ExpandableEntry struct {
	Match []string
	Const map[string][]string
}

func expandEntry(root string, entry ExpandableEntry, ig *ignore.GitIgnore) []map[string]string {
	matchCombos := []map[string]string{{}}
	if len(entry.Match) > 0 {
		var discovered []map[string]string
		for _, pat := range entry.Match {
			for _, combo := range enumeratePattern(root, pat, ig) {
				if !containsLabels(discovered, combo) {
					discovered = append(discovered, combo)
				}
			}
		}
		if len(discovered) > 0 {
			matchCombos = discovered
		}
	}

	constCombos := []map[string]string{{}}
	if len(entry.Const) > 0 {
		constCombos = expandConst(entry.Const)
	}

	var result []map[string]string
	for _, mc := range matchCombos {
		for _, cc := range constCombos {
			merged := make(map[string]string, len(mc)+len(cc))
			for k, v := range mc {
				merged[k] = v
			}
			for k, v := range cc {
				merged[k] = v
			}
			result = append(result, merged)
		}
	}
	return result
}

func expandConst(consts map[string][]string) []map[string]string {
	keys := make([]string, 0, len(consts))
	for k := range consts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := []map[string]string{{}}
	for _, key := range keys {
		values := consts[key]
		var next []map[string]string
		for _, combo := range result {
			for _, val := range values {
				newCombo := make(map[string]string, len(combo)+1)
				for k, v := range combo {
					newCombo[k] = v
				}
				newCombo[key] = val
				next = append(next, newCombo)
			}
		}
		result = next
	}
	return result
}

func enumeratePattern(root, pattern string, ig *ignore.GitIgnore) []map[string]string {
	pat := strings.TrimPrefix(pattern, "./")
	parts := strings.Split(pat, "/")
	return walk(root, root, parts, map[string]string{}, ig)
}

func walk(root, base string, parts []string, labels map[string]string, ig *ignore.GitIgnore) []map[string]string {
	if len(parts) == 0 {
		return []map[string]string{copyMap(labels)}
	}

	part := parts[0]
	rest := parts[1:]

	if strings.HasPrefix(part, "$") {
		varName := part[1:]
		entries, err := os.ReadDir(base)
		if err != nil {
			return nil
		}
		var results []map[string]string
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			entryPath := filepath.Join(base, name)
			rel, _ := filepath.Rel(root, entryPath)
			if ig != nil {
				checkPath := filepath.ToSlash(rel)
				if entry.IsDir() {
					checkPath += "/"
				}
				if ig.MatchesPath(checkPath) {
					continue
				}
			}
			newLabels := copyMap(labels)
			newLabels[varName] = name
			results = append(results, walk(root, entryPath, rest, newLabels, ig)...)
		}
		return results
	}

	if strings.ContainsAny(part, "*?") {
		return []map[string]string{copyMap(labels)}
	}

	return walk(root, filepath.Join(base, part), rest, labels, ig)
}

// ResolveFiles substitutes labels and globs for matching files.
func ResolveFiles(root, pattern string, labels map[string]string, ig *ignore.GitIgnore) ([]string, error) {
	resolved := pattern
	for k, v := range labels {
		resolved = strings.ReplaceAll(resolved, "$"+k, v)
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
