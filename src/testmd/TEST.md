# CLI output is readable

```yaml
on_change:
  - ./report.py
  - ./cli.py
```

Run `testmd status` on a project with mixed statuses (resolved, failed, outdated, pending).
1. Grouping by test makes visual sense — easy to scan
2. Colors are distinct and meaningful (green=ok, red=fail, yellow=outdated, cyan=pending)
3. Piped output (`testmd status | cat`) has no ANSI garbage
4. Long labels don't break alignment beyond reason
5. Time-ago display ("2h ago", "3d ago") is intuitive

# Error messages are helpful

```yaml
on_change: ./cli.py
```

Trigger each error scenario and verify the message helps the user fix the problem:
1. No TEST.md found — message says where it searched
2. Missing `on_change` — message includes test title and line number
3. Invalid matrix variable — message names the undefined variable
4. Nonexistent test id — suggests checking `testmd status`
5. No tracebacks in any error case

# State diff is reviewable

```yaml
on_change: ./state.py
```

Resolve a few tests, then `git diff` the TEST.md.
1. The state JSON is formatted (indent=2), not a single line
2. Reviewer can see which tests changed status
3. File hashes are present but don't dominate the diff
4. The `<!-- State -->` wrapper makes the block collapsible in GitHub

# Reports are useful as CI artifacts

```yaml
on_change: ./report.py
```

Generate `--report-md` and `--report-json`, open them:
1. Markdown report renders correctly in GitHub/GitLab
2. Tables have correct alignment
3. JSON report is parseable by `jq` without issues
4. Both reports include all tests, not just failing ones
5. Summary counts match actual test counts

# Documentation matches code

```yaml
on_change: ./**
```

Code changed — check that docs are still in sync:
1. CLI flags and commands in `docs/cli.md` match actual `--help` output
2. Config fields in `docs/specification.md` match what the parser accepts
3. State format in `docs/specification.md` matches what `state.py` writes
4. Module table in `docs/architecture.md` matches actual file structure
5. README quick start still works end-to-end
