# Skill describes installation correctly

```yaml
watch:
  - ./testmd/SKILL.md
```

Verify the skill includes install instructions:
1. Contains the `curl | sh` install command from install.sh
2. Instructs the agent to install automatically if `testmd` not found (no asking)
3. Mentions `testmd init` for project setup

# Skill teaches ci fix workflow

```yaml
watch:
  - ./testmd/SKILL.md
```

The `ci fix` workflow is the primary use case. Verify the skill describes it correctly:
1. Run `testmd status` to find non-resolved tests
2. For each: `testmd get <id>` to read what to verify
3. Perform the actual verification (run commands, read code, compare)
4. If problem found and fixable — fix, re-verify, then `resolve`
5. If problem found but unfixable — `fail` with specific reason
6. If check passes — `resolve`
7. Explicitly states: never resolve without actual verification

# Skill teaches meaningful resolve/fail

```yaml
watch:
  - ./testmd/SKILL.md
```

Verify the skill explains how to resolve/fail properly:
1. For `outdated` tests: check changed files specifically, not the whole test
2. For `pending` tests: perform full verification per description
3. For `failed` tests: check if previous failure reason is now fixed
4. Fail messages must be specific (file, error, what was tried) — not just "test failed"
5. Each test is verified individually, not resolved in bulk

# Skill teaches TEST.md authoring

```yaml
watch:
  - ./testmd/SKILL.md
  - ../docs/specification.md
```

Verify the skill covers writing and editing TEST.md files:
1. Format: `# Title`, yaml block with `watch` (required), optional `id`, `each`, `combinations`
2. No frontmatter in TEST.md
3. Description should be step-by-step verification instructions
4. Watch patterns should be specific (not `**/*`)
5. After writing: run `testmd status` to verify parsing and file discovery
6. If 0 files matched — watch pattern is wrong, fix it
7. `each` and `combinations` are mutually exclusive
8. Variable syntax: `{var}` in watch patterns and descriptions

# Skill documents commands and format accurately

```yaml
watch:
  - ./testmd/SKILL.md
  - ../docs/cli.md
  - ../internal/cli/cli.go
```

Verify the skill lists all commands with correct syntax:
1. Commands: `init`, `status`, `resolve <id>`, `fail <id> <message>`, `get <id>`, `gc`, `ci`
2. Global flag: `--root` (not `--testmd`)
3. ID format: 18 hex chars, no dashes, prefix matching
4. State file: `.testmd.lock` (YAML, single file in root) — not TEST.md.lock, not JSON
5. Root discovery: `.testmd.yaml` / `.testmd.yml` (no .git fallback)
6. TEST.md auto-discovery under root
7. Report flags: `--report-md`, `--report-json` on status/ci

# Skill includes detailed documentation in references

```yaml
watch:
  - ./testmd/SKILL.md
  - ./testmd/references/specification.md
  - ./testmd/references/cli.md
  - ./testmd/references/examples.md
  - ./testmd/references/architecture.md
```

Verify the skill bundles detailed documentation for the agent:
1. `references/` directory exists with specification.md, cli.md, examples.md, architecture.md
2. Each file matches its counterpart in the project's `docs/` directory (run `diff docs/X skills/testmd/references/X`)
3. SKILL.md contains a "Detailed documentation" section that lists all reference files with guidance on when to read each one
4. Reference files are kept in sync — run `make sync-skill` and verify no diff

# Skill enforces correct tool usage

```yaml
watch:
  - ./testmd/SKILL.md
```

Verify the skill explicitly tells the agent what to use and what to avoid:
1. Always `testmd get <id>` — never read TEST.md directly for test details
2. Always `testmd status` — never parse files to discover tests
3. Never edit `.testmd.lock` manually
4. Never add frontmatter to TEST.md
5. Never resolve without performing the actual check described in the test
