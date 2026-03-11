import json
import tempfile
from pathlib import Path

from typer.testing import CliRunner
from github_utils.cli import app

runner = CliRunner()


def test_stats_help():
    result = runner.invoke(app, ["stats", "--help"])
    assert result.exit_code == 0


def test_stats_snapshot():
    result = runner.invoke(app, ["stats", "snapshot"])
    assert result.exit_code == 0
    assert "token" in result.output.lower()


def test_stats_compare():
    before_data = {
        "timestamp": "2024-01-01T00:00:00",
        "total_contributions": 100,
        "current_streak": 5,
        "longest_streak": 10,
        "active_days": 50,
    }
    after_data = {
        "timestamp": "2024-06-01T00:00:00",
        "total_contributions": 250,
        "current_streak": 12,
        "longest_streak": 15,
        "active_days": 80,
    }
    with tempfile.TemporaryDirectory() as tmpdir:
        before_path = Path(tmpdir) / "before.json"
        after_path = Path(tmpdir) / "after.json"
        before_path.write_text(json.dumps(before_data))
        after_path.write_text(json.dumps(after_data))

        result = runner.invoke(app, ["stats", "compare", "--before", str(before_path), "--after", str(after_path)])
        assert result.exit_code == 0
        assert "150" in result.output  # delta of total_contributions


def test_stats_compare_missing_file():
    result = runner.invoke(app, ["stats", "compare", "--before", "/nonexistent/before.json", "--after", "/nonexistent/after.json"])
    assert result.exit_code == 1
