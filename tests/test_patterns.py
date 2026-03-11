import tempfile
from pathlib import Path
from github_utils.painter.patterns import load_pattern, list_builtin_patterns
from github_utils.painter.grid import ContributionGrid

def test_list_builtin_patterns():
    patterns = list_builtin_patterns()
    assert "wave" in patterns
    assert "checkerboard" in patterns
    assert "heart" in patterns

def test_load_builtin_pattern():
    grid = load_pattern("wave")
    assert isinstance(grid, ContributionGrid)
    assert grid.rows == 7
    assert grid.cols == 52
    cells = grid.nonzero_cells()
    assert len(cells) > 0

def test_load_custom_pattern(tmp_path):
    custom = tmp_path / "custom.yaml"
    row = [0] * 52
    row[0] = 3
    custom.write_text(
        "name: custom\n"
        "grid:\n"
        + "".join(f"  - {row}\n" if i == 0 else f"  - {[0]*52}\n" for i in range(7))
    )
    grid = load_pattern(str(custom))
    assert grid.get(0, 0) == 3

def test_load_nonexistent_raises():
    import pytest
    with pytest.raises(FileNotFoundError):
        load_pattern("nonexistent_pattern_xyz")
