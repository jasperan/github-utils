from datetime import date
from github_utils.painter.grid import ContributionGrid

def test_grid_dimensions():
    grid = ContributionGrid.empty()
    assert grid.rows == 7
    assert grid.cols == 52

def test_grid_get_set():
    grid = ContributionGrid.empty()
    assert grid.get(0, 0) == 0
    grid.set(0, 0, 3)
    assert grid.get(0, 0) == 3

def test_grid_intensity_clamped():
    grid = ContributionGrid.empty()
    grid.set(0, 0, 7)
    assert grid.get(0, 0) == 4
    grid.set(0, 0, -1)
    assert grid.get(0, 0) == 0

def test_grid_to_date_mapping():
    """Rightmost column maps to the current week."""
    grid = ContributionGrid.for_date(date(2026, 3, 11))  # Wednesday
    # (row=0, col=51) is Sunday of the current week
    d = grid.cell_to_date(row=0, col=51)
    assert d == date(2026, 3, 8)  # Sunday of that week
    # (row=3, col=51) is Wednesday
    d = grid.cell_to_date(row=3, col=51)
    assert d == date(2026, 3, 11)

def test_grid_from_list():
    data = [[0]*52 for _ in range(7)]
    data[0][0] = 4
    data[6][51] = 2
    grid = ContributionGrid.from_list(data)
    assert grid.get(0, 0) == 4
    assert grid.get(6, 51) == 2

def test_grid_to_list():
    grid = ContributionGrid.empty()
    grid.set(3, 10, 2)
    data = grid.to_list()
    assert len(data) == 7
    assert len(data[0]) == 52
    assert data[3][10] == 2

def test_grid_nonzero_cells():
    grid = ContributionGrid.empty()
    grid.set(0, 0, 1)
    grid.set(3, 25, 4)
    cells = grid.nonzero_cells()
    assert len(cells) == 2
    assert (0, 0, 1) in cells
    assert (3, 25, 4) in cells
