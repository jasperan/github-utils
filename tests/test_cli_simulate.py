from typer.testing import CliRunner
from github_utils.cli import app

runner = CliRunner()


def test_simulate_dry_run():
    result = runner.invoke(app, ["simulate", "--days", "30", "--dry-run"])
    assert result.exit_code == 0
    assert "commit" in result.output.lower()


def test_simulate_with_profile():
    result = runner.invoke(app, ["simulate", "--days", "30", "--profile", "night-owl", "--dry-run"])
    assert result.exit_code == 0


def test_simulate_help():
    result = runner.invoke(app, ["simulate", "--help"])
    assert result.exit_code == 0


def test_simulate_with_seed():
    result = runner.invoke(app, ["simulate", "--days", "7", "--seed", "42", "--dry-run"])
    assert result.exit_code == 0


def test_simulate_no_dry_run():
    result = runner.invoke(app, ["simulate", "--days", "7"])
    assert result.exit_code == 0
    assert "not yet implemented" in result.output.lower()
