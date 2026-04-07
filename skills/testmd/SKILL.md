---
name: testmd
description: Work with testmd — a tool for tracking manual/semi-automated tests described in TEST.md files. Helps understand the workflow, run commands, and resolve tests correctly.
---

# testmd skill

testmd encodes cross-cutting rules — "if you changed X, verify Y" — as executable contracts in `TEST.md` files. When code changes, testmd detects which contracts are affected via file hashing and requires explicit resolution.

## Setup

If the `testmd` command is not available, install it:

```bash
curl -fsSL https://raw.githubusercontent.com/exa-pub/test.md/main/install.sh | sh
```

This auto-detects OS/arch, downloads the latest release binary, verifies checksums, and installs to `/usr/local/bin` (or `~/.local/bin` if no root access). You can pin a version with `TESTMD_VERSION=v1.0.0` or change the install path with `TESTMD_INSTALL_DIR=/path`.

## Core workflow

1. After changing code, run `testmd status` to see which tests are affected
2. For each affected test, run `testmd get <id>` to read the test description and verification steps
3. Perform the verification (manually or by running described checks)
4. Mark each test: `testmd resolve <id>` (passed) or `testmd fail <id> "reason"` (failed)

In CI, `testmd ci` exits non-zero if any test is not `resolved`.

## Commands

| Command | Purpose |
|---|---|
| `testmd status` | Show all tests and their statuses |
| `testmd get <id>` | Show full test details: title, labels, description, watched files, status |
| `testmd resolve <id>` | Mark test as resolved (verified and passing) |
| `testmd fail <id> "msg"` | Mark test as failed with a reason |
| `testmd ci` | CI gate — fails if any test is not resolved |
| `testmd gc` | Remove orphaned state records |

IDs support abbreviations: full `aabbcc-ddeeff`, first-part `aabbcc`, or unambiguous prefix `aab`.

## Critical principles

### Use `testmd get`, not file reading

TEST.md files can be very large. **Always use `testmd get <id>` to read test details** instead of reading TEST.md directly. The `get` command:
- Substitutes label variables (`{var}`) with actual values in titles and descriptions
- Shows only the relevant test, not the entire file
- Includes current status, watched files, and labels
- Is far more efficient than parsing markdown yourself

### Use `testmd status` for discovery

Don't parse TEST.md to figure out what tests exist or which are affected. Run `testmd status` — it computes file hashes and shows exactly which tests need attention.

### Resolve after verification, not before

Only run `testmd resolve <id>` after you have actually verified that the test passes. The resolve command records the current file hashes — resolving prematurely means the test won't trigger again if you make more changes.

### Understand statuses

| Status | Meaning | Action needed |
|---|---|---|
| `pending` | New test, never verified | Verify and resolve/fail |
| `resolved` | Verified and passing | None |
| `failed` | Verified and failing | Fix the issue, then resolve |
| `outdated` | Was resolved/failed, but watched files changed | Re-verify and resolve/fail |

### State lives in TEST.md.lock

All state is stored in `TEST.md.lock` files (JSON, one per TEST.md). Don't modify lock files manually — always use `testmd resolve` / `testmd fail`.

### When you change code

After making code changes:
1. Run `testmd status` to check if any tests became `outdated` or are `pending`
2. For each non-resolved test, run `testmd get <id>` to understand what to verify
3. Perform the verification steps described in the test
4. Resolve or fail each test

This is especially important before committing or opening a PR — `testmd ci` will block merges if tests are unresolved.

## TEST.md file structure

A TEST.md file has two optional sections, in order:

1. **Frontmatter** — YAML between `---` delimiters at the very beginning
2. **Test definitions** — sections starting with `# Title`

State is stored separately in `TEST.md.lock`.

### Frontmatter

Optional YAML block at the top of the file:

````markdown
---
include: [tests/integration/TEST.md, tests/e2e/TEST.md]
ignorefile: .testmdignore
---
````

| Field | Default | Description |
|---|---|---|
| `include` | `[]` | Paths to other TEST.md files (relative to current). Tests are merged; each file stores its own state. |
| `ignorefile` | `.gitignore` | Gitignore-format file. Matching entries are excluded from label discovery and file hashing. |

### Test definition

Each test is an h1 heading + a YAML config block + a free-form description:

````markdown
# API returns valid JSON

```yaml
watch: ./src/api/**
```

Send GET /users and verify response is valid JSON with correct schema.
````

Config fields:

| Field | Type | Required | Description |
|---|---|---|---|
| `watch` | string or list | **yes** | Glob pattern(s) for watched files |
| `id` | string | no | Explicit stable id (so renaming the title doesn't change the id) |
| `each` | object | no | Variable sources, cartesian product |
| `combinations` | list | no | Variable sources, union of entries |

### Watch patterns

```yaml
watch: ./src/auth/**          # recursive glob
watch: ./config/*.yaml        # single-level wildcard
watch:                        # multiple patterns
  - ./src/api/**
  - ./schema/openapi.yaml
watch: ./services/{name}/**   # variable substitution
```

Variables use `{name}` syntax — substituted before globbing.

### Variables: `each` (cartesian product)

`each` defines variable sources. All are expanded as a cartesian product.

````markdown
# {service} healthcheck

```yaml
each:
  service: ./services/*/
watch: ./services/{service}/**
```

Verify `{service}` responds to GET /health with 200.
````

Source types:
- `./path/*/` — directory names (trailing `/` = dirs only)
- `./path/*.ext` — file names without extension
- `[a, b, c]` — explicit list

With filesystem `services/{auth,billing,gateway}/`, this creates three test instances.

### Variables: `combinations` (union)

When cartesian product is not appropriate:

```yaml
combinations:
  - db: [postgres, mysql]
    suite: [full]
  - db: [sqlite]
    suite: [basic]
watch: ./migrations/{db}/**
```

Each entry is a cartesian product internally; entries are combined via union.

### Includes: splitting tests across files

````markdown
---
include: [tests/integration/TEST.md, tests/e2e/TEST.md]
---

# Unit test sanity

```yaml
watch: ./src/**
```

Run `make test` and verify all unit tests pass.
````

`testmd status` shows tests from all included files. Each file stores its own state independently. Nested includes (an included file including another) are not supported.

### Full example

A complete TEST.md showing multiple features together:

````markdown
---
include: [components/TEST.md]
ignorefile: .gitignore
---

# Go implementation works correctly

```yaml
watch:
  - ./internal/**
  - ./cmd/**
```

Go code changed — verify it works correctly:
1. Go tests pass: `go test ./internal/...`
2. Build succeeds: `go build -o ./bin/ ./cmd/...`
3. Run `./testmd-go status` on a sample TEST.md and verify output

# Deploy smoke test for {service}

```yaml
each:
  service: ./services/*/
  env: [prod, staging]
watch:
  - ./services/{service}/**
  - ./deploy/{env}.yaml
```

After deploying `{service}` to `{env}`:
1. Verify the service starts
2. Check /health returns 200
3. Run basic smoke test
````
