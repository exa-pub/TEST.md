package models

// EachSource defines where variable values come from.
// Exactly one of Glob or Values is set.
type EachSource struct {
	Glob   string   // e.g. "./services/*/" — discover from filesystem
	Values []string // e.g. ["prod", "staging"] — explicit list
}

// TestDefinition is one test from TEST.md (before label expansion).
type TestDefinition struct {
	Title        string
	ExplicitID   string                    // empty if not set
	Watch        []string                  // glob patterns for watched files
	Each         map[string]EachSource     // nil if not specified; cartesian product
	Combinations []map[string]EachSource   // nil if not specified; union of entries
	Description  string
	SourceFile   string // absolute path
	SourceLine   int
}

// TestInstance is a concrete test after label expansion + file hashing.
type TestInstance struct {
	ID               string
	Definition       *TestDefinition
	Labels           map[string]string
	ResolvedPatterns []string
	MatchedFiles     []string
	ContentHash      string
	FileHashes       map[string]string
}

// TestRecord is a state entry stored in TEST.md.lock.
type TestRecord struct {
	Title       string            `json:"title"`
	Labels      map[string]string `json:"labels"`
	ContentHash string            `json:"content_hash"`
	Files       map[string]string `json:"files"`
	Status      string            `json:"status"`
	ResolvedAt  *string           `json:"resolved_at"`
	FailedAt    *string           `json:"failed_at"`
	Message     *string           `json:"message"`
}

// State is the top-level state structure in the lock file.
type State struct {
	Version int                    `json:"version"`
	Tests   map[string]*TestRecord `json:"tests"`
}

// StatusResult pairs an instance with its effective status.
type StatusResult struct {
	Instance *TestInstance
	Status   string      // "pending", "resolved", "failed", "outdated"
	Record   *TestRecord // nil for pending
}
