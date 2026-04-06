# testmd — CLI Reference

## Global option

All commands accept `--testmd PATH`:

```
testmd [--testmd PATH] <command> [args]
```

| `--testmd`        | TEST.md file                | Root directory (for on_change) |
|-------------------|-----------------------------|-------------------------------|
| not specified     | search upward from cwd      | directory of found file       |
| path to directory | `<dir>/TEST.md`             | the directory                 |
| path to file      | the file                    | parent directory              |

**Upward search:** when `--testmd` is not specified, the tool searches for `TEST.md` in the current directory, then parent, then grandparent, etc. (like `git` searches for `.git/`). Error if not found.

---

## Commands

### `testmd status`

Show the status of all tests.

```
testmd status [--report-md FILE] [--report-json FILE]
```

Output:
```
OAuth login flow
  ✓ abc123-def456  provider=google env=prod  resolved  (2h ago)
  ✗ abc123-789abc  provider=github env=prod  failed    "Redirect broken"
  ⟳ abc123-112233  provider=apple  env=prod  outdated

Database migrations
  … 445566-e3b0c4  pending

Summary: 1 resolved, 1 failed, 1 outdated, 1 pending
```

Flags:
- `--report-md FILE` — save report as markdown
- `--report-json FILE` — save report as JSON

---

### `testmd resolve <id>`

Mark test(s) as resolved. Saves the current content hash.

```
$ testmd resolve abc123-def456
Resolved: OAuth login flow (provider=google env=prod)
```

Using just the first part resolves all instances:
```
$ testmd resolve abc123
Resolved: OAuth login flow (provider=google env=prod)
Resolved: OAuth login flow (provider=github env=prod)
Resolved: OAuth login flow (provider=apple env=prod)
```

---

### `testmd fail <id> <message>`

Mark test(s) as failed with a message.

```
$ testmd fail abc123-789abc "Redirect returns 500 on staging"
Failed: OAuth login flow (provider=github env=prod)
  Message: Redirect returns 500 on staging
```

---

### `testmd get <id>`

Show test details: status, labels, watched patterns, files, and the full description.

```
$ testmd get abc123-789abc

# OAuth login flow
Labels: provider=github env=prod
Status: failed
Failed at: 2026-04-05T10:00:00Z
Message: Redirect returns 500 on staging
Patterns: ./services/github/prod/**
Files: 3
---
Verify that OAuth works for each provider:
1. Navigate to /login
2. Click "Sign in with github"
...
```

For outdated tests, shows which files changed:
```
Changed:
  services/google/handler.go
  services/google/config.yaml
```

Label variables in the title and description are substituted with actual values.

---

### `testmd gc`

Remove orphaned state records — tests that no longer exist in TEST.md or whose label values no longer match the filesystem.

```
$ testmd gc
Removed 2 orphaned record(s).
```

---

### `testmd ci`

Like `status`, but exits with code 1 if any test is not resolved.

```
$ testmd ci
FAIL: 2 test(s) require attention

  ✗  abc123-789abc  OAuth login flow (provider=github env=prod)  failed
  ⟳  abc123-112233  OAuth login flow (provider=apple env=prod)   outdated
```

Exit codes:
- `0` — all tests resolved
- `1` — at least one test is pending, outdated, or failed

Supports `--report-md` and `--report-json` flags.

---

## ID resolution

When specifying `<id>` in commands, you can use:

| Input            | Matches                                          |
|------------------|--------------------------------------------------|
| `abc123-def456`  | Exact match                                      |
| `abc123`         | All instances with first part `abc123`            |
| `abc`            | All instances whose id starts with `abc`          |

Resolution is tried in order: exact → first-part → prefix. First non-empty result wins.
