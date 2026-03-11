"""GitHub GraphQL API client with contribution data parsing."""

from __future__ import annotations

import math
import subprocess
from dataclasses import dataclass, field
from typing import Dict, List, Optional

import httpx

_CONTRIBUTIONS_QUERY = """
query($from: DateTime!, $to: DateTime!) {
  viewer {
    login
    contributionsCollection(from: $from, to: $to) {
      totalContributions
      contributionCalendar {
        totalContributions
        weeks {
          contributionDays {
            date
            contributionCount
            color
            weekday
          }
        }
      }
    }
  }
  rateLimit {
    remaining
    resetAt
  }
}
"""


@dataclass
class ContributionData:
    """Parsed contribution calendar data from the GitHub GraphQL API."""

    total: int
    day_counts: Dict[str, int] = field(default_factory=dict)

    @classmethod
    def from_graphql_response(cls, raw: dict) -> ContributionData:
        """Parse a raw GraphQL response into a ContributionData instance."""
        if "errors" in raw:
            msgs = "; ".join(e.get("message", str(e)) for e in raw["errors"])
            raise ValueError(f"GitHub GraphQL error: {msgs}")
        if raw.get("data") is None:
            raise ValueError("GitHub GraphQL response missing 'data' field")
        collection = raw["data"]["viewer"]["contributionsCollection"]
        calendar = collection["contributionCalendar"]
        day_counts: Dict[str, int] = {}
        for week in calendar["weeks"]:
            for day in week["contributionDays"]:
                day_counts[day["date"]] = day["contributionCount"]
        return cls(total=calendar["totalContributions"], day_counts=day_counts)

    def intensity_thresholds(self) -> List[int]:
        """Calculate quartile-based intensity thresholds from non-zero contribution days.

        Returns a list of 4 thresholds corresponding to intensity levels 1-4.
        GitHub uses a similar quartile approach for coloring the contribution graph.
        """
        nonzero = sorted(v for v in self.day_counts.values() if v > 0)
        if not nonzero:
            return [1, 2, 3, 4]
        q = [0.25, 0.50, 0.75, 1.0]
        thresholds: List[int] = []
        for quantile in q:
            idx = max(0, math.ceil(len(nonzero) * quantile) - 1)
            thresholds.append(nonzero[idx])
        return thresholds

    def commits_for_intensity(self, level: int) -> int:
        """Return the number of commits needed to reach a given intensity level (0-4).

        Level 0 means no contributions needed (returns 0).
        Levels 1-4 map to the corresponding quartile thresholds.
        """
        if level <= 0:
            return 0
        thresholds = self.intensity_thresholds()
        idx = min(level - 1, len(thresholds) - 1)
        return thresholds[idx]


class GitHubClient:
    """Client for the GitHub GraphQL API, focused on contribution data."""

    GRAPHQL_URL = "https://api.github.com/graphql"

    def __init__(self, token: str):
        self._token = token
        self._client = httpx.Client(
            headers={
                "Authorization": f"bearer {token}",
                "Content-Type": "application/json",
            },
            timeout=30.0,
        )
        self.rate_limit_remaining: Optional[int] = None
        self.rate_limit_reset: Optional[str] = None

    @classmethod
    def from_gh_cli(cls) -> GitHubClient:
        """Create a client using the token from ``gh auth token``."""
        result = subprocess.run(
            ["gh", "auth", "token"],
            capture_output=True,
            text=True,
            check=True,
        )
        return cls(result.stdout.strip())

    def fetch_contributions(
        self, from_date: str, to_date: str
    ) -> ContributionData:
        """Fetch contribution data for a date range (ISO 8601 datetime strings).

        Updates ``rate_limit_remaining`` and ``rate_limit_reset`` from the response.
        """
        resp = self._client.post(
            self.GRAPHQL_URL,
            json={
                "query": _CONTRIBUTIONS_QUERY,
                "variables": {"from": from_date, "to": to_date},
            },
        )
        resp.raise_for_status()
        data = resp.json()
        if "rateLimit" in data.get("data", {}):
            rl = data["data"]["rateLimit"]
            self.rate_limit_remaining = rl.get("remaining")
            self.rate_limit_reset = rl.get("resetAt")
        return ContributionData.from_graphql_response(data)

    def close(self) -> None:
        """Close the underlying HTTP client."""
        self._client.close()
