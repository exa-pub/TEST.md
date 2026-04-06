---
include: [src/testmd/TEST.md]
---

# Documentation is accurate

```yaml
on_change:
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
