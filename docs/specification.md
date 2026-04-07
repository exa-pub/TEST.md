# testmd — Specification

testmd encodes cross-cutting rules — "if you changed X, verify Y" — as executable contracts in `TEST.md` files. Every codebase has implicit knowledge: rename an API field and the docs break, change a schema and the migration needs updating. testmd makes these rules explicit, trackable, and enforceable in CI.

This is especially valuable when code is written by AI agents, which have no way of knowing a project's unwritten rules. An agent runs `testmd ci`, sees which contracts its changes have broken, and either fixes the issues or reports what it cannot resolve.

## Core loop

1. Developer or agent changes code
2. `testmd status` / `testmd ci` shows which contracts are affected (file hashes changed)
3. The author verifies each flagged area and runs `testmd resolve <id>` or `testmd fail <id> <message>`
4. CI calls `testmd ci` and fails if there are unresolved tests

## TEST.md format

A TEST.md file has two optional sections, in order:

1. **Frontmatter** — YAML between `---` delimiters at the very beginning
2. **Test definitions** — sections starting with `# Title`

State is stored separately in `TEST.md.lock` (see [State file](#state-file)).

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
# OAuth login flow on {env}

```yaml
id: oauth
each:
  provider: ./services/*/
  env: [prod, staging]
watch:
  - ./services/{provider}/**
  - ./deploy/{env}.yaml
```

Verify that OAuth works for each provider:
1. Navigate to /login
2. Click "Sign in with {provider}"
3. Verify redirect and session creation
```

### Test config fields

| Field          | Type           | Required | Default | Description                                         |
|----------------|----------------|----------|---------|-----------------------------------------------------|
| `watch`        | string or list | **yes**  | —       | Glob pattern(s) for watched files                   |
| `id`           | string         | no       | —       | Explicit first part of the test id                  |
| `each`         | object         | no       | —       | Variable sources for cartesian product (see [Each](#each)) |
| `combinations` | list           | no       | —       | Variable sources for union of entries (see [Combinations](#combinations)) |

`each` and `combinations` are mutually exclusive — using both is an error.

### State file

State is stored in a separate lock file alongside TEST.md. For `TEST.md`, state is in `TEST.md.lock`. The lock file contains plain JSON (no markdown wrapping):

```json
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

The JSON is formatted with 2-space indent for readable diffs.

Implementations MUST:
- Read state from `<testmd-path>.lock`
- Write state to `<testmd-path>.lock`
- Delete the lock file when state is empty
- Never modify TEST.md when saving state

---

## Watch patterns

The `watch` field uses glob patterns with variable substitution:

```
./path/to/file.go            — exact file
./*.go                       — single-level wildcard
./services/{name}/**         — variable substitution + recursive glob
./services/{name}/api/*.go   — mixed
```

Variables use `{name}` syntax. Before globbing, `{var}` placeholders are replaced with the actual label values from `each` or `combinations`.

### Special segments

| Segment  | Meaning                                             |
|----------|-----------------------------------------------------|
| `{name}` | Variable placeholder, substituted before glob       |
| `*`      | Any name at one path level (standard glob)          |
| `**`     | Any sub-path, zero or more levels (standard glob)   |

---

## Each

`each` defines variable sources. All variables are expanded as a **cartesian product**, producing one test instance per combination.

```yaml
each:
  service: ./services/*/
  env: [prod, staging]
watch: ./services/{service}/**
```

Each value in the `each` map is a **source**:

| Syntax | Result | Example |
|---|---|---|
| `./path/*/` | Directory names (trailing `/` = dirs only) | `service: ./services/*/` → `[auth, billing]` |
| `./path/*` | File and directory names | `item: ./data/*` → `[foo.txt, bar]` |
| `./path/*.ext` | File names without extension | `config: ./configs/*.yaml` → `[app, db]` |
| `[a, b, c]` | Explicit list | `env: [prod, staging]` |

Glob sources use standard glob syntax (including `**`), are filtered by the ignorefile, and exclude hidden files (starting with `.`). Results are sorted and deduplicated.

Example: `each: {service: ./services/*/, env: [prod, staging]}` with `services/{auth, billing}/` produces 4 instances: `service=auth,env=prod`, `service=auth,env=staging`, `service=billing,env=prod`, `service=billing,env=staging`.

---

## Combinations

When a cartesian product is not appropriate, `combinations` provides explicit control over which label sets are generated. Each entry is an object of variable sources (same syntax as `each`), and entries are combined via **union**.

```yaml
combinations:
  - db: [postgres, mysql]
    suite: [full]
  - db: [sqlite]
    suite: [basic]
watch: ./migrations/{db}/**
```

This produces: `db=postgres,suite=full`, `db=mysql,suite=full`, `db=sqlite,suite=basic`.

Within each entry, sources are expanded as a cartesian product (same as `each`). Across entries, results are unioned.

Glob sources work inside `combinations` too:

```yaml
combinations:
  - service: ./services/*/
    env: [prod, staging]
  - service: [legacy-monolith]
    env: [prod]
```

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

1. Expand `watch` patterns into a file list (with `{var}` substitution)
2. Exclude the source lock file (`TEST.md.lock`) — it changes on every resolve
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

1. **Variable discovery** — ignored directories are not enumerated as variable values
2. **File matching** — ignored files are not included in hash computation

This prevents `__pycache__`, `node_modules`, build artifacts, etc. from affecting tests.

For directory entries, the path is checked with a trailing `/` to match gitignore directory patterns correctly.

---

## State storage

### Location

State is stored in a lock file alongside each TEST.md, as described in the [State file](#state-file) section. `TEST.md` → `TEST.md.lock`.

### Per-file state with includes

When using `include`, each TEST.md file has its own lock file storing state for its own tests only. The root file does not aggregate state from included files. When saving:

1. Group test instances by their source file
2. Write each file's lock file with only its tests
3. If a file has no tests with state, delete the lock file

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

## Variable substitution

Variables use `{var}` syntax and are substituted in:
- `watch` patterns (before file matching)
- Test titles (in `get` output)
- Test descriptions (in `get` output)

This allows descriptions to reference the current label values:

```markdown
# {service} health check

Verify that `{service}` responds to healthcheck on `{env}`.
```
