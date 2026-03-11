from typer.testing import CliRunner
from github_utils.cli import app

runner = CliRunner()


def test_legacy_help():
    result = runner.invoke(app, ["legacy", "--help"])
    assert result.exit_code == 0
    assert "commit" in result.output.lower()


def test_legacy_commit_dry_run():
    result = runner.invoke(app, ["legacy", "commit", "100", "--dry-run"])
    assert result.exit_code == 0


def test_legacy_commit_no_dry_run():
    result = runner.invoke(app, ["legacy", "commit", "500"])
    assert result.exit_code == 0
    assert "not yet implemented" in result.output.lower()


def test_legacy_branch_dry_run():
    result = runner.invoke(app, ["legacy", "branch", "inf", "--dry-run"])
    assert result.exit_code == 0
    assert "legacy mode" in result.output.lower()


def test_legacy_branch_no_dry_run():
    result = runner.invoke(app, ["legacy", "branch", "inf"])
    assert result.exit_code == 0
