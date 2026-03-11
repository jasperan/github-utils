from datetime import datetime, timezone
from pathlib import Path
from github_utils.git.operations import GitEngine

def _init_test_repo(tmp_path):
    from git import Repo
    repo = Repo.init(tmp_path / "test-repo")
    readme = tmp_path / "test-repo" / "README.md"
    readme.write_text("init")
    repo.index.add(["README.md"])
    repo.index.commit("initial commit")
    return repo

def test_create_backdated_commit(tmp_path):
    repo = _init_test_repo(tmp_path)
    engine = GitEngine(repo)
    target_date = datetime(2025, 6, 15, 14, 30, 0, tzinfo=timezone.utc)
    engine.create_backdated_commit(
        date=target_date,
        message="test: backdated commit",
        file_changes={"test.txt": "hello world"},
    )
    commit = repo.head.commit
    assert commit.message == "test: backdated commit"
    assert commit.authored_datetime.date() == target_date.date()
    assert commit.committed_datetime.date() == target_date.date()

def test_create_multiple_backdated_commits(tmp_path):
    repo = _init_test_repo(tmp_path)
    engine = GitEngine(repo)
    dates = [
        datetime(2025, 1, i, 10, 0, 0, tzinfo=timezone.utc)
        for i in range(1, 6)
    ]
    for i, d in enumerate(dates):
        engine.create_backdated_commit(
            date=d,
            message=f"commit {i}",
            file_changes={f"file_{i}.txt": f"content {i}"},
        )
    log = list(repo.iter_commits())
    assert len(log) == 6  # 1 initial + 5

def test_setup_orphan_branch(tmp_path):
    repo = _init_test_repo(tmp_path)
    engine = GitEngine(repo)
    engine.setup_orphan_branch("test-canvas")
    assert repo.active_branch.name == "test-canvas"

def test_file_changes_written(tmp_path):
    repo = _init_test_repo(tmp_path)
    engine = GitEngine(repo)
    target_date = datetime(2025, 3, 1, 12, 0, 0, tzinfo=timezone.utc)
    engine.create_backdated_commit(
        date=target_date,
        message="multi-file commit",
        file_changes={
            "src/main.py": "print('hello')",
            "docs/readme.md": "# Title",
        },
    )
    assert (tmp_path / "test-repo" / "src" / "main.py").read_text() == "print('hello')"
    assert (tmp_path / "test-repo" / "docs" / "readme.md").read_text() == "# Title"
