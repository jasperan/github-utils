from typer.testing import CliRunner
from github_utils.cli import app

runner = CliRunner()


def test_paint_preview_text():
    result = runner.invoke(app, ["paint", "HI", "--preview", "--dry-run"])
    assert result.exit_code == 0


def test_paint_preview_pattern():
    result = runner.invoke(app, ["paint", "--pattern", "wave", "--preview", "--dry-run"])
    assert result.exit_code == 0


def test_paint_dry_run_shows_plan():
    result = runner.invoke(app, ["paint", "A", "--dry-run"])
    assert result.exit_code == 0
    assert "commit" in result.output.lower()


def test_paint_help():
    result = runner.invoke(app, ["paint", "--help"])
    assert result.exit_code == 0


def test_paint_no_input_errors():
    result = runner.invoke(app, ["paint"])
    assert result.exit_code == 1


def test_paint_intensity():
    result = runner.invoke(app, ["paint", "X", "--intensity", "2", "--preview"])
    assert result.exit_code == 0


def test_paint_text_no_flags_shows_preview():
    result = runner.invoke(app, ["paint", "AB"])
    assert result.exit_code == 0
    assert "not yet implemented" in result.output.lower()
