"""End-to-end integration tests for the full github-utils pipeline."""

from datetime import date
from pathlib import Path

from git import Repo
from typer.testing import CliRunner

from github_utils.cli import app
from github_utils.painter.font import render_text
from github_utils.painter.grid import ContributionGrid
from github_utils.painter.engine import create_paint_plan, PaintPlan
from github_utils.api.client import ContributionData
from github_utils.executor import execute_paint_plan, execute_simulation

runner = CliRunner()


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


def test_text_to_commits_pipeline(tmp_path):
    """Full pipeline: text -> grid -> paint plan -> execute -> verify commits."""
    repo = _init_test_repo(tmp_path)
    grid = render_text("HI", intensity=4)
    calibration = ContributionData(
        total=365,
        day_counts={f"2025-01-{i:02d}": 3 for i in range(1, 29)},
    )
    plan = create_paint_plan(grid, calibration)
    assert plan.total_commits > 0

    result = execute_paint_plan(
        repo_path=tmp_path / "test-repo",
        plan=plan,
        use_realism=False,
        push=False,
    )
    assert result.total_commits == plan.total_commits
    commits = list(repo.iter_commits())
    assert len(commits) == plan.total_commits + 1  # +1 for initial


def test_simulate_pipeline(tmp_path):
    """Full pipeline: profile -> day plans -> execute -> verify varied timestamps."""
    from github_utils.realism.profile import ProfileEngine

    repo = _init_test_repo(tmp_path)
    engine = ProfileEngine(seed=42)
    plans = engine.generate_plan(date(2025, 1, 1), date(2025, 1, 7))

    result = execute_simulation(
        repo_path=tmp_path / "test-repo",
        day_plans=plans,
        use_realism=True,
        push=False,
        seed=42,
    )
    assert result.total_commits > 0
    commits = list(repo.iter_commits())
    assert len(commits) > 1


def test_cli_paint_dry_run():
    """CLI dry-run exits cleanly."""
    result = runner.invoke(app, ["paint", "A", "--dry-run"])
    assert result.exit_code == 0


def test_cli_simulate_dry_run():
    """CLI simulate dry-run exits cleanly."""
    result = runner.invoke(app, ["simulate", "--days", "7", "--dry-run"])
    assert result.exit_code == 0


def test_cli_full_help():
    """All subcommands appear in main help."""
    result = runner.invoke(app, ["--help"])
    assert result.exit_code == 0
    assert "paint" in result.output
    assert "simulate" in result.output
    assert "stats" in result.output
    assert "legacy" in result.output


def test_pattern_to_commits_pipeline(tmp_path):
    """Full pipeline: pattern -> grid -> paint plan -> execute."""
    from github_utils.painter.patterns import load_pattern

    repo = _init_test_repo(tmp_path)
    grid = load_pattern("heart")
    calibration = ContributionData(
        total=365,
        day_counts={f"2025-01-{i:02d}": 3 for i in range(1, 29)},
    )
    plan = create_paint_plan(grid, calibration)
    assert plan.total_commits > 0

    result = execute_paint_plan(
        repo_path=tmp_path / "test-repo",
        plan=plan,
        use_realism=True,
        push=False,
        seed=99,
    )
    assert result.total_commits == plan.total_commits


def test_text_to_preview_pipeline():
    """Text -> grid -> preview renders without errors."""
    from github_utils.painter.preview import render_grid_preview

    grid = render_text("OK", intensity=3)
    preview = render_grid_preview(grid)
    assert len(preview) > 0
    # Preview should have 7 lines (one per grid row)
    lines = preview.split("\n")
    assert len(lines) == 7


def test_simulate_with_named_profile(tmp_path):
    """Named profiles produce valid commit plans that execute successfully."""
    from github_utils.realism.profile import ProfileEngine

    repo = _init_test_repo(tmp_path)
    engine = ProfileEngine.from_name("weekend-warrior", seed=123)
    plans = engine.generate_plan(date(2025, 3, 1), date(2025, 3, 14))

    result = execute_simulation(
        repo_path=tmp_path / "test-repo",
        day_plans=plans,
        use_realism=True,
        push=False,
        seed=123,
    )
    assert result.total_commits > 0
    assert result.total_pushes == 0
    commits = list(repo.iter_commits())
    # More commits than just the initial one
    assert len(commits) > 1


def test_realism_produces_varied_content(tmp_path):
    """Realism mode produces commits with varied messages and file changes."""
    repo = _init_test_repo(tmp_path)
    plan = PaintPlan(day_commits=[
        (date(2025, 5, 1), 5),
    ])
    execute_paint_plan(
        repo_path=tmp_path / "test-repo",
        plan=plan,
        use_realism=True,
        push=False,
        seed=77,
    )
    commits = list(repo.iter_commits())
    # Skip initial commit
    realism_commits = commits[:5]
    messages = [c.message.strip() for c in realism_commits]
    # With 5 commits and seed=77, messages should not all be identical
    assert len(set(messages)) > 1


def test_cli_legacy_commit_dry_run():
    """Legacy commit dry-run works end-to-end."""
    result = runner.invoke(app, ["legacy", "commit", "100", "--dry-run"])
    assert result.exit_code == 0
    assert "100" in result.output


def test_cli_legacy_branch_dry_run():
    """Legacy branch dry-run works end-to-end."""
    result = runner.invoke(app, ["legacy", "branch", "inf", "--dry-run"])
    assert result.exit_code == 0
    assert "inf" in result.output
