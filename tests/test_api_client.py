# tests/test_api_client.py
from github_utils.api.client import GitHubClient, ContributionData


def test_parse_contribution_response():
    raw = {
        "data": {
            "viewer": {
                "contributionsCollection": {
                    "totalContributions": 150,
                    "contributionCalendar": {
                        "totalContributions": 150,
                        "weeks": [
                            {
                                "contributionDays": [
                                    {"date": "2025-03-09", "contributionCount": 5, "color": "#30a14e", "weekday": 0},
                                    {"date": "2025-03-10", "contributionCount": 0, "color": "#ebedf0", "weekday": 1},
                                    {"date": "2025-03-11", "contributionCount": 12, "color": "#216e39", "weekday": 2},
                                    {"date": "2025-03-12", "contributionCount": 3, "color": "#9be9a8", "weekday": 3},
                                    {"date": "2025-03-13", "contributionCount": 0, "color": "#ebedf0", "weekday": 4},
                                    {"date": "2025-03-14", "contributionCount": 7, "color": "#30c463", "weekday": 5},
                                    {"date": "2025-03-15", "contributionCount": 0, "color": "#ebedf0", "weekday": 6},
                                ]
                            }
                        ],
                    }
                }
            }
        }
    }
    data = ContributionData.from_graphql_response(raw)
    assert data.total == 150
    assert data.day_counts["2025-03-11"] == 12
    assert data.day_counts["2025-03-10"] == 0


def test_calculate_intensity_thresholds():
    data = ContributionData(
        total=100,
        day_counts={f"2025-01-{i:02d}": i for i in range(1, 21)},
    )
    thresholds = data.intensity_thresholds()
    assert len(thresholds) == 4
    assert thresholds[0] <= thresholds[1] <= thresholds[2] <= thresholds[3]


def test_commits_needed_for_intensity():
    data = ContributionData(
        total=100,
        day_counts={f"2025-01-{i:02d}": i for i in range(1, 21)},
    )
    n = data.commits_for_intensity(4)
    assert n > 0
    n0 = data.commits_for_intensity(0)
    assert n0 == 0
