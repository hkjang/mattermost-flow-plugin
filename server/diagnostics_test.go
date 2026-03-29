package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildBoardDiagnosticsReport(t *testing.T) {
	board := Board{ID: "board-1"}
	columns := []BoardColumn{
		{ID: "todo", Name: "Todo"},
		{ID: "done", Name: "Done"},
	}
	cards := []Card{
		{
			ID:       "card-1",
			BoardID:  board.ID,
			ColumnID: "todo",
			Title:    "Valid",
			Position: 0,
			DueDate:  "2026-04-01",
		},
		{
			ID:        "card-2",
			BoardID:   board.ID,
			ColumnID:  "missing-column",
			Title:     "Orphan column",
			Position:  0,
			StartDate: "2026-04-10",
			DueDate:   "2026-04-05",
		},
		{
			ID:       "card-3",
			BoardID:  board.ID,
			ColumnID: "todo",
			Title:    "Duplicate position",
			Position: 0,
		},
	}
	dependencies := []Dependency{
		{ID: "dep-1", SourceCardID: "card-1", TargetCardID: "missing-card"},
		{ID: "dep-2", SourceCardID: "card-3", TargetCardID: "card-3"},
	}

	report := buildBoardDiagnosticsReport(board, columns, nil, cards, dependencies, nil)

	require.False(t, report.Healthy)
	require.True(t, report.RepairAvailable)
	require.Equal(t, 3, report.Summary.Cards)
	require.Equal(t, 1, report.Summary.InvalidDates)

	codes := make(map[string]BoardDiagnosticsIssue, len(report.Issues))
	for _, issue := range report.Issues {
		codes[issue.Code] = issue
	}

	require.Contains(t, codes, "orphan_column_cards")
	require.Contains(t, codes, "invalid_card_dates")
	require.Contains(t, codes, "duplicate_card_positions")
	require.Contains(t, codes, "orphan_dependencies")
	require.Contains(t, codes, "self_dependencies")
}
