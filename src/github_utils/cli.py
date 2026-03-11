import typer

app = typer.Typer(
    name="github-utils",
    help="GitHub contribution graph painter, activity simulator, and analytics toolkit.",
)

@app.callback(invoke_without_command=True)
def main(ctx: typer.Context):
    if ctx.invoked_subcommand is None:
        typer.echo(ctx.get_help())

if __name__ == "__main__":
    app()
