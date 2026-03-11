# tests/test_temporal.py
from datetime import date, datetime, timezone
from github_utils.realism.temporal import TemporalEngine

def test_generate_timestamps_count():
    engine = TemporalEngine(timezone_name="UTC")
    timestamps = engine.generate_timestamps(date(2025, 3, 11), count=5)
    assert len(timestamps) == 5

def test_timestamps_are_on_correct_date():
    engine = TemporalEngine(timezone_name="UTC")
    target = date(2025, 6, 15)
    timestamps = engine.generate_timestamps(target, count=10)
    for ts in timestamps:
        assert ts.date() == target

def test_timestamps_are_sorted():
    engine = TemporalEngine(timezone_name="UTC")
    timestamps = engine.generate_timestamps(date(2025, 3, 11), count=20)
    for i in range(len(timestamps) - 1):
        assert timestamps[i] <= timestamps[i + 1]

def test_timestamps_favor_business_hours():
    engine = TemporalEngine(timezone_name="UTC", seed=42)
    timestamps = engine.generate_timestamps(date(2025, 3, 11), count=100)
    business_hours = sum(1 for ts in timestamps if 9 <= ts.hour <= 18)
    assert business_hours > 60

def test_seed_reproducibility():
    engine1 = TemporalEngine(timezone_name="UTC", seed=123)
    engine2 = TemporalEngine(timezone_name="UTC", seed=123)
    ts1 = engine1.generate_timestamps(date(2025, 3, 11), count=10)
    ts2 = engine2.generate_timestamps(date(2025, 3, 11), count=10)
    assert ts1 == ts2
