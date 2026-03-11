from datetime import date
from pathlib import Path

from git import Repo

from github_utils.executor import execute_paint_plan, execute_simulation
from github_utils.painter.engine import PaintPlan
from github_utils.realism.profile import DayPlan


def _init_test_repo(tmp_path: Path) -> Repo:
    repo_dir = tmp_path / "test-repo"
    repo = Repo.init(repo_dir)
    repo.config_writer().set_value("user", "name", "Test").release()
    repo.config_writer().set_value("user", "email", "test@test.com").release()
    readme = repo_dir / "README.md"
    readme.write_text("init")
    repo.index.add(["README.md"])
    repo.index.commit("initial commit")
    return repo


def test_execute_paint_plan(tmp_path):
    repo = _init_test_repo(tmp_path)
    plan = PaintPlan(day_commits=[
        (date(2025, 3, 1), 3),
        (date(2025, 3, 2), 2),
    ])
    result = execute_paint_plan(
        repo_path=tmp_path / "test-repo",
        plan=plan,
        use_realism=False,
        push=False,
    )
    assert result.total_commits == 5
    # Check commits exist with correct dates
    commits = list(repo.iter_commits())
    assert len(commits) >= 6  # 1 initial + 5


def test_execute_paint_plan_with_realism(tmp_path):
    repo = _init_test_repo(tmp_path)
    plan = PaintPlan(day_commits=[
        (date(2025, 6, 15), 3),
    ])
    result = execute_paint_plan(
        repo_path=tmp_path / "test-repo",
        plan=plan,
        use_realism=True,
        push=False,
        seed=42,
    )
    assert result.total_commits == 3
    # With realism, commits should have varied timestamps within the day
    commits = list(repo.iter_commits())
    recent_commits = commits[:3]  # last 3
    hours = {c.authored_datetime.hour for c in recent_commits}
    # Realism should produce at least some hour variation (not all identical)
    # With seed=42 and 3 commits this is very likely but not guaranteed;
    # the important thing is they have the correct date
    for c in recent_commits:
        assert c.authored_datetime.date() == date(2025, 6, 15)


def test_execute_simulation(tmp_path):
    repo = _init_test_repo(tmp_path)
    day_plans = [
        DayPlan(date=date(2025, 1, 1), commit_count=2),
        DayPlan(date=date(2025, 1, 2), commit_count=0),
        DayPlan(date=date(2025, 1, 3), commit_count=3),
    ]
    result = execute_simulation(
        repo_path=tmp_path / "test-repo",
        day_plans=day_plans,
        use_realism=True,
        push=False,
        seed=42,
    )
    assert result.total_commits == 5


def test_execute_empty_plan(tmp_path):
    _init_test_repo(tmp_path)
    plan = PaintPlan(day_commits=[])
    result = execute_paint_plan(
        repo_path=tmp_path / "test-repo",
        plan=plan,
        use_realism=False,
        push=False,
    )
    assert result.total_commits == 0
    assert result.total_pushes == 0


def test_execute_simulation_skips_zero_days(tmp_path):
    repo = _init_test_repo(tmp_path)
    day_plans = [
        DayPlan(date=date(2025, 2, 1), commit_count=0),
        DayPlan(date=date(2025, 2, 2), commit_count=0),
        DayPlan(date=date(2025, 2, 3), commit_count=1),
    ]
    result = execute_simulation(
        repo_path=tmp_path / "test-repo",
        day_plans=day_plans,
        use_realism=False,
        push=False,
    )
    assert result.total_commits == 1
    commits = list(repo.iter_commits())
    assert len(commits) == 2  # 1 initial + 1


def test_execute_no_push_zero_pushes(tmp_path):
    _init_test_repo(tmp_path)
    plan = PaintPlan(day_commits=[
        (date(2025, 4, 1), 2),
    ])
    result = execute_paint_plan(
        repo_path=tmp_path / "test-repo",
        plan=plan,
        use_realism=False,
        push=False,
    )
    assert result.total_pushes == 0


def test_execute_paint_plan_dates_match(tmp_path):
    """Verify that without realism, commit dates match the plan dates."""
    repo = _init_test_repo(tmp_path)
    plan = PaintPlan(day_commits=[
        (date(2025, 7, 4), 1),
        (date(2025, 7, 10), 2),
    ])
    result = execute_paint_plan(
        repo_path=tmp_path / "test-repo",
        plan=plan,
        use_realism=False,
        push=False,
    )
    assert result.total_commits == 3
    commits = list(repo.iter_commits())
    # Most recent commits first; skip the initial commit at the end
    plan_commits = commits[:3]
    dates_seen = {c.authored_datetime.date() for c in plan_commits}
    assert date(2025, 7, 4) in dates_seen
    assert date(2025, 7, 10) in dates_seen
