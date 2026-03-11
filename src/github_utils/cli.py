from typing import Optional

import typer

from github_utils import __version__
from github_utils.commands.paint import paint_command

app = typer.Typer(
    name="github-utils",
    help="GitHub contribution graph painter, activity simulator, and analytics toolkit.",
)

app.command(name="paint")(paint_command)


def _version_callback(value: bool):
    if value:
        typer.echo(f"github-utils {__version__}")
        raise typer.Exit()


@app.callback(invoke_without_command=True)
def main(
    ctx: typer.Context,
    version: Optional[bool] = typer.Option(None, "--version", "-V", callback=_version_callback, is_eager=True, help="Show version and exit."),
):
    if ctx.invoked_subcommand is None:
        typer.echo(ctx.get_help())

if __name__ == "__main__":
    app()
