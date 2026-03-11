from __future__ import annotations

from pathlib import Path
from typing import List

import yaml

from github_utils.painter.grid import ContributionGrid

_BUILTIN_DIR = Path(__file__).resolve().parent.parent.parent.parent / "patterns"


def list_builtin_patterns() -> List[str]:
    """Return sorted list of builtin pattern names (without extension)."""
    if not _BUILTIN_DIR.exists():
        return []
    return sorted(p.stem for p in _BUILTIN_DIR.glob("*.yaml"))


def load_pattern(name_or_path: str, anchor_date=None) -> ContributionGrid:
    """Load a pattern by builtin name or file path.

    Parameters
    ----------
    name_or_path:
        Either a builtin pattern name (e.g. "wave") or a filesystem path
        to a YAML pattern file.
    anchor_date:
        Optional anchor date for the resulting grid.

    Returns
    -------
    ContributionGrid populated with the pattern data.

    Raises
    ------
    FileNotFoundError
        If the name does not match a builtin pattern and is not a valid path.
    """
    path = Path(name_or_path)
    if path.exists() and path.suffix in (".yaml", ".yml"):
        return _load_from_file(path, anchor_date)

    builtin_path = _BUILTIN_DIR / f"{name_or_path}.yaml"
    if builtin_path.exists():
        return _load_from_file(builtin_path, anchor_date)

    raise FileNotFoundError(
        f"Pattern '{name_or_path}' not found as file or builtin pattern"
    )


def _load_from_file(path: Path, anchor_date=None) -> ContributionGrid:
    """Parse a YAML pattern file and return a ContributionGrid."""
    with open(path) as f:
        data = yaml.safe_load(f)
    grid_data = data["grid"]
    return ContributionGrid.from_list(grid_data, anchor_date)
