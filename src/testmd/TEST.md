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

<!-- State
```testmd
{
  "tests": {
    "168c6f-e3b0c4": {
      "content_hash": "80abe734bb110f0814f9456ccb21db5ca4182114ad66fc18565d8a2fe9f6b71d",
      "failed_at": null,
      "files": {
        "src/testmd/cli.py": "b569ee631a03c1b63d431e7cc40a72aaa80bf74eedb80c294940b9647ff77553",
        "src/testmd/report.py": "1b7a0b467a01c25e5964b56187e61e6c820410c8a895b7314f4810ccf112bb7c"
      },
      "labels": {},
      "message": null,
      "resolved_at": "2026-04-06T21:36:31.105426+00:00",
      "status": "resolved",
      "title": "CLI output is readable"
    },
    "55f3cc-e3b0c4": {
      "content_hash": "597d336926311a9ff0977a12884c082888cf71d9df97f03002e4bc9e2fd65401",
      "failed_at": null,
      "files": {
        "src/testmd/cli.py": "b569ee631a03c1b63d431e7cc40a72aaa80bf74eedb80c294940b9647ff77553"
      },
      "labels": {},
      "message": null,
      "resolved_at": "2026-04-06T21:35:30.747887+00:00",
      "status": "resolved",
      "title": "Error messages are helpful"
    },
    "6307ac-e3b0c4": {
      "content_hash": "98abcedd31740e32647e288d8f9c0322bb388cf308cb5abc836ca99c6100e173",
      "failed_at": null,
      "files": {
        "src/testmd/state.py": "7e1b426792a1f1fef6ee9d3185017929e9c0e51bb02abf80baff4e6b1f824136"
      },
      "labels": {},
      "message": null,
      "resolved_at": "2026-04-06T21:40:09.250020+00:00",
      "status": "resolved",
      "title": "State diff is reviewable"
    },
    "966efd-e3b0c4": {
      "content_hash": "344cfe5d34decacdd0c460d6a8f5e93e7f7af4aa6aca287d331e4c9402dfa3a7",
      "failed_at": null,
      "files": {
        "src/testmd/report.py": "1b7a0b467a01c25e5964b56187e61e6c820410c8a895b7314f4810ccf112bb7c"
      },
      "labels": {},
      "message": null,
      "resolved_at": "2026-04-06T21:36:08.530259+00:00",
      "status": "resolved",
      "title": "Reports are useful as CI artifacts"
    },
    "f3663c-e3b0c4": {
      "content_hash": "3261f50562f04dd6742ce5ee6ad34d8803077d208a9441c0fde17da5fe56c0a6",
      "failed_at": null,
      "files": {
        "src/testmd/__init__.py": "3f8ac0eaaf86d35f2b93f63eacb541d6f20a69ebc68c5affdf42b1016872c0fe",
        "src/testmd/cli.py": "b569ee631a03c1b63d431e7cc40a72aaa80bf74eedb80c294940b9647ff77553",
        "src/testmd/hashing.py": "5c406850aa878191fa716c0116a8de7338c9fc45f600e82d259c5c32cd82a007",
        "src/testmd/models.py": "80cea856a9911183a9a390c8720e5b5b105a4d8513c8bcc4089123874db8a5f6",
        "src/testmd/parser.py": "36a1b6997b297b7b5207c45fd5f7ef70eb2412031cc3bfccfed5a48ef06c0c21",
        "src/testmd/patterns.py": "0c01c3ef3ae968bea814503d2dd7a82ce578e1848e42f0ea641f88531fc37cfc",
        "src/testmd/report.py": "1b7a0b467a01c25e5964b56187e61e6c820410c8a895b7314f4810ccf112bb7c",
        "src/testmd/resolver.py": "e1af32a06cb7a6e74bd909dcc0487c9d0d12bff8ea38ad9221a3e623b91f055b",
        "src/testmd/state.py": "7e1b426792a1f1fef6ee9d3185017929e9c0e51bb02abf80baff4e6b1f824136"
      },
      "labels": {},
      "message": null,
      "resolved_at": "2026-04-06T21:40:12.358937+00:00",
      "status": "resolved",
      "title": "Documentation matches code"
    }
  },
  "version": 1
}
```
-->
