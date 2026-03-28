package main

func (p *Plugin) publishBoardEvent(event BoardStreamEvent) {
	if p.eventBroker == nil || event.BoardID == "" {
		return
	}

	if needsColumnCardSnapshot(event.Action) && len(event.ColumnCardIDs) == 0 && p.service != nil {
		snapshot, err := p.service.BuildColumnCardIDs(event.BoardID)
		if err == nil {
			event.ColumnCardIDs = snapshot
		}
	}

	if event.Type == "" {
		event.Type = "board_event"
	}
	if event.OccurredAt == 0 {
		event.OccurredAt = nowMillis()
	}
	if event.Activity == nil {
		event.Activity = p.latestBoardActivity(event.BoardID)
	}

	p.eventBroker.Publish(event)
	p.publishBoardSummaryEvent(event)
}

func (p *Plugin) publishBoardSummaryEvent(event BoardStreamEvent) {
	if p.eventBroker == nil || p.service == nil || event.BoardID == "" {
		return
	}

	board := event.Board
	if board == nil {
		loadedBoard, err := p.service.GetBoard(event.BoardID)
		if err != nil {
			return
		}
		board = &loadedBoard
	}

	scopeKeys := scopeKeysForBoard(*board)
	if len(scopeKeys) == 0 {
		return
	}

	summaryEvent := BoardSummaryStreamEvent{
		Type:       "board_summary_event",
		BoardID:    event.BoardID,
		Action:     event.Action,
		OccurredAt: event.OccurredAt,
	}

	if event.Action != "board.deleted" {
		summary, err := p.service.GetBoardSummary(event.BoardID)
		if err == nil {
			summaryEvent.Summary = &summary
		}
	}

	p.eventBroker.PublishSummary(scopeKeys, summaryEvent)
}

func needsColumnCardSnapshot(action string) bool {
	switch action {
	case "card.moved", "card.completed":
		return true
	default:
		return false
	}
}

func (p *Plugin) latestBoardActivity(boardID string) *Activity {
	if p.service == nil || boardID == "" {
		return nil
	}

	activity, err := p.service.ListActivity(boardID)
	if err != nil || len(activity) == 0 {
		return nil
	}

	item := activity[0]
	return &item
}

func scopeKeysForBoard(board Board) []string {
	return uniqueStrings([]string{
		scopeKeyForChannel(board.ChannelID),
		scopeKeyForTeam(board.TeamID),
	})
}

func scopeKeyForTeam(teamID string) string {
	if teamID == "" {
		return ""
	}
	return "team:" + teamID
}

func scopeKeyForChannel(channelID string) string {
	if channelID == "" {
		return ""
	}
	return "channel:" + channelID
}
