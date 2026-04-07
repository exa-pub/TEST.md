# Go implementation works correctly

```yaml
watch:
  - ./internal/**
  - ./cmd/**
```

Go code changed — verify it works correctly:
1. Go tests pass: `go test ./internal/...`
2. Build succeeds: `go build -o ./bin/ ./cmd/...`
3. Run `./testmd-go status` on a sample TEST.md and verify output is correct

# Documentation is accurate

```yaml
watch:
  - ./docs/specification.md
  - ./docs/cli.md
  - ./docs/examples.md
  - ./docs/architecture.md
  - ./README.md
```

Read through each doc and verify:
1. All documented commands actually work as described
2. All examples are copy-pasteable and produce expected output
3. No references to removed features or old behavior
4. Architecture doc matches actual module structure

# Agent skill is up to date

```yaml
watch:
  - ./skills/testmd/SKILL.md
  - ./docs/specification.md
  - ./docs/cli.md
  - ./internal/cli/cli.go
```

The agent skill (`skills/testmd/SKILL.md`) must accurately describe the current CLI:
1. All commands and flags match the implementation
2. ID format matches specification (18 hex, no dashes, prefix matching)
3. State file format and location are correct (.testmd.lock, YAML)
4. Project configuration described correctly (.testmd.yaml, no frontmatter)
5. No references to removed features (include, per-file lock files, JSON state, --testmd flag)

