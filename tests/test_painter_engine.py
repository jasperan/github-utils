# tests/test_painter_engine.py
from datetime import date
from github_utils.painter.engine import PaintPlan, create_paint_plan
from github_utils.painter.grid import ContributionGrid
from github_utils.api.client import ContributionData


def _mock_contribution_data():
    return ContributionData(
        total=500,
        day_counts={f"2025-{m:02d}-{d:02d}": 5 for m in range(1, 4) for d in range(1, 29)},
    )


def test_create_paint_plan_from_grid():
    grid = ContributionGrid.empty(anchor_date=date(2025, 6, 15))
    grid.set(0, 51, 4)
    grid.set(3, 25, 2)
    calibration = _mock_contribution_data()
    plan = create_paint_plan(grid, calibration)
    assert isinstance(plan, PaintPlan)
    assert len(plan.day_commits) > 0
    assert plan.total_commits > 0


def test_paint_plan_dates_are_valid():
    grid = ContributionGrid.for_date(date(2025, 6, 15))
    grid.set(0, 51, 1)
    calibration = _mock_contribution_data()
    plan = create_paint_plan(grid, calibration)
    for day_date, count in plan.day_commits:
        assert isinstance(day_date, date)
        assert count > 0


def test_paint_plan_empty_grid():
    grid = ContributionGrid.empty()
    calibration = _mock_contribution_data()
    plan = create_paint_plan(grid, calibration)
    assert plan.total_commits == 0
    assert len(plan.day_commits) == 0
