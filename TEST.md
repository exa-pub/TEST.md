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

<!-- State
```testmd
{
  "tests": {
    "7a4284-e3b0c4": {
      "content_hash": "d87e8473fd964d98a5f24f535172043af9bd14f191cfef08bd7a25e177d43ba2",
      "failed_at": null,
      "files": {
        "README.md": "42841c99e244ed325b0e8e50e12a30e9f6bd1bcf5bea365d5b2afc6fa58aebd6",
        "docs/architecture.md": "bf31e22d52e6314d4e4ed026d387cf76e01eb45bf8d82e932b0e22dc89a920c4",
        "docs/cli.md": "58906b73f4d2f62296c59c2cc80fdd4d38a1184fd41c0d0265107f63d6342a85",
        "docs/examples.md": "c385819d26fee494ab5ea5b3914aff79ffe870738299f1d5e9ac6f3722f35d09",
        "docs/specification.md": "8ca6de64ca06c9a79e745c697ea2c869a221c08129c1cf2079010dca69938701"
      },
      "labels": {},
      "message": null,
      "resolved_at": "2026-04-06T21:39:22.992285+00:00",
      "status": "resolved",
      "title": "Documentation is accurate"
    }
  },
  "version": 1
}
```
-->
