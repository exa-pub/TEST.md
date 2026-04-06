# testmd — Specification

testmd encodes cross-cutting rules — "if you changed X, verify Y" — as executable contracts in `TEST.md` files. Every codebase has implicit knowledge: rename an API field and the docs break, change a schema and the migration needs updating. testmd makes these rules explicit, trackable, and enforceable in CI.

This is especially valuable when code is written by AI agents, which have no way of knowing a project's unwritten rules. An agent runs `testmd ci`, sees which contracts its changes have broken, and either fixes the issues or reports what it cannot resolve.

## Core loop

1. Developer or agent changes code
2. `testmd status` / `testmd ci` shows which contracts are affected (file hashes changed)
3. The author verifies each flagged area and runs `testmd resolve <id>` or `testmd fail <id> <message>`
4. CI calls `testmd ci` and fails if there are unresolved tests

## TEST.md format

A TEST.md file has three optional sections, in order:

1. **Frontmatter** — YAML between `---` delimiters at the very beginning
2. **Test definitions** — sections starting with `# Title`
3. **State block** — auto-managed, at the end of the file

### Frontmatter

```yaml
---
include: [path/to/other/TEST.md]
ignorefile: .gitignore
---
```

| Field        | Type   | Default      | Description                                                  |
|--------------|--------|--------------|--------------------------------------------------------------|
| `include`    | list   | `[]`         | Paths to other TEST.md files (relative to current file). Tests from included files are merged. Each file stores its own state. Nested includes are not supported. |
| `ignorefile` | string | `.gitignore` | Path to a gitignore-format file (relative to root). Matching entries are excluded from label discovery and file hashing. |

### Test definition

Each test starts with a level-1 heading (`# Title`) followed by a YAML config block and a description:

```markdown
# OAuth login flow

```yaml
id: oauth
on_change: ./services/$provider/**
matrix:
  - match:
      - ./services/$provider/
    const:
      env: [prod, staging]
```

Verify that OAuth works for each provider:
1. Navigate to /login
2. Click "Sign in with $provider"
3. Verify redirect and session creation
```

### Test config fields

| Field       | Type           | Required | Default | Description                                   |
|-------------|----------------|----------|---------|-----------------------------------------------|
| `on_change` | string or list | **yes**  | —       | Glob pattern(s) for watched files             |
| `id`        | string         | no       | —       | Explicit first part of the test id            |
| `matrix`    | list           | no       | —       | Label combinations (see [Matrix](#matrix))    |

### State block

State is stored as a fenced code block at the end of the file, wrapped in an HTML comment:

```markdown
<!-- State
```testmd
{
  "version": 1,
  "tests": {
    "abc123-def456": {
      "title": "OAuth login flow",
      "labels": {"provider": "google", "env": "prod"},
      "content_hash": "...",
      "files": {
        "services/google/main.go": "..."
      },
      "status": "resolved",
      "resolved_at": "2026-04-06T12:00:00Z",
      "failed_at": null,
      "message": null
    }
  }
}
```
-->
```

The HTML comment makes the block invisible in markdown renderers. The `testmd` language tag on the code fence allows the tool to find and replace the block. The JSON is formatted with indent for readable diffs.

Implementations MUST:
- Strip the state block before parsing test definitions
- Replace the existing block (or append if absent) when saving state
- Use the `<!-- State\n```testmd\n...\n```\n-->` format exactly
- Never include the state block in test descriptions

---

## Patterns

The `on_change` field uses glob patterns with label variables:

```
./path/to/file.go          — exact file
./*.go                     — single-level wildcard
./services/$name/**        — label variable + recursive glob
./services/$name/api/*.go  — mixed
```

### Special segments

| Segment       | Meaning                                                       |
|---------------|---------------------------------------------------------------|
| `$identifier` | Label variable. Each unique value produces a test instance.   |
| `*`           | Any name at one path level (standard glob)                    |
| `**`          | Any sub-path, zero or more levels (standard glob)             |

### Label discovery (auto mode)

When `matrix` is **not** specified, `$var` segments in `on_change` are discovered from the filesystem:

1. Walk the pattern segments left-to-right
2. When a `$var` segment is reached, enumerate directory entries at that position
3. Skip entries starting with `.` (dotfiles)
4. Skip entries matching the ignorefile
5. When a glob wildcard (`*`, `**`, `?`) is reached, stop walking — the rest is handled by glob
6. Each unique value of `$var` produces a separate test instance

Example: pattern `./services/$name/**` with filesystem `services/{auth,billing}/` produces two instances with labels `name=auth` and `name=billing`.

### Label discovery with multiple patterns

When `on_change` is a list of patterns:
- Labels are discovered from all patterns that contain `$var`
- Patterns without `$var` are applied to all instances
- The final file set for each instance is the union of files from all patterns

---

## Matrix

Matrix decouples label generation from file watching. It is useful when:
- Labels don't derive from file structure
- Fixed combinations are needed (environments, configs)
- Auto-discovery and explicit values should be mixed

### Syntax

```yaml
matrix:
  - const:
      env: [prod, staging]
      region: [us, eu]

  - match:
      - ./services/$service/
    const:
      tier: [api, worker]

  - match:
      - ./modules/$module/
```

### Entry types

**`const`** — explicit values, cartesian product within one entry:

```yaml
- const:
    A: [a, b]
    B: [x, y]
# produces: A=a,B=x  A=a,B=y  A=b,B=x  A=b,B=y
```

**`match`** — discover values from filesystem (same algorithm as auto-discovery):

```yaml
- match:
    - ./services/$name/
# produces: name=auth  name=billing  (from filesystem)
```

**`match` + `const`** — cartesian product of discovered and explicit:

```yaml
- match:
    - ./services/$name/
  const:
    env: [prod, dev]
# produces: name=auth,env=prod  name=auth,env=dev  name=billing,env=prod ...
```

### Combining entries

Entries in the matrix list are combined via **union** (not cartesian product):

```yaml
matrix:
  - const: {A: [a, b], B: [x]}
  - const: {A: [c], B: [y, z]}
# produces: A=a,B=x  A=b,B=x  A=c,B=y  A=c,B=z
```

### Interaction with on_change

- Without `matrix`: `$var` in `on_change` = auto-discovery + substitution (sugar)
- With `matrix`: matrix is the sole source of labels, `$var` in `on_change` is substitution only

**Validation rules:**
- `$var` in `on_change` not defined in `matrix` → **error**
- `$var` in `matrix` not used in `on_change` → **warning** (instances will watch the same files)

---

## Test identification

### ID format: `aabbcc-ddeeff`

```
aabbcc   — first 6 hex chars of sha256(title), or sha256(explicit_id) if set
ddeeff   — first 6 hex chars of sha256(label_string)
```

The label string is a sorted, comma-separated list of `key=value` pairs. For tests without labels, it is an empty string (hash: `e3b0c4`).

The explicit `id` field provides a stable input to the hash, so renaming the title doesn't change the id.

### ID abbreviations

Users can refer to tests by:
- Full id: `aabbcc-ddeeff` — exact match
- First part: `aabbcc` — matches all instances of that test
- Prefix: `aab` — matches if unambiguous

Resolution order: exact → first-part → prefix.

---

## File hashing

For each test instance:

1. Expand `on_change` patterns into a file list (with label substitution)
2. Exclude the source TEST.md file itself (it contains the state block, which changes on every resolve)
3. Sort files alphabetically
4. For each file: `sha256(relative_path + "\0" + file_content)`
5. Content hash: `sha256(concat(all_file_hashes))`

The content hash is compared with the stored value. If different, the test status becomes `outdated`.

The individual file hashes are stored in state to show **which files** changed.

---

## Test statuses

| Status     | Meaning                                            |
|------------|----------------------------------------------------|
| `pending`  | New test, or test with no stored state              |
| `resolved` | Verified and passed                                 |
| `failed`   | Verified and failed (with message)                  |
| `outdated` | Was resolved/failed, but content hash changed       |

### Transitions

```
new test ──────────────────────────► pending
pending  ── resolve ──────────────► resolved
pending  ── fail ─────────────────► failed
resolved ── content hash changed ─► outdated
failed   ── content hash changed ─► outdated
outdated ── resolve ──────────────► resolved
outdated ── fail ─────────────────► failed
```

---

## Ignorefile

The `ignorefile` frontmatter field (default: `.gitignore`) specifies a gitignore-format file. Matching entries are excluded from:

1. **Label discovery** — ignored directories are not enumerated as `$var` values
2. **File matching** — ignored files are not included in hash computation

This prevents `__pycache__`, `node_modules`, build artifacts, etc. from affecting tests.

For directory entries, the path is checked with a trailing `/` to match gitignore directory patterns correctly.

---

## State storage

### Location

State is stored inline in each TEST.md file as described in the [State block](#state-block) section. There is no external state directory.

### Per-file state with includes

When using `include`, each TEST.md file stores state for its own tests only. The root file does not aggregate state from included files. When saving:

1. Group test instances by their source file
2. Write each file's state block with only its tests
3. If a file has no tests with state, remove the state block

### State record fields

| Field          | Type              | Description                              |
|----------------|-------------------|------------------------------------------|
| `title`        | string            | Test title (from `# Title`)              |
| `labels`       | object            | Label key-value pairs                     |
| `content_hash` | string            | Hash of all watched files at resolve time |
| `files`        | object            | `{relative_path: sha256_hash}` for each file |
| `status`       | string            | `pending`, `resolved`, `failed`, `outdated` |
| `resolved_at`  | string or null    | ISO 8601 timestamp                        |
| `failed_at`    | string or null    | ISO 8601 timestamp                        |
| `message`      | string or null    | Failure message                           |

---

## Label substitution

Label variables (`$var`) are substituted in:
- `on_change` patterns (for file matching)
- Test titles (in `get` output)
- Test descriptions (in `get` output)

This allows descriptions to reference the current label values:

```markdown
# $service health check

Verify that `$service` responds to healthcheck on `$env`.
```
