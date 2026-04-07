package parser

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/testmd/testmd/internal/models"
)

var yamlBlockRE = regexp.MustCompile("(?s)```ya?ml\n(.*?)```")

// Frontmatter holds parsed frontmatter fields.
type Frontmatter struct {
	Include    []string `yaml:"include"`
	Ignorefile string   `yaml:"ignorefile"`
}

// Parse parses TEST.md content into frontmatter and test definitions.
func Parse(text, sourceFile string) (Frontmatter, []models.TestDefinition, error) {
	var fm Frontmatter
	lineOffset := 0

	// 1. Extract frontmatter
	if strings.HasPrefix(text, "---\n") {
		end := strings.Index(text[4:], "\n---\n")
		if end != -1 {
			if err := yaml.Unmarshal([]byte(text[4:4+end]), &fm); err != nil {
				return fm, nil, fmt.Errorf("invalid frontmatter: %w", err)
			}
			lineOffset = strings.Count(text[:4+end+5], "\n")
			text = text[4+end+5:]
		}
	}

	// 2. Parse tests
	lines := strings.Split(text, "\n")
	var tests []models.TestDefinition
	i := 0

	for i < len(lines) {
		if !strings.HasPrefix(lines[i], "# ") {
			i++
			continue
		}

		title := strings.TrimSpace(lines[i][2:])
		sourceLine := i + 1 + lineOffset
		i++

		var bodyLines []string
		for i < len(lines) && !strings.HasPrefix(lines[i], "# ") {
			bodyLines = append(bodyLines, lines[i])
			i++
		}

		body := strings.Join(bodyLines, "\n")

		m := yamlBlockRE.FindStringSubmatchIndex(body)
		if m == nil {
			return fm, nil, fmt.Errorf("test '%s' (line %d): missing yaml config block", title, sourceLine)
		}

		yamlContent := body[m[2]:m[3]]
		var config struct {
			ID       string      `yaml:"id"`
			OnChange interface{} `yaml:"on_change"`
			Matrix   []struct {
				Match interface{}         `yaml:"match"`
				Const map[string][]string `yaml:"const"`
			} `yaml:"matrix"`
		}
		if err := yaml.Unmarshal([]byte(yamlContent), &config); err != nil {
			return fm, nil, fmt.Errorf("test '%s' (line %d): invalid yaml: %w", title, sourceLine, err)
		}

		onChange, err := toStringSlice(config.OnChange)
		if err != nil || len(onChange) == 0 {
			return fm, nil, fmt.Errorf("test '%s' (line %d): missing on_change", title, sourceLine)
		}

		description := strings.TrimSpace(body[:m[0]] + body[m[1]:])

		var matrix []models.MatrixEntry
		if config.Matrix != nil {
			for _, entry := range config.Matrix {
				me := models.MatrixEntry{Const: entry.Const}
				if entry.Match != nil {
					me.Match, err = toStringSlice(entry.Match)
					if err != nil {
						return fm, nil, fmt.Errorf("test '%s' (line %d): invalid match in matrix: %w", title, sourceLine, err)
					}
				}
				matrix = append(matrix, me)
			}
		}

		tests = append(tests, models.TestDefinition{
			Title:       title,
			ExplicitID:  config.ID,
			OnChange:    onChange,
			Matrix:      matrix,
			Description: description,
			SourceFile:  sourceFile,
			SourceLine:  sourceLine,
		})
	}

	return fm, tests, nil
}

func toStringSlice(v interface{}) ([]string, error) {
	switch val := v.(type) {
	case string:
		return []string{val}, nil
	case []interface{}:
		result := make([]string, len(val))
		for i, item := range val {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("expected string, got %T", item)
			}
			result[i] = s
		}
		return result, nil
	case []string:
		return val, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("expected string or list, got %T", v)
	}
}
