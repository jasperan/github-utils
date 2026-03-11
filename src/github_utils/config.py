from __future__ import annotations

import os
from pathlib import Path
from typing import Optional

import yaml
from pydantic import BaseModel


class Config(BaseModel):
    token: Optional[str] = None
    branch: str = "github-utils-canvas"
    push_every: int = 500
    timezone: str = "UTC"
    username: Optional[str] = None


def load_config(path: Optional[Path] = None) -> Config:
    """Load config from YAML file, with env var overrides."""
    data: dict = {}

    if path and path.exists():
        with open(path) as f:
            raw = yaml.safe_load(f) or {}
        # Legacy format support
        if "personal_access_token" in raw and "token" not in raw:
            raw["token"] = raw.pop("personal_access_token")
        data.update(raw)

    # Env var overrides
    env_map = {
        "GITHUB_UTILS_TOKEN": "token",
        "GITHUB_UTILS_BRANCH": "branch",
        "GITHUB_UTILS_PUSH_EVERY": "push_every",
        "GITHUB_UTILS_TIMEZONE": "timezone",
        "GITHUB_UTILS_USERNAME": "username",
    }
    for env_key, config_key in env_map.items():
        val = os.environ.get(env_key)
        if val is not None:
            data[config_key] = val

    # Also try GITHUB_TOKEN as fallback
    if "token" not in data or data["token"] is None:
        gh_token = os.environ.get("GITHUB_TOKEN")
        if gh_token:
            data["token"] = gh_token

    return Config(**data)
