# CLAUDE.md

## Project overview

testmd is a tool for tracking manual/semi-automated tests described in TEST.md files. It watches source files via hashing and tracks which tests need re-verification when code changes.

The canonical specification is in `docs/specification.md`. The architecture is described in `docs/architecture.md`. Read those before making changes.

## Current implementations

- **Python** — `src/testmd/` (reference implementation)

## Key principles

- testmd is a **multi-language project**. The Python implementation is the reference, but the tool may be rewritten in other languages. All implementations must follow the same specification.
- When modifying behavior, update `docs/specification.md` first, then the implementation(s).
- The specification and docs are **language-agnostic**. Never add Python-specific details to docs.
- Tests in `tests/` are Python-specific. Other implementations will have their own test suites.

## Architecture rules

- State is always stored inline in TEST.md (no external directories)
- The state block format is `<!-- State\n```testmd\n{json}\n```\n-->`
- Hashing must be deterministic: same files + same content = same hash
- Labels and files are always sorted before hashing or display
- Ignorefile defaults to `.gitignore`, parsed as gitignore format

## Commands

```
testmd [--testmd PATH] status [--report-md F] [--report-json F]
testmd [--testmd PATH] resolve <id>
testmd [--testmd PATH] fail <id> <message>
testmd [--testmd PATH] get <id>
testmd [--testmd PATH] gc
testmd [--testmd PATH] ci [--report-md F] [--report-json F]
```

## Running tests (Python)

```
pip install -e . --break-system-packages
python -m pytest tests/ -v
```
