from typer.testing import CliRunner
from github_utils.cli import app

runner = CliRunner()

def test_cli_shows_help():
    result = runner.invoke(app, ["--help"])
    assert result.exit_code == 0
    assert "GitHub contribution graph painter" in result.output
