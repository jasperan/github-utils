from typer.testing import CliRunner
from github_utils.cli import app
from github_utils import __version__

runner = CliRunner()

def test_cli_shows_help():
    result = runner.invoke(app, ["--help"])
    assert result.exit_code == 0
    assert "GitHub contribution graph painter" in result.output

def test_cli_bare_invocation_shows_help():
    result = runner.invoke(app, [])
    assert result.exit_code == 0
    assert "GitHub contribution graph painter" in result.output

def test_cli_version():
    result = runner.invoke(app, ["--version"])
    assert result.exit_code == 0
    assert __version__ in result.output
