# testmd — Examples

## Simple: single file, no labels

```markdown
# Login page renders

```yaml
watch: ./src/auth/**
```

1. Open /login
2. Verify the form has email and password fields
3. Verify "Forgot password" link is present
```

```
$ testmd status
Login page renders
  … a1b2c3-e3b0c4  pending

$ testmd resolve a1b2c3
Resolved: Login page renders

$ testmd status
Login page renders
  ✓ a1b2c3-e3b0c4  resolved  (5s ago)
```

## Labels from filesystem (each with glob)

```markdown
# {service} healthcheck

```yaml
each:
  service: ./services/*/
watch: ./services/{service}/**
```

Verify `{service}` responds to GET /health with 200.
```

With filesystem:
```
services/
  auth/
    main.go
  billing/
    main.go
  gateway/
    main.go
```

```
$ testmd status
{service} healthcheck
  … ed4be2-fe0c31  service=auth     pending
  … ed4be2-c9054d  service=billing  pending
  … ed4be2-ab1234  service=gateway  pending

$ testmd resolve ed4be2
Resolved: auth healthcheck (service=auth)
Resolved: billing healthcheck (service=billing)
Resolved: gateway healthcheck (service=gateway)
```

Adding a new service (`services/payments/`) automatically creates a new pending test instance.

## Each with explicit values

```markdown
# API compatibility

```yaml
each:
  version: [v1, v2, v3]
watch: ./api/**
```

Verify the API contract for version `{version}`.
```

```
$ testmd status
API compatibility
  … abc123-111111  version=v1  pending
  … abc123-222222  version=v2  pending
  … abc123-333333  version=v3  pending
```

## Each with glob + explicit (cartesian product)

```markdown
# Deploy smoke test

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
```

## Combinations for irregular sets

```markdown
# Database migrations

```yaml
combinations:
  - db: [postgres, mysql]
    suite: [full]
  - db: [sqlite]
    suite: [basic]
watch: ./migrations/{db}/**
```

Run `{suite}` migration tests for `{db}`.
```

## Multiple watch patterns

```markdown
# Config validation

```yaml
each:
  env: ./config/*/
watch:
  - ./config/{env}/**
  - ./schema/config.json
```

Verify that `{env}` config validates against the JSON schema.
```

Changes to any config directory OR `schema/config.json` will mark the test as outdated.

## Include files

Root `TEST.md`:
```yaml
---
include: [tests/integration/TEST.md, tests/e2e/TEST.md]
---

# Unit test sanity

```yaml
watch: ./src/**
```

Run `make test` and verify all unit tests pass.
```

`tests/integration/TEST.md`:
```markdown
# Integration: database

```yaml
watch: ./src/db/**
```

Run integration tests against a real database.
```

```
$ testmd status
Unit test sanity
  … aaa111-e3b0c4  pending

Integration: database
  … bbb222-e3b0c4  pending
```

Each file stores its own state in its own lock file. Resolving "Integration: database" writes state to `tests/integration/TEST.md.lock`, not to the root.

## Custom ignorefile

```yaml
---
ignorefile: .testmdignore
---
```

`.testmdignore`:
```gitignore
__pycache__/
*.pyc
node_modules/
dist/
*.min.js
```

If `ignorefile` is not specified, `.gitignore` is used by default.

## CI integration

```yaml
# GitHub Actions
- name: Check manual tests
  run: testmd ci --report-md test-report.md

# GitLab CI
test:manual:
  script:
    - testmd ci --report-json report.json
  artifacts:
    paths: [report.json]
```

## Explicit ID for stable references

```markdown
# OAuth flow

```yaml
id: oauth
watch: ./services/auth/**
```

This test has a stable id `oauth-e3b0c4` that won't change if the title is renamed.
```
