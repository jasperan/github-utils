"""Snapshot persistence for contribution analytics.

Save, load, and compare point-in-time snapshots of GitHub contribution
metrics so users can track progress over days, weeks, or months.
"""
from __future__ import annotations

import json
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Dict


@dataclass
class Snapshot:
    """Point-in-time capture of contribution statistics."""

    timestamp: str
    total_contributions: int
    current_streak: int
    longest_streak: int
    active_days: int

    def save(self, directory: Path) -> Path:
        """Persist this snapshot as a JSON file in *directory*.

        The filename is derived from the timestamp (colons replaced with
        hyphens so it is filesystem-safe on all platforms).

        Returns the path to the written file.
        """
        directory.mkdir(parents=True, exist_ok=True)
        safe_ts = self.timestamp.replace(":", "-")
        path = directory / f"snapshot-{safe_ts}.json"
        with open(path, "w") as f:
            json.dump(asdict(self), f, indent=2)
        return path

    @classmethod
    def load(cls, path: Path) -> Snapshot:
        """Deserialise a snapshot from a JSON file."""
        with open(path) as f:
            data = json.load(f)
        return cls(**data)


def compare_snapshots(before: Snapshot, after: Snapshot) -> Dict[str, int]:
    """Return per-field deltas between two snapshots.

    Positive values indicate growth from *before* to *after*.
    """
    return {
        "total_contributions": after.total_contributions - before.total_contributions,
        "current_streak": after.current_streak - before.current_streak,
        "longest_streak": after.longest_streak - before.longest_streak,
        "active_days": after.active_days - before.active_days,
    }
