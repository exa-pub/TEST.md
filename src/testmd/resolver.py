from __future__ import annotations

import sys
from datetime import datetime, timezone
from pathlib import Path

import pathspec

from .hashing import hash_files, make_id
from .models import TestDefinition, TestInstance
from .patterns import (
    enumerate_labels,
    expand_matrix,
    find_label_vars,
    resolve_files,
)

StatusResult = tuple[TestInstance, str, dict | None]


def build_instances(
    root: Path,
    definitions: list[TestDefinition],
    ignore: pathspec.PathSpec | None = None,
) -> list[TestInstance]:
    """Expand definitions into concrete instances with computed hashes."""
    instances: list[TestInstance] = []

    for defn in definitions:
        on_change = _rebase_patterns(root, defn.source_file, defn.on_change)
        matrix = _rebase_matrix(root, defn.source_file, defn.matrix)

        if matrix:
            _validate_matrix_vars(defn)
            label_combos = expand_matrix(root, matrix, ignore)
        else:
            label_combos = enumerate_labels(root, on_change, ignore)

        for labels in label_combos:
            resolved_patterns: list[str] = []
            all_files: set[str] = set()

            for pat in on_change:
                resolved = pat
                for var, val in labels.items():
                    resolved = resolved.replace(f"${var}", val)
                resolved_patterns.append(resolved)
                all_files.update(resolve_files(root, pat, labels, ignore))

            # Exclude the source TEST.md itself — it contains the state
            # block which changes on every resolve, causing a loop.
            self_rel = str(defn.source_file.relative_to(root))
            all_files.discard(self_rel)

            matched = sorted(all_files)
            content_hash, file_hashes = hash_files(root, matched)
            tid = make_id(defn.title, defn.explicit_id, labels)

            instances.append(
                TestInstance(
                    id=tid,
                    definition=defn,
                    labels=labels,
                    resolved_patterns=resolved_patterns,
                    matched_files=matched,
                    content_hash=content_hash,
                    file_hashes=file_hashes,
                )
            )

    return instances


def compute_statuses(
    instances: list[TestInstance], state: dict
) -> list[StatusResult]:
    results: list[StatusResult] = []
    for inst in instances:
        rec = state["tests"].get(inst.id)
        if rec is None:
            status = "pending"
        elif rec["content_hash"] != inst.content_hash:
            status = "outdated"
        else:
            status = rec["status"]
        results.append((inst, status, rec))
    return results


def resolve_test(state: dict, inst: TestInstance) -> None:
    state["tests"][inst.id] = _make_record(inst, "resolved")
    state["tests"][inst.id]["resolved_at"] = _now()


def fail_test(state: dict, inst: TestInstance, message: str) -> None:
    state["tests"][inst.id] = _make_record(inst, "failed")
    state["tests"][inst.id]["failed_at"] = _now()
    state["tests"][inst.id]["message"] = message


def gc_state(state: dict, instances: list[TestInstance]) -> int:
    current_ids = {i.id for i in instances}
    orphans = [tid for tid in state["tests"] if tid not in current_ids]
    for tid in orphans:
        del state["tests"][tid]
    return len(orphans)


def find_instances(
    instances: list[TestInstance], query: str
) -> list[TestInstance]:
    """Find instances matching a full id, first-part, or prefix."""
    exact = [i for i in instances if i.id == query]
    if exact:
        return exact
    by_first = [i for i in instances if i.id.split("-")[0] == query]
    if by_first:
        return by_first
    return [i for i in instances if i.id.startswith(query)]


def changed_files(inst: TestInstance, record: dict | None) -> list[str]:
    if not record or "files" not in record:
        return list(inst.matched_files)
    old = record["files"]
    cur = inst.file_hashes
    changed = set()
    for f in cur:
        if f not in old or old[f] != cur[f]:
            changed.add(f)
    for f in old:
        if f not in cur:
            changed.add(f)
    return sorted(changed)


# ---------------------------------------------------------------------------
# Validation
# ---------------------------------------------------------------------------


def _validate_matrix_vars(defn: TestDefinition) -> None:
    on_change_vars: set[str] = set()
    for pat in defn.on_change:
        on_change_vars.update(find_label_vars(pat))

    matrix_vars: set[str] = set()
    for entry in defn.matrix:  # type: ignore[union-attr]
        if "match" in entry:
            patterns = entry["match"]
            if isinstance(patterns, str):
                patterns = [patterns]
            for pat in patterns:
                matrix_vars.update(find_label_vars(pat))
        if "const" in entry:
            matrix_vars.update(entry["const"].keys())

    undefined = on_change_vars - matrix_vars
    if undefined:
        raise ValueError(
            f"Test '{defn.title}': variables {undefined} in on_change "
            f"not defined in matrix"
        )

    unused = matrix_vars - on_change_vars
    if unused:
        print(
            f"Warning: test '{defn.title}': matrix variables {unused} "
            f"not used in on_change",
            file=sys.stderr,
        )


# ---------------------------------------------------------------------------
# Pattern rebasing
# ---------------------------------------------------------------------------


def _rebase_patterns(
    root: Path, source_file: Path, patterns: list[str]
) -> list[str]:
    """Adjust patterns from source_file-relative to root-relative."""
    source_dir = source_file.parent
    if source_dir == root:
        return patterns
    rel = source_dir.relative_to(root)
    return [
        f"./{rel}/{p[2:]}" if p.startswith("./") else f"./{rel}/{p}"
        for p in patterns
    ]


def _rebase_matrix(
    root: Path, source_file: Path, matrix: list[dict] | None
) -> list[dict] | None:
    if not matrix or source_file.parent == root:
        return matrix
    rebased = []
    for entry in matrix:
        new_entry = dict(entry)
        if "match" in entry:
            patterns = entry["match"]
            if isinstance(patterns, str):
                patterns = [patterns]
            new_entry["match"] = _rebase_patterns(root, source_file, patterns)
        rebased.append(new_entry)
    return rebased


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_record(inst: TestInstance, status: str) -> dict:
    return {
        "title": inst.definition.title,
        "labels": inst.labels,
        "content_hash": inst.content_hash,
        "files": inst.file_hashes,
        "status": status,
        "resolved_at": None,
        "failed_at": None,
        "message": None,
    }


def _now() -> str:
    return datetime.now(timezone.utc).isoformat()
