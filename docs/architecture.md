# testmd — Architecture

This document describes the internal architecture of testmd, independent of implementation language.

## Data flow

```
TEST.md ──► parse ──► TestDefinition[] ──► expand labels ──► TestInstance[]
                                               │                    │
                                          (matrix or            (hash files)
                                           auto-discover)           │
                                                              ┌─────┘
                                                              ▼
                                          state.json ◄──► compute statuses
                                        (TEST.md.lock)        │
                                                              ▼
                                                         CLI output
```

## Components

### Parser

**Input:** TEST.md file content, source file path
**Output:** frontmatter dict, list of TestDefinition

Responsibilities:
1. Extract frontmatter (`---` delimited YAML at file start)
2. Split by `# ` headings (h1 only)
3. For each heading: extract the first ` ```yaml ` block as config, everything else as description
4. Validate: yaml block required, `on_change` required
5. Track source line numbers (accounting for frontmatter offset)

The parser does NOT handle includes or state loading — those are the caller's responsibility.

### Pattern resolver

Two responsibilities:

**1. Label enumeration** — discover `$var` values from the filesystem.

Algorithm for a single pattern:
```
split pattern into path segments
walk segments left-to-right:
  $var  → enumerate directory entries at this position (skip dotfiles, ignored)
  */?   → stop (rest is glob territory)
  other → descend into literal directory
```

For multiple patterns: union of discovered combinations.
For matrix: expand const (cartesian product) × match (filesystem discovery), then union across entries.

**2. File resolution** — substitute labels into patterns and glob for files.

Algorithm:
```
replace $var with label values
strip leading ./
normalize trailing /** to /**/*  (pathlib compatibility)
glob from root directory
filter: only files, exclude ignored
sort alphabetically
```

### Hasher

Computes content hashes for change detection:

- **File hash:** `sha256(relative_path + "\0" + file_content)` — path is included so renaming a file changes the hash
- **Content hash:** `sha256(concat(file_hashes))` — files must be sorted before concatenation
- **Test ID:** `sha256(title_or_explicit_id)[:6] + "-" + sha256(label_string)[:6]`

### State manager

**Read:** read `TEST.md.lock`, parse JSON. Missing file → empty state.
**Write:** serialize JSON (formatted, indent=2), write to `TEST.md.lock`. Empty state → delete file.

State is per-file. When includes are used, each file has its own lock file.

### Resolver

Ties everything together:

1. For each TestDefinition, expand labels (matrix or auto-discovery)
2. For each label combination, resolve on_change patterns to files
3. Compute content hash from matched files
4. Generate test ID from title and labels
5. Create TestInstance with all computed data

Status computation:
- No stored record → `pending`
- Stored hash ≠ current hash → `outdated`
- Otherwise → stored status (`resolved` or `failed`)

### Reporter

Formats output for terminal (with colors) and files (markdown, JSON).

Groups test instances by their source definition (not by title string — two definitions with the same title from different files are displayed separately).

Label substitution: `$var` in titles and descriptions is replaced with actual values in `get` output.

## Key invariants

1. **State is always in TEST.md.lock** — never inline in TEST.md
2. **Hashing is deterministic** — same files with same content always produce the same hash
3. **Labels are sorted** — in IDs, state records, and display, labels are always sorted by key
4. **Files are sorted** — file lists are always sorted alphabetically before hashing
5. **Lock file is excluded from hashing** — the lock file changes on every resolve and must not affect content hash
6. **Ignorefile applies to both discovery and matching** — an ignored path never produces a label or contributes to a hash
7. **Include is one level** — included files cannot include other files
8. **State is per-file** — each TEST.md has its own lock file storing state for its tests
9. **ID is deterministic** — same title + same labels always produce the same ID

## Module boundaries

| Module    | Depends on       | Responsibility                      |
|-----------|------------------|-------------------------------------|
| models    | —                | Data structures                     |
| parser    | models           | TEST.md → definitions               |
| patterns  | (filesystem)     | Label discovery, file globbing      |
| hashing   | (filesystem)     | SHA256, ID generation               |
| state     | (filesystem)     | Read/write state in TEST.md         |
| resolver  | patterns,hashing | Build instances, compute statuses   |
| report    | resolver         | Format output                       |
| cli       | all above        | CLI commands, path resolution       |

The dependency graph is acyclic. `models` has no dependencies. `cli` depends on everything else but nothing depends on `cli`.
