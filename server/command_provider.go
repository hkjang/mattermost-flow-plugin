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
