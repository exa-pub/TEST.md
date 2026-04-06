# testmd

Track manual tests in markdown. Know when they need re-checking.

## What it does

You describe tests in natural language in a `TEST.md` file. testmd watches which source files each test covers. When those files change, the test is marked as outdated until someone re-verifies it.

````markdown
# Login page

```yaml
on_change: ./src/auth/**
```

1. Open /login
2. Fill in email and password
3. Click "Sign in"
4. Verify redirect to /dashboard
````

```
$ testmd status
Login page
  ⟳ a1b2c3-e3b0c4  outdated

$ testmd resolve a1b2c3
Resolved: Login page

$ testmd ci
OK: all tests resolved
```

## Features

- **File watching via hashing** — detects changes without git dependency
- **Label variables** — `./services/$name/**` auto-discovers test instances from filesystem structure
- **Matrix** — explicit label combinations with `const` and `match`
- **Inline state** — test state stored directly in TEST.md, no extra files
- **Includes** — split tests across multiple files
- **Ignorefile** — respects `.gitignore` by default
- **CI mode** — `testmd ci` exits 1 if tests need attention

## Install

```
pip install -e .
```

## Quick start

Create a `TEST.md`:

````markdown
# API returns valid JSON

```yaml
on_change: ./src/api/**
```

Send GET /users and verify response is valid JSON with correct schema.
````

```bash
testmd status          # see what needs checking
testmd resolve a1b2c3  # mark as verified
testmd fail a1b2c3 "schema mismatch on /users"  # mark as failed
testmd get a1b2c3      # see details
testmd gc              # clean up orphaned records
testmd ci              # exit 1 if anything unresolved
```

## Documentation

- [Specification](docs/specification.md) — full format and behavior reference
- [CLI Reference](docs/cli.md) — all commands and options
- [Architecture](docs/architecture.md) — internal design and data flow
- [Examples](docs/examples.md) — common usage patterns

## How state works

State is stored at the bottom of TEST.md itself:

````markdown
<!-- State
```testmd
{"version":1,"tests":{"a1b2c3-e3b0c4":{"status":"resolved",...}}}
```
-->
````

No `.testmd/` directory, no external databases. The state is committed with the tests, visible in diffs, and invisible in markdown renderers.
