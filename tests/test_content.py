"""Tests for the Content Realism Engine."""

from github_utils.realism.content import ContentEngine


def test_generate_commit_message():
    engine = ContentEngine(seed=42)
    msg = engine.generate_message()
    assert isinstance(msg, str)
    assert len(msg) > 5
    assert ":" in msg  # conventional commits format


def test_messages_are_varied():
    engine = ContentEngine(seed=42)
    messages = [engine.generate_message() for _ in range(20)]
    unique = set(messages)
    assert len(unique) > 10


def test_generate_file_changes():
    engine = ContentEngine(seed=42)
    changes = engine.generate_file_changes()
    assert isinstance(changes, dict)
    assert len(changes) >= 1
    for path, content in changes.items():
        assert isinstance(path, str)
        assert isinstance(content, str)
        assert len(content) > 0


def test_file_changes_varied_extensions():
    engine = ContentEngine(seed=42)
    all_paths = []
    for _ in range(50):
        changes = engine.generate_file_changes()
        all_paths.extend(changes.keys())
    extensions = {p.rsplit(".", 1)[-1] for p in all_paths if "." in p}
    assert len(extensions) >= 3


def test_seed_reproducibility():
    e1 = ContentEngine(seed=99)
    e2 = ContentEngine(seed=99)
    assert e1.generate_message() == e2.generate_message()
    assert e1.generate_file_changes() == e2.generate_file_changes()
