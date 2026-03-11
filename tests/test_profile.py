# tests/test_profile.py
from datetime import date
from github_utils.realism.profile import ProfileEngine, DayPlan

def test_default_profile_generates_plans():
    engine = ProfileEngine(seed=42)
    plans = engine.generate_plan(start_date=date(2025, 1, 1), end_date=date(2025, 3, 31))
    assert len(plans) == 90
    assert all(isinstance(p, DayPlan) for p in plans)

def test_plans_have_varied_commit_counts():
    engine = ProfileEngine(seed=42)
    plans = engine.generate_plan(date(2025, 1, 1), date(2025, 3, 31))
    counts = [p.commit_count for p in plans]
    assert min(counts) == 0
    assert max(counts) > 5
    unique_counts = set(counts)
    assert len(unique_counts) > 3

def test_weekday_pattern():
    engine = ProfileEngine(seed=42)
    plans = engine.generate_plan(date(2025, 1, 1), date(2025, 6, 30))
    weekday_avg = sum(p.commit_count for p in plans if p.date.weekday() < 5) / max(1, sum(1 for p in plans if p.date.weekday() < 5))
    weekend_avg = sum(p.commit_count for p in plans if p.date.weekday() >= 5) / max(1, sum(1 for p in plans if p.date.weekday() >= 5))
    assert weekday_avg > weekend_avg

def test_named_profile_night_owl():
    engine = ProfileEngine.from_name("night-owl", seed=42)
    plans = engine.generate_plan(date(2025, 1, 1), date(2025, 1, 31))
    assert len(plans) == 31

def test_named_profile_weekend_warrior():
    engine = ProfileEngine.from_name("weekend-warrior", seed=42)
    plans = engine.generate_plan(date(2025, 1, 1), date(2025, 3, 31))
    weekend_avg = sum(p.commit_count for p in plans if p.date.weekday() >= 5) / max(1, sum(1 for p in plans if p.date.weekday() >= 5))
    assert weekend_avg > 2

def test_seed_reproducibility():
    e1 = ProfileEngine(seed=77)
    e2 = ProfileEngine(seed=77)
    p1 = e1.generate_plan(date(2025, 1, 1), date(2025, 1, 31))
    p2 = e2.generate_plan(date(2025, 1, 1), date(2025, 1, 31))
    assert [p.commit_count for p in p1] == [p.commit_count for p in p2]
