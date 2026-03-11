import os
from pathlib import Path
from github_utils.config import load_config, Config

def test_config_defaults():
    """Config has sensible defaults with no file or env vars."""
    cfg = Config()
    assert cfg.token is None
    assert cfg.branch == "github-utils-canvas"
    assert cfg.push_every == 500
    assert cfg.timezone == "UTC"

def test_config_from_yaml(tmp_path):
    """Loads config from YAML file."""
    config_file = tmp_path / "config.yaml"
    config_file.write_text(
        "token: ghp_test123\n"
        "branch: my-branch\n"
        "push_every: 100\n"
        "timezone: Europe/Madrid\n"
    )
    cfg = load_config(config_file)
    assert cfg.token == "ghp_test123"
    assert cfg.branch == "my-branch"
    assert cfg.push_every == 100
    assert cfg.timezone == "Europe/Madrid"

def test_config_env_override(tmp_path, monkeypatch):
    """Env vars override YAML values."""
    config_file = tmp_path / "config.yaml"
    config_file.write_text("token: from_yaml\n")
    monkeypatch.setenv("GITHUB_UTILS_TOKEN", "from_env")
    cfg = load_config(config_file)
    assert cfg.token == "from_env"

def test_config_from_legacy_yaml(tmp_path):
    """Reads token from legacy config.yaml format."""
    config_file = tmp_path / "config.yaml"
    config_file.write_text("personal_access_token: ghp_legacy123\n")
    cfg = load_config(config_file)
    assert cfg.token == "ghp_legacy123"

def test_config_missing_file():
    """Returns defaults when config file doesn't exist."""
    cfg = load_config(Path("/nonexistent/config.yaml"))
    assert cfg.token is None
    assert cfg.branch == "github-utils-canvas"
