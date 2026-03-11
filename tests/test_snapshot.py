import json
from pathlib import Path
from github_utils.analytics.snapshot import Snapshot, compare_snapshots

def test_snapshot_save_load(tmp_path):
    snap = Snapshot(timestamp="2025-03-11T10:00:00Z", total_contributions=500, current_streak=15, longest_streak=30, active_days=200)
    snap.save(tmp_path)
    files = list(tmp_path.glob("*.json"))
    assert len(files) == 1
    loaded = Snapshot.load(files[0])
    assert loaded.total_contributions == 500
    assert loaded.current_streak == 15

def test_compare_snapshots():
    before = Snapshot(timestamp="2025-01-01T00:00:00Z", total_contributions=500, current_streak=10, longest_streak=20, active_days=150)
    after = Snapshot(timestamp="2025-03-11T00:00:00Z", total_contributions=1200, current_streak=45, longest_streak=45, active_days=220)
    deltas = compare_snapshots(before, after)
    assert deltas["total_contributions"] == 700
    assert deltas["current_streak"] == 35
    assert deltas["active_days"] == 70
