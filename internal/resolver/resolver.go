package resolver

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/testmd/testmd/internal/hashing"
	"github.com/testmd/testmd/internal/models"
	"github.com/testmd/testmd/internal/patterns"
)

// BuildInstances expands definitions into concrete instances with computed hashes.
func BuildInstances(root string, defs []models.TestDefinition, ig *ignore.GitIgnore) ([]*models.TestInstance, error) {
	var instances []*models.TestInstance

	for i := range defs {
		defn := &defs[i]
		onChange := rebasePatterns(root, defn.SourceFile, defn.OnChange)
		matrix := rebaseMatrix(root, defn.SourceFile, defn.Matrix)

		var labelCombos []map[string]string
		if len(matrix) > 0 {
			if err := validateMatrixVars(defn); err != nil {
				return nil, err
			}
			entries := make([]patterns.ExpandableEntry, len(matrix))
			for j, m := range matrix {
				entries[j] = patterns.ExpandableEntry{Match: m.Match, Const: m.Const}
			}
			labelCombos = patterns.ExpandMatrix(root, entries, ig)
		} else {
			labelCombos = patterns.EnumerateLabels(root, onChange, ig)
		}

		// Exclude the lock file from hashing — it changes on every resolve.
		lockRel, _ := filepath.Rel(root, defn.SourceFile+".lock")
		lockRel = filepath.ToSlash(lockRel)

		for _, labels := range labelCombos {
			var resolvedPatterns []string
			allFiles := map[string]bool{}

			for _, pat := range onChange {
				resolved := pat
				for k, v := range labels {
					resolved = strings.ReplaceAll(resolved, "$"+k, v)
				}
				resolvedPatterns = append(resolvedPatterns, resolved)

				files, err := patterns.ResolveFiles(root, pat, labels, ig)
				if err != nil {
					return nil, err
				}
				for _, f := range files {
					allFiles[f] = true
				}
			}

			delete(allFiles, lockRel)

			matched := make([]string, 0, len(allFiles))
			for f := range allFiles {
				matched = append(matched, f)
			}
			sort.Strings(matched)

			contentHash, fileHashes, err := hashing.HashFiles(root, matched)
			if err != nil {
				return nil, err
			}
			tid := hashing.MakeID(defn.Title, defn.ExplicitID, labels)

			instances = append(instances, &models.TestInstance{
				ID:               tid,
				Definition:       defn,
				Labels:           labels,
				ResolvedPatterns: resolvedPatterns,
				MatchedFiles:     matched,
				ContentHash:      contentHash,
				FileHashes:       fileHashes,
			})
		}
	}
	return instances, nil
}

// ComputeStatuses determines the effective status of each instance.
func ComputeStatuses(instances []*models.TestInstance, st *models.State) []models.StatusResult {
	results := make([]models.StatusResult, len(instances))
	for i, inst := range instances {
		rec := st.Tests[inst.ID]
		status := "pending"
		if rec != nil {
			if rec.ContentHash != inst.ContentHash {
				status = "outdated"
			} else {
				status = rec.Status
			}
		}
		results[i] = models.StatusResult{Instance: inst, Status: status, Record: rec}
	}
	return results
}

// ResolveTest marks a test as resolved.
func ResolveTest(st *models.State, inst *models.TestInstance) {
	now := time.Now().UTC().Format(time.RFC3339)
	st.Tests[inst.ID] = makeRecord(inst, "resolved")
	st.Tests[inst.ID].ResolvedAt = &now
}

// FailTest marks a test as failed with a message.
func FailTest(st *models.State, inst *models.TestInstance, message string) {
	now := time.Now().UTC().Format(time.RFC3339)
	st.Tests[inst.ID] = makeRecord(inst, "failed")
	st.Tests[inst.ID].FailedAt = &now
	st.Tests[inst.ID].Message = &message
}

// GCState removes orphaned test records. Returns the count removed.
func GCState(st *models.State, instances []*models.TestInstance) int {
	currentIDs := map[string]bool{}
	for _, inst := range instances {
		currentIDs[inst.ID] = true
	}
	var orphans []string
	for id := range st.Tests {
		if !currentIDs[id] {
			orphans = append(orphans, id)
		}
	}
	for _, id := range orphans {
		delete(st.Tests, id)
	}
	return len(orphans)
}

// FindInstances finds instances matching a full id, first-part, or prefix.
func FindInstances(instances []*models.TestInstance, query string) []*models.TestInstance {
	// Exact match
	var exact []*models.TestInstance
	for _, inst := range instances {
		if inst.ID == query {
			exact = append(exact, inst)
		}
	}
	if len(exact) > 0 {
		return exact
	}

	// First-part match
	var byFirst []*models.TestInstance
	for _, inst := range instances {
		parts := strings.SplitN(inst.ID, "-", 2)
		if parts[0] == query {
			byFirst = append(byFirst, inst)
		}
	}
	if len(byFirst) > 0 {
		return byFirst
	}

	// Prefix match
	var byPrefix []*models.TestInstance
	for _, inst := range instances {
		if strings.HasPrefix(inst.ID, query) {
			byPrefix = append(byPrefix, inst)
		}
	}
	return byPrefix
}

// ChangedFiles returns files that differ between instance and stored record.
func ChangedFiles(inst *models.TestInstance, rec *models.TestRecord) []string {
	if rec == nil || rec.Files == nil {
		return inst.MatchedFiles
	}
	changed := map[string]bool{}
	for f, h := range inst.FileHashes {
		if rec.Files[f] != h {
			changed[f] = true
		}
	}
	for f := range rec.Files {
		if _, ok := inst.FileHashes[f]; !ok {
			changed[f] = true
		}
	}
	result := make([]string, 0, len(changed))
	for f := range changed {
		result = append(result, f)
	}
	sort.Strings(result)
	return result
}

func validateMatrixVars(defn *models.TestDefinition) error {
	onChangeVars := map[string]bool{}
	for _, pat := range defn.OnChange {
		for _, v := range patterns.FindLabelVars(pat) {
			onChangeVars[v] = true
		}
	}

	matrixVars := map[string]bool{}
	for _, entry := range defn.Matrix {
		for _, pat := range entry.Match {
			for _, v := range patterns.FindLabelVars(pat) {
				matrixVars[v] = true
			}
		}
		for k := range entry.Const {
			matrixVars[k] = true
		}
	}

	// $var in on_change not in matrix → error
	for v := range onChangeVars {
		if !matrixVars[v] {
			return fmt.Errorf("test '%s': variable $%s in on_change not defined in matrix", defn.Title, v)
		}
	}

	// $var in matrix not in on_change → warning (stderr)
	for v := range matrixVars {
		if !onChangeVars[v] {
			fmt.Fprintf(os.Stderr, "Warning: test '%s': matrix variable $%s not used in on_change\n", defn.Title, v)
		}
	}

	return nil
}

func rebasePatterns(root, sourceFile string, patterns []string) []string {
	sourceDir := filepath.Dir(sourceFile)
	if sourceDir == root {
		return patterns
	}
	rel, err := filepath.Rel(root, sourceDir)
	if err != nil {
		return patterns
	}
	rel = filepath.ToSlash(rel)

	rebased := make([]string, len(patterns))
	for i, p := range patterns {
		p = strings.TrimPrefix(p, "./")
		rebased[i] = "./" + rel + "/" + p
	}
	return rebased
}

func rebaseMatrix(root, sourceFile string, matrix []models.MatrixEntry) []models.MatrixEntry {
	if len(matrix) == 0 || filepath.Dir(sourceFile) == root {
		return matrix
	}
	rebased := make([]models.MatrixEntry, len(matrix))
	for i, entry := range matrix {
		rebased[i] = models.MatrixEntry{Const: entry.Const}
		if len(entry.Match) > 0 {
			rebased[i].Match = rebasePatterns(root, sourceFile, entry.Match)
		}
	}
	return rebased
}

func makeRecord(inst *models.TestInstance, status string) *models.TestRecord {
	labels := inst.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	files := inst.FileHashes
	if files == nil {
		files = map[string]string{}
	}
	return &models.TestRecord{
		Title:       inst.Definition.Title,
		Labels:      labels,
		ContentHash: inst.ContentHash,
		Files:       files,
		Status:      status,
	}
}
