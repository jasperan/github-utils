# github-utils

A toolkit for painting text and patterns on your GitHub contribution graph, simulating realistic developer activity, and tracking contribution analytics.

## Installation

<!-- one-command-install -->
> **One-command install** — clone, configure, and run in a single step:
>
> ```bash
> curl -fsSL https://raw.githubusercontent.com/jasperan/github-utils/master/install.sh | bash
> ```
>
> <details><summary>Advanced options</summary>
>
> Override install location:
> ```bash
> PROJECT_DIR=/opt/myapp curl -fsSL https://raw.githubusercontent.com/jasperan/github-utils/master/install.sh | bash
> ```
>
> Or install manually:
> ```bash
> git clone https://github.com/jasperan/github-utils.git
> cd github-utils
> # See below for setup instructions
> ```
> </details>


```bash
# From source (development)
git clone https://github.com/jasperan/github-utils.git
cd github-utils
pip install -e ".[dev]"

# From PyPI
pip install github-utils
```

Requires Python 3.10+.

## Quick Start

```bash
# Paint "HELLO" on your contribution graph (dry run)
github-utils paint "HELLO" --dry-run

# Simulate 90 days of realistic activity
github-utils simulate --days 90 --profile default --dry-run

# Check available subcommands
github-utils --help
```

## Commands Reference

### paint

Render text, built-in patterns, or custom YAML pattern files onto the contribution graph.

```bash
# Paint text with default intensity (4)
github-utils paint "HI"

# Paint text with lower intensity
github-utils paint "CODE" --intensity 2

# Paint a built-in pattern
github-utils paint --pattern heart

# Paint from a custom YAML file
github-utils paint --file my-pattern.yaml

# Preview only (no commits)
github-utils paint "TEST" --preview

# Dry run (show commit plan without executing)
github-utils paint "ABC" --dry-run
```

Options:

| Option | Short | Description |
|--------|-------|-------------|
| `--pattern` | `-p` | Named built-in pattern (heart, wave, checkerboard) |
| `--file` | `-f` | Path to a YAML pattern file |
| `--intensity` | `-i` | Intensity level 1-4 for text mode (default: 4) |
| `--preview` | | Show terminal grid preview only |
| `--dry-run` | | Show commit plan without executing |

Text is rendered using a built-in 5x7 pixel font supporting A-Z, 0-9, and common punctuation (space, dash, period, exclamation mark). Each character occupies 5 columns with a 1-column gap. The contribution graph has 52 columns, so approximately 8 characters fit.

### simulate

Generate realistic GitHub activity patterns using configurable developer profiles.

```bash
# Simulate 90 days with the default profile
github-utils simulate --days 90

# Use a specific profile
github-utils simulate --days 180 --profile night-owl

# Reproducible output with a seed
github-utils simulate --days 30 --profile 9-to-5 --seed 42 --dry-run
```

Options:

| Option | Short | Description |
|--------|-------|-------------|
| `--days` | `-d` | Number of days to simulate (default: 90) |
| `--profile` | `-p` | Activity profile name (default: default) |
| `--seed` | `-s` | Random seed for reproducibility |
| `--dry-run` | | Show plan without executing |

### stats

Contribution analytics with snapshot and comparison capabilities.

```bash
# Take a snapshot of current contribution stats
github-utils stats snapshot

# Compare two snapshots
github-utils stats compare --before snap-before.json --after snap-after.json
```

Subcommands:

- `snapshot` -- Capture a point-in-time snapshot of your contribution statistics (requires GitHub API token).
- `compare` -- Load two snapshot JSON files and display deltas for total contributions, current streak, longest streak, and active days.

### legacy

Backward-compatible bridge for the original `utils.py` CLI interface.

```bash
# Create commits (legacy mode)
github-utils legacy commit 50000 --dry-run

# Branch operations (legacy mode)
github-utils legacy branch inf --dry-run
```

## Pattern Library

Three built-in patterns ship with the project. Custom patterns can be created as YAML files.

| Pattern | Description |
|---------|-------------|
| `heart` | Heart shape centered on the grid |
| `wave` | Sine wave pattern across the full width |
| `checkerboard` | Alternating filled and empty cells |

Custom pattern YAML format:

```yaml
name: my-pattern
grid:
- [0, 0, 4, 4, 0, 0, ...]  # row 0 (Sunday)
- [0, 4, 4, 4, 4, 0, ...]  # row 1 (Monday)
# ... 7 rows total, 52 columns each
# Values 0-4 represent intensity levels
```

## Activity Profiles

The simulate command uses a state machine with three modes (normal, burst, quiet) to produce organic-looking contribution patterns. Five named profiles are available:

| Profile | Description |
|---------|-------------|
| `default` | Balanced weekday activity with lighter weekends. Moderate burst/quiet cycles. |
| `night-owl` | Slightly higher weekday intensity, shorter bursts, fewer quiet days. |
| `weekend-warrior` | Low weekday activity, high weekend commits. Frequent short bursts. |
| `9-to-5` | Strong weekday focus, near-zero weekends. Lower burst probability, longer quiet periods. |
| `open-source-maintainer` | Consistent daily activity including weekends. Long burst phases, rare quiet days. |

Each profile configures: weekly pattern multipliers (Mon-Sun), commits per active day range, burst/quiet probabilities and durations, and burst multiplier.

## Realism Engines

When realism is enabled (default for `simulate`, optional for `paint`), two engines enhance the output:

- **Temporal Engine** -- Generates timestamps weighted toward business hours (peaks at 10-12 and 14-17) with per-commit jitter, so commits are spread naturally across each day rather than clustering at a single time.
- **Content Engine** -- Produces conventional-commit-style messages (e.g., `feat: add auth support`) from a bank of 2400+ unique combinations. File changes span multiple extensions (.py, .js, .ts, .md, .yaml, .json) with 1-4 files per commit.

## Configuration

Configuration is loaded from a YAML file with environment variable overrides.

### Config file

Default location: `~/.github-utils.yaml`

```yaml
token: ghp_xxxxxxxxxxxx    # GitHub personal access token
branch: github-utils-canvas # Branch name for painted commits
push_every: 500             # Push interval (commits between pushes)
timezone: UTC               # Timezone for timestamp generation
username: your-username     # GitHub username
```

Legacy format (`personal_access_token` key) is also accepted.

### Environment variables

| Variable | Config key | Description |
|----------|-----------|-------------|
| `GITHUB_UTILS_TOKEN` | `token` | GitHub API token |
| `GITHUB_UTILS_BRANCH` | `branch` | Target branch name |
| `GITHUB_UTILS_PUSH_EVERY` | `push_every` | Push interval |
| `GITHUB_UTILS_TIMEZONE` | `timezone` | Timezone |
| `GITHUB_UTILS_USERNAME` | `username` | GitHub username |
| `GITHUB_TOKEN` | `token` (fallback) | Fallback token source |

### GitHub CLI token

If no token is configured, the API client can pull a token from the GitHub CLI:

```bash
gh auth login
# github-utils will use `gh auth token` automatically
```

## Development

### Setup

```bash
conda create -n github-utils python=3.12
conda activate github-utils
pip install -e ".[dev]"
```

### Running tests

```bash
# Full suite with coverage
pytest --cov=github_utils -v

# Single test file
pytest tests/test_integration.py -v

# Only unit tests (fast)
pytest tests/ -v --ignore=tests/test_integration.py
```

### Project structure

```
src/github_utils/
  cli.py               # Typer app entry point
  config.py             # YAML + env var configuration
  executor.py           # Wires paint/simulation plans to git operations
  commands/
    paint.py            # paint subcommand
    simulate.py         # simulate subcommand
    stats.py            # stats subcommand
    legacy.py           # legacy bridge
  painter/
    font.py             # 5x7 pixel font (A-Z, 0-9, punctuation)
    grid.py             # 7x52 ContributionGrid model
    engine.py           # Grid -> PaintPlan orchestrator
    patterns.py         # Built-in and custom YAML pattern loader
    preview.py          # Terminal grid preview renderer
  realism/
    profile.py          # Developer activity profiles and state machine
    temporal.py         # Timestamp generation with hour weighting
    content.py          # Commit message and file change generation
  analytics/
    snapshot.py         # Snapshot persistence and comparison
  api/
    client.py           # GitHub GraphQL API client
  git/
    operations.py       # GitEngine for backdated commits and push
patterns/               # Built-in YAML patterns (heart, wave, checkerboard)
tests/                  # Unit and integration tests
```

## License

See [LICENSE](LICENSE) for details.
