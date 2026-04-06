from __future__ import annotations

import re
from itertools import product as cart_product
from pathlib import Path, PurePosixPath

import pathspec


def find_label_vars(pattern: str) -> list[str]:
    """Extract $variable names from a pattern."""
    return re.findall(r"\$([a-zA-Z_]\w*)", pattern)


def load_ignorefile(root: Path, ignorefile: str) -> pathspec.PathSpec | None:
    """Load a gitignore-style file and return a PathSpec matcher."""
    path = root / ignorefile
    if not path.exists():
        return None
    lines = path.read_text().splitlines()
    return pathspec.PathSpec.from_lines("gitwildmatch", lines)


# ---------------------------------------------------------------------------
# Auto-discovery from on_change (simple mode, no matrix)
# ---------------------------------------------------------------------------


def enumerate_labels(
    root: Path, patterns: list[str], ignore: pathspec.PathSpec | None = None
) -> list[dict[str, str]]:
    """Discover all label combinations by scanning the filesystem."""
    all_combos: list[dict[str, str]] = []

    for pat in patterns:
        if not find_label_vars(pat):
            continue
        for combo in _enumerate_pattern(root, pat, ignore):
            if combo not in all_combos:
                all_combos.append(combo)

    return all_combos or [{}]


# ---------------------------------------------------------------------------
# Matrix expansion
# ---------------------------------------------------------------------------


def expand_matrix(
    root: Path, matrix: list[dict], ignore: pathspec.PathSpec | None = None
) -> list[dict[str, str]]:
    """Expand matrix entries into label combinations (union of entries)."""
    all_combos: list[dict[str, str]] = []
    for entry in matrix:
        for combo in _expand_entry(root, entry, ignore):
            if combo not in all_combos:
                all_combos.append(combo)
    return all_combos or [{}]


def _expand_entry(
    root: Path, entry: dict, ignore: pathspec.PathSpec | None
) -> list[dict[str, str]]:
    """Expand one matrix entry (match × const)."""
    match_combos: list[dict[str, str]] = [{}]
    if "match" in entry:
        patterns = entry["match"]
        if isinstance(patterns, str):
            patterns = [patterns]
        discovered: list[dict[str, str]] = []
        for pat in patterns:
            for combo in _enumerate_pattern(root, pat, ignore):
                if combo not in discovered:
                    discovered.append(combo)
        if discovered:
            match_combos = discovered

    const_combos: list[dict[str, str]] = [{}]
    if "const" in entry:
        const_combos = _expand_const(entry["const"])

    return [{**mc, **cc} for mc in match_combos for cc in const_combos]


def _expand_const(const: dict[str, list]) -> list[dict[str, str]]:
    """Cartesian product of const values."""
    keys = sorted(const.keys())
    values = [c if isinstance(c, list) else [c] for c in (const[k] for k in keys)]
    return [
        {k: str(v) for k, v in zip(keys, combo)}
        for combo in cart_product(*values)
    ]


# ---------------------------------------------------------------------------
# Pattern → filesystem enumeration
# ---------------------------------------------------------------------------


def _enumerate_pattern(
    root: Path, pattern: str, ignore: pathspec.PathSpec | None
) -> list[dict[str, str]]:
    pat = pattern[2:] if pattern.startswith("./") else pattern
    parts = list(PurePosixPath(pat).parts)
    return _walk(root, root, parts, {}, ignore)


def _walk(
    root: Path,
    base: Path,
    parts: list[str],
    labels: dict[str, str],
    ignore: pathspec.PathSpec | None,
) -> list[dict[str, str]]:
    """Walk path parts, enumerating $var segments from the filesystem."""
    if not parts:
        return [dict(labels)]

    part, rest = parts[0], parts[1:]

    if part.startswith("$"):
        var_name = part[1:]
        if not base.is_dir():
            return []
        results: list[dict[str, str]] = []
        for entry in sorted(base.iterdir()):
            if entry.name.startswith("."):
                continue
            rel = str(entry.relative_to(root))
            if ignore and ignore.match_file(rel + "/" if entry.is_dir() else rel):
                continue
            results.extend(
                _walk(root, entry, rest, {**labels, var_name: entry.name}, ignore)
            )
        return results

    if "*" in part or "?" in part:
        return [dict(labels)]

    return _walk(root, base / part, rest, labels, ignore)


# ---------------------------------------------------------------------------
# Pattern → file list
# ---------------------------------------------------------------------------


def resolve_files(
    root: Path,
    pattern: str,
    labels: dict[str, str],
    ignore: pathspec.PathSpec | None = None,
) -> list[str]:
    """Substitute labels into pattern and glob for matching files."""
    resolved = pattern
    for var, val in labels.items():
        resolved = resolved.replace(f"${var}", val)

    if resolved.startswith("./"):
        resolved = resolved[2:]

    # "dir/**" doesn't match files in pathlib — normalize to "dir/**/*"
    if resolved.endswith("/**"):
        resolved += "/*"

    files = [
        str(p.relative_to(root)) for p in root.glob(resolved) if p.is_file()
    ]

    if ignore:
        files = [f for f in files if not ignore.match_file(f)]

    return sorted(files)
