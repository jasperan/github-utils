from __future__ import annotations

from datetime import datetime
from pathlib import Path
from typing import Dict, Optional

from git import Repo
from git.exc import GitCommandError


class GitEngine:
    """Thin wrapper around GitPython that creates backdated commits."""

    def __init__(self, repo: Repo) -> None:
        self.repo = repo
        self._commit_count = 0

    def create_backdated_commit(
        self,
        date: datetime,
        message: str,
        file_changes: Dict[str, str],
    ) -> None:
        repo_dir = Path(self.repo.working_dir)
        paths = []
        for rel_path, content in file_changes.items():
            full_path = repo_dir / rel_path
            full_path.parent.mkdir(parents=True, exist_ok=True)
            full_path.write_text(content)
            paths.append(rel_path)
        self.repo.index.add(paths)
        self.repo.index.commit(
            message=message,
            author_date=date,
            commit_date=date,
        )
        self._commit_count += 1

    def setup_orphan_branch(self, branch_name: str) -> None:
        self.repo.git.checkout("--orphan", branch_name)
        # An orphan branch over an empty worktree has nothing to remove; swallow
        # that specific failure and propagate any other git error.
        try:
            self.repo.git.rm("-rf", ".")
        except GitCommandError:
            pass

    def push(
        self, remote_name: str = "origin", branch: Optional[str] = None
    ) -> None:
        remote = self.repo.remote(remote_name)
        refspec = branch or self.repo.active_branch.name
        remote.push(refspec)

    @property
    def commit_count(self) -> int:
        return self._commit_count
