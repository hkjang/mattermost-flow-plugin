package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildBoardCalendarICS(t *testing.T) {
	board := Board{
		ID:          "board-1",
		Name:        "Release board",
		Description: "Cross-team release calendar",
	}
	columns := []BoardColumn{
		{ID: "col-1", Name: "In Progress"},
	}
	cards := []Card{
		{
			ID:          "card-1",
			BoardID:     board.ID,
			ColumnID:    "col-1",
			Title:       "Release train",
			Description: "Finalize notes\nPublish rollout",
			Labels:      []string{"release", "ops"},
			Priority:    "high",
			StartDate:   "2026-03-29",
			DueDate:     "2026-03-30",
			Progress:    65,
			UpdatedAt:   1760000000000,
		},
	}

	calendar := buildBoardCalendarICS(board, columns, cards, func(cardID string) string {
		return "https://example.com/card/" + cardID
	})

	require.Contains(t, calendar, "BEGIN:VCALENDAR")
	require.Contains(t, calendar, "X-WR-CALNAME:Release board")
	require.Contains(t, calendar, "SUMMARY:Release train")
	require.Contains(t, calendar, "DTSTART;VALUE=DATE:20260329")
	require.Contains(t, calendar, "DTEND;VALUE=DATE:20260331")
	require.Contains(t, calendar, "CATEGORIES:release\\,ops")
	require.Contains(t, calendar, "Open in Mattermost Flow: https://example.com/card/card-1")
	require.True(t, strings.HasSuffix(calendar, "END:VCALENDAR\r\n"))
}
