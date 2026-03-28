package main

type flowColumnDetails struct {
	currentName string
	nextID      string
	nextName    string
	hasNext     bool
	doneName    string
	hasDone     bool
	inDone      bool
}

func (p *Plugin) buildFlowCardPostProps(board Board, card Card) map[string]any {
	details := p.describeCardColumns(board.ID, card.ColumnID)
	cardLinkURL := p.buildFlowCardURL(board, card.ID, "board")
	ganttLinkURL := p.buildFlowCardURL(board, card.ID, "gantt")
	checklistCompletedCount, checklistTotalCount, nextChecklistItem := describeChecklist(card.Checklist)

	return map[string]any{
		"board_id":                  board.ID,
		"board_name":                board.Name,
		"card_id":                   card.ID,
		"card_title":                card.Title,
		"due_date":                  card.DueDate,
		"progress":                  card.Progress,
		"assignee_ids":              append([]string(nil), card.AssigneeIDs...),
		"current_column_name":       details.currentName,
		"next_column_id":            details.nextID,
		"next_column_name":          details.nextName,
		"has_next_column":           details.hasNext,
		"done_column_name":          details.doneName,
		"has_done_column":           details.hasDone,
		"in_done_column":            details.inDone,
		"card_link_url":             cardLinkURL,
		"gantt_link_url":            ganttLinkURL,
		"checklist_completed_count": checklistCompletedCount,
		"checklist_total_count":     checklistTotalCount,
		"next_checklist_item":       nextChecklistItem,
	}
}

func (p *Plugin) describeCardColumns(boardID, columnID string) flowColumnDetails {
	if p.service == nil || p.service.store == nil || boardID == "" {
		return flowColumnDetails{}
	}

	columns, err := p.service.store.GetColumns(boardID)
	if err != nil {
		return flowColumnDetails{}
	}

	details := flowColumnDetails{}
	if doneColumn, ok := findDoneColumn(columns); ok {
		details.doneName = doneColumn.Name
		details.hasDone = true
		details.inDone = doneColumn.ID == columnID
	}

	for index, column := range columns {
		if column.ID != columnID {
			continue
		}

		details.currentName = column.Name
		if index+1 < len(columns) {
			nextColumn := columns[index+1]
			details.nextID = nextColumn.ID
			details.nextName = nextColumn.Name
			details.hasNext = true
		}
		return details
	}

	return details
}

func describeChecklist(items []ChecklistItem) (int, int, string) {
	completedCount := 0
	nextItemText := ""

	for _, item := range items {
		if item.Completed {
			completedCount++
			continue
		}
		if nextItemText == "" {
			nextItemText = item.Text
		}
	}

	return completedCount, len(items), nextItemText
}
