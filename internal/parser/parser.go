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
			ID           string                   `yaml:"id"`
			Watch        interface{}               `yaml:"watch"`
			Each         map[string]interface{}    `yaml:"each"`
			Combinations []map[string]interface{}  `yaml:"combinations"`
		}
		if err := yaml.Unmarshal([]byte(yamlContent), &config); err != nil {
			return fm, nil, fmt.Errorf("test '%s' (line %d): invalid yaml: %w", title, sourceLine, err)
		}

		watch, err := toStringSlice(config.Watch)
		if err != nil || len(watch) == 0 {
			return fm, nil, fmt.Errorf("test '%s' (line %d): missing watch", title, sourceLine)
		}

		if config.Each != nil && config.Combinations != nil {
			return fm, nil, fmt.Errorf("test '%s' (line %d): cannot use both 'each' and 'combinations'", title, sourceLine)
		}

		var each map[string]models.EachSource
		if config.Each != nil {
			each, err = parseEachMap(config.Each)
			if err != nil {
				return fm, nil, fmt.Errorf("test '%s' (line %d): invalid each: %w", title, sourceLine, err)
			}
		}

		var combinations []map[string]models.EachSource
		if config.Combinations != nil {
			for j, entry := range config.Combinations {
				parsed, err := parseEachMap(entry)
				if err != nil {
					return fm, nil, fmt.Errorf("test '%s' (line %d): invalid combinations[%d]: %w", title, sourceLine, j, err)
				}
				combinations = append(combinations, parsed)
			}
		}

		description := strings.TrimSpace(body[:m[0]] + body[m[1]:])

		tests = append(tests, models.TestDefinition{
			Title:        title,
			ExplicitID:   config.ID,
			Watch:        watch,
			Each:         each,
			Combinations: combinations,
			Description:  description,
			SourceFile:   sourceFile,
			SourceLine:   sourceLine,
		})
	}

	return fm, tests, nil
}

func parseEachMap(raw map[string]interface{}) (map[string]models.EachSource, error) {
	result := make(map[string]models.EachSource, len(raw))
	for k, v := range raw {
		src, err := parseEachSource(v)
		if err != nil {
			return nil, fmt.Errorf("variable '%s': %w", k, err)
		}
		result[k] = src
	}
	return result, nil
}

func parseEachSource(v interface{}) (models.EachSource, error) {
	switch val := v.(type) {
	case string:
		return models.EachSource{Glob: val}, nil
	case []interface{}:
		values := make([]string, len(val))
		for i, item := range val {
			s, ok := item.(string)
			if !ok {
				return models.EachSource{}, fmt.Errorf("expected string, got %T", item)
			}
			values[i] = s
		}
		return models.EachSource{Values: values}, nil
	default:
		return models.EachSource{}, fmt.Errorf("expected string (glob) or list (values), got %T", v)
	}
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
