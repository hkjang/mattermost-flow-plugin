package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildImportedBoardData(t *testing.T) {
	startOffset := 1
	dueOffset := 3
	req := ImportBoardRequest{
		TeamID: "team-1",
		Name:   "Imported Release Board",
		Package: BoardExportPackage{
			Version: 1,
			SourceBoard: Board{
				ID:          "source-board",
				Name:        "Release Board",
				Description: "Original board",
				Visibility:  "team",
				AdminIDs:    []string{"owner-1"},
				Settings: BoardSettings{
					PostUpdates:         true,
					PostDueSoon:         true,
					AllowMentions:       true,
					DefaultView:         "dashboard",
					CalendarFeedEnabled: true,
				},
			},
			Columns: []BoardColumn{
				{ID: "col-a", Name: "Todo", SortOrder: 0},
				{ID: "col-b", Name: "Done", SortOrder: 1},
			},
			Templates: []CardTemplate{
				{
					ID:              "tpl-1",
					Name:            "Release",
					Title:           "Ship release",
					Priority:        "high",
					StartOffsetDays: &startOffset,
					DueOffsetDays:   &dueOffset,
					Checklist:       []ChecklistItem{{ID: "check-1", Text: "Draft notes"}},
					AttachmentLinks: []AttachmentLink{{ID: "link-1", Title: "Runbook", URL: "https://example.com/runbook"}},
				},
			},
			Cards: []Card{
				{
					ID:          "card-1",
					ColumnID:    "col-a",
					Title:       "Prepare release",
					Position:    0,
					Priority:    "high",
					Checklist:   []ChecklistItem{{ID: "item-1", Text: "Freeze branch"}},
					Comments:    []CardComment{{ID: "comment-1", CardID: "card-1", UserID: "user-1", Message: "Ready soon"}},
					Description: "Track go-live tasks",
				},
				{
					ID:          "card-2",
					ColumnID:    "col-b",
					Title:       "Launch",
					Position:    0,
					Priority:    "urgent",
					Progress:    100,
					Milestone:   true,
					Description: "Release day",
				},
			},
			Dependencies: []Dependency{
				{ID: "dep-1", SourceCardID: "card-1", TargetCardID: "card-2", Type: "finish_to_start"},
			},
		},
	}

	board, columns, templates, cards, dependencies, err := buildImportedBoardData("actor-1", req)
	require.NoError(t, err)

	require.Equal(t, "Imported Release Board", board.Name)
	require.Equal(t, "team-1", board.TeamID)
	require.Empty(t, board.ChannelID)
	require.True(t, board.Settings.CalendarFeedEnabled)
	require.Len(t, columns, 2)
	require.Len(t, templates, 1)
	require.Len(t, cards, 2)
	require.Len(t, dependencies, 1)
	require.NotEqual(t, "source-board", board.ID)
	require.NotEqual(t, "col-a", columns[0].ID)
	require.NotEqual(t, "tpl-1", templates[0].ID)
	require.NotEqual(t, "card-1", cards[0].ID)
	require.Equal(t, board.ID, cards[0].BoardID)
	require.Equal(t, cards[0].ID, cards[0].Comments[0].CardID)
	require.NotEqual(t, "dep-1", dependencies[0].ID)
	require.Equal(t, cards[0].ID, dependencies[0].SourceCardID)
	require.Equal(t, cards[1].ID, dependencies[0].TargetCardID)
}
