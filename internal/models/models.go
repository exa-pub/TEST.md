package models

// TestDefinition is one test from TEST.md (before label expansion).
type TestDefinition struct {
	Title       string
	ExplicitID  string // empty if not set
	OnChange    []string
	Matrix      []MatrixEntry // nil if not specified
	Description string
	SourceFile  string // absolute path
	SourceLine  int
}

// MatrixEntry represents one entry in the matrix list.
type MatrixEntry struct {
	Match []string            // patterns for filesystem discovery
	Const map[string][]string // explicit values
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

// TestRecord is a state entry stored in TEST.md.
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

// State is the top-level state structure in the ```testmd block.
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
