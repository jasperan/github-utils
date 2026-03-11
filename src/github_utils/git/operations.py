from __future__ import annotations
import os
from datetime import datetime
from pathlib import Path
from typing import Dict, Optional
from git import Repo, Actor


class GitEngine:
    def __init__(self, repo: Repo):
        self.repo = repo
        self._commit_count = 0

    @classmethod
    def from_path(cls, path: str | Path) -> GitEngine:
        return cls(Repo(path))

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
        try:
            self.repo.git.rm("-rf", ".")
        except Exception:
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
