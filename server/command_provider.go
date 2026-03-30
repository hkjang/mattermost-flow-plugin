package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

type commandProvider struct {
	plugin *Plugin
}

func (p *commandProvider) OpenBoardURL(args *model.CommandArgs) (string, error) {
	teamID := strings.TrimSpace(args.TeamId)
	channelID := strings.TrimSpace(args.ChannelId)

	if teamID == "" {
		return "", fmt.Errorf("the command must run inside a team context")
	}

	team, appErr := p.plugin.API.GetTeam(teamID)
	if appErr != nil || team == nil {
		return "", fmt.Errorf("unable to resolve the current team")
	}

	query := make([]string, 0, 2)
	if channelID != "" {
		query = append(query, fmt.Sprintf("channel_id=%s", channelID))
	}

	boards, err := p.plugin.service.ListBoards(ScopeQuery{TeamID: teamID, ChannelID: channelID})
	if err == nil {
		if boardID := selectCommandBoardID(boards); boardID != "" {
			query = append(query, fmt.Sprintf("board_id=%s", boardID))
		}
	}

	base := strings.TrimRight(args.SiteURL, "/")
	if len(query) == 0 {
		return fmt.Sprintf("%s/%s/%s/boards", base, team.Name, PluginID), nil
	}

	return fmt.Sprintf("%s/%s/%s/boards?%s", base, team.Name, PluginID, strings.Join(query, "&")), nil
}

func (p *commandProvider) CreateCard(args *model.CommandArgs, title, dueDate string) (string, error) {
	teamID := strings.TrimSpace(args.TeamId)
	channelID := strings.TrimSpace(args.ChannelId)

	boards, err := p.plugin.service.ListBoards(ScopeQuery{TeamID: teamID, ChannelID: channelID})
	if err != nil {
		return "", err
	}
	boardID := selectCommandBoardID(boards)
	if boardID == "" {
		return "", fmt.Errorf("no board exists in this scope yet; run /flow open first")
	}

	bundle, err := p.plugin.service.GetBoardBundle(boardID, args.UserId)
	if err != nil {
		return "", err
	}
	if len(bundle.Columns) == 0 {
		return "", fmt.Errorf("the selected board has no columns configured")
	}

	result, err := p.plugin.service.CreateCard(args.UserId, CreateCardRequest{
		BoardID:  boardID,
		ColumnID: bundle.Columns[0].ID,
		Title:    title,
		DueDate:  dueDate,
		Priority: "normal",
	})
	if err != nil {
		return "", err
	}

	p.plugin.postCardUpdate(result.Board, result.Card, args.UserId, fmt.Sprintf("created card **%s** in **%s**", result.Card.Title, result.ColumnName))
	p.plugin.publishBoardEvent(BoardStreamEvent{
		BoardID:    result.Board.ID,
		EntityType: "card",
		Action:     "card.created",
		ActorID:    args.UserId,
		CardID:     result.Card.ID,
		Board:      &result.Board,
		Card:       &result.Card,
	})
	linkURL := p.plugin.buildFlowCardURL(result.Board, result.Card.ID, "board")
	if linkURL == "" {
		return fmt.Sprintf("Created **%s** in **%s / %s**.", result.Card.Title, result.Board.Name, result.ColumnName), nil
	}
	return fmt.Sprintf("Created **%s** in **%s / %s**. [Open card](%s)", result.Card.Title, result.Board.Name, result.ColumnName, linkURL), nil
}

func (p *commandProvider) ListBoardsSummary(args *model.CommandArgs) (string, error) {
	teamID := strings.TrimSpace(args.TeamId)
	channelID := strings.TrimSpace(args.ChannelId)

	boards, err := p.plugin.service.ListBoards(ScopeQuery{TeamID: teamID, ChannelID: channelID})
	if err != nil {
		return "", err
	}

	if len(boards) == 0 {
		return "No boards found in this scope. Use `/flow open` then create one.", nil
	}

	lines := make([]string, 0, len(boards)+1)
	lines = append(lines, fmt.Sprintf("**%d board(s)** in this scope:", len(boards)))
	for _, summary := range boards {
		defaultMark := ""
		if summary.DefaultBoard {
			defaultMark = " (default)"
		}
		lines = append(lines, fmt.Sprintf("- **%s**%s — %d cards, %d overdue, %d due soon", summary.Board.Name, defaultMark, summary.CardCount, summary.OverdueCount, summary.DueSoonCount))
	}
	return strings.Join(lines, "\n"), nil
}

func (p *commandProvider) BoardStatus(args *model.CommandArgs) (string, error) {
	teamID := strings.TrimSpace(args.TeamId)
	channelID := strings.TrimSpace(args.ChannelId)

	boards, err := p.plugin.service.ListBoards(ScopeQuery{TeamID: teamID, ChannelID: channelID})
	if err != nil {
		return "", err
	}

	boardID := selectCommandBoardID(boards)
	if boardID == "" {
		return "No board exists in this scope yet. Create one with `/flow open`.", nil
	}

	bundle, err := p.plugin.service.GetBoardBundle(boardID, args.UserId)
	if err != nil {
		return "", err
	}

	lines := make([]string, 0, 8)
	lines = append(lines, fmt.Sprintf("**%s** status:", bundle.Board.Name))

	totalCards := len(bundle.Cards)
	completedCards := 0
	for _, card := range bundle.Cards {
		if card.Progress >= 100 {
			completedCards++
		}
	}

	lines = append(lines, fmt.Sprintf("| Metric | Value |"))
	lines = append(lines, fmt.Sprintf("| --- | --- |"))
	lines = append(lines, fmt.Sprintf("| Columns | %d |", len(bundle.Columns)))
	lines = append(lines, fmt.Sprintf("| Total cards | %d |", totalCards))
	lines = append(lines, fmt.Sprintf("| Completed | %d |", completedCards))
	lines = append(lines, fmt.Sprintf("| Overdue | %d |", bundle.Summary.OverdueCount))
	lines = append(lines, fmt.Sprintf("| Due soon | %d |", bundle.Summary.DueSoonCount))

	if totalCards > 0 {
		pct := completedCards * 100 / totalCards
		lines = append(lines, fmt.Sprintf("\nOverall progress: **%d%%** (%d/%d)", pct, completedCards, totalCards))
	}

	return strings.Join(lines, "\n"), nil
}

func (p *commandProvider) AssignCard(args *model.CommandArgs, titleKeyword, assigneeUsername string) (string, error) {
	teamID := strings.TrimSpace(args.TeamId)
	channelID := strings.TrimSpace(args.ChannelId)

	boards, err := p.plugin.service.ListBoards(ScopeQuery{TeamID: teamID, ChannelID: channelID})
	if err != nil {
		return "", err
	}

	boardID := selectCommandBoardID(boards)
	if boardID == "" {
		return "", fmt.Errorf("no board exists in this scope")
	}

	// Resolve user by username
	user, appErr := p.plugin.API.GetUserByUsername(assigneeUsername)
	if appErr != nil || user == nil {
		return "", fmt.Errorf("could not find user @%s", assigneeUsername)
	}

	cards, err := p.plugin.service.ListCards(boardID)
	if err != nil {
		return "", err
	}

	keyword := strings.ToLower(titleKeyword)
	var matchedCard *Card
	for _, card := range cards {
		if strings.Contains(strings.ToLower(card.Title), keyword) {
			matchedCard = &card
			break
		}
	}

	if matchedCard == nil {
		return "", fmt.Errorf("no card found matching \"%s\"", titleKeyword)
	}

	// Check if already assigned
	for _, assigneeID := range matchedCard.AssigneeIDs {
		if assigneeID == user.Id {
			return fmt.Sprintf("**%s** is already assigned to @%s.", matchedCard.Title, assigneeUsername), nil
		}
	}

	newAssignees := append(matchedCard.AssigneeIDs, user.Id)
	_, err = p.plugin.service.UpdateCard(args.UserId, matchedCard.ID, UpdateCardRequest{
		AssigneeIDs: &newAssignees,
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Assigned @%s to **%s**.", assigneeUsername, matchedCard.Title), nil
}

func selectCommandBoardID(boards []BoardSummary) string {
	if len(boards) == 0 {
		return ""
	}
	for _, board := range boards {
		if board.DefaultBoard {
			return board.Board.ID
		}
	}
	return boards[0].Board.ID
}
