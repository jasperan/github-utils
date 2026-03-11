from __future__ import annotations

from datetime import date, timedelta
from typing import List, Tuple


class ContributionGrid:
    """52-column x 7-row grid modeling a GitHub contribution graph."""

    ROWS = 7
    COLS = 52

    def __init__(self, data: List[List[int]], anchor_date: date | None = None):
        if len(data) != self.ROWS:
            raise ValueError(f"Grid must have {self.ROWS} rows, got {len(data)}")
        for i, row in enumerate(data):
            if len(row) != self.COLS:
                raise ValueError(f"Row {i} must have {self.COLS} columns, got {len(row)}")
        self._data = data
        self._anchor = anchor_date or date.today()

    @property
    def rows(self) -> int:
        return self.ROWS

    @property
    def cols(self) -> int:
        return self.COLS

    @classmethod
    def empty(cls, anchor_date: date | None = None) -> ContributionGrid:
        data = [[0] * cls.COLS for _ in range(cls.ROWS)]
        return cls(data, anchor_date)

    @classmethod
    def for_date(cls, d: date) -> ContributionGrid:
        return cls.empty(anchor_date=d)

    @classmethod
    def from_list(cls, data: List[List[int]], anchor_date: date | None = None) -> ContributionGrid:
        normalized = []
        for row in data[:cls.ROWS]:
            clamped = [max(0, min(4, v)) for v in row[:cls.COLS]]
            # Pad short rows
            clamped.extend([0] * (cls.COLS - len(clamped)))
            normalized.append(clamped)
        # Pad missing rows
        while len(normalized) < cls.ROWS:
            normalized.append([0] * cls.COLS)
        return cls(normalized, anchor_date)

    def get(self, row: int, col: int) -> int:
        return self._data[row][col]

    def set(self, row: int, col: int, value: int) -> None:
        self._data[row][col] = max(0, min(4, value))

    def to_list(self) -> List[List[int]]:
        return [row[:] for row in self._data]

    def cell_to_date(self, row: int, col: int) -> date:
        """Convert grid position to calendar date.
        Row 0 = Sunday, Row 6 = Saturday.
        Col 51 = current week, Col 0 = 51 weeks ago.
        """
        anchor_weekday = self._anchor.isoweekday() % 7  # 0=Sun
        anchor_sunday = self._anchor - timedelta(days=anchor_weekday)
        origin_sunday = anchor_sunday - timedelta(weeks=self.COLS - 1)
        return origin_sunday + timedelta(weeks=col, days=row)

    def nonzero_cells(self) -> List[Tuple[int, int, int]]:
        """Return list of (row, col, intensity) for all non-zero cells."""
        cells = []
        for r in range(self.ROWS):
            for c in range(self.COLS):
                if self._data[r][c] > 0:
                    cells.append((r, c, self._data[r][c]))
        return cells
