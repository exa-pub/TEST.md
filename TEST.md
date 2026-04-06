# Parser correctness

```yaml
on_change: ./src/testmd/parser.py
```

Verify that TEST.md parsing handles edge cases:
1. Multiple tests in one file
2. Missing yaml block raises an error
3. on_change as string and as list both work

# Pattern expansion

```yaml
on_change: ./src/testmd/patterns.py
```

Verify label expansion with $variables:
1. Single $var enumerates directory entries
2. Nested $var1/$var2 produces cartesian combinations
3. Patterns without $vars return a single empty label set

# CLI commands

```yaml
on_change:
  - ./src/testmd/cli.py
  - ./src/testmd/report.py
```

Verify all CLI commands work end-to-end:
1. `testmd status` shows correct output
2. `testmd resolve` / `testmd fail` update state
3. `testmd ci` exits 1 when tests are pending
4. `testmd gc` removes orphaned records

# Service health

```yaml
on_change: ./services/$name/**
```

Verify that service `$name` starts and responds to healthcheck.

<!-- State
```testmd
{
  "version": 1,
  "tests": {
    "ed4be2-e3b0c4": {
      "title": "Service health",
      "labels": {},
      "content_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      "files": {},
      "status": "resolved",
      "resolved_at": "2026-04-06T21:02:35.001355+00:00",
      "failed_at": null,
      "message": null
    }
  }
}
```
-->
