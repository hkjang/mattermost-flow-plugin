package main

import (
	"fmt"
	"net/url"
	"strings"
)

func (p *Plugin) buildFlowBoardURL(board Board, view string) string {
	query := url.Values{}
	if view != "" {
		query.Set("view", view)
	}

	return p.buildFlowURL(board, query)
}

func (p *Plugin) buildFlowCardURL(board Board, cardID, view string) string {
	query := url.Values{}
	if cardID != "" {
		query.Set("card_id", cardID)
	}
	if view != "" {
		query.Set("view", view)
	}

	return p.buildFlowURL(board, query)
}

func (p *Plugin) buildFlowURL(board Board, query url.Values) string {
	config := p.API.GetConfig()
	if config == nil || config.ServiceSettings.SiteURL == nil {
		return ""
	}

	siteURL := strings.TrimRight(strings.TrimSpace(*config.ServiceSettings.SiteURL), "/")
	if siteURL == "" {
		return ""
	}

	teamName := p.resolveBoardTeamName(board)
	if teamName == "" {
		return ""
	}

	if query == nil {
		query = url.Values{}
	}
	query.Set("board_id", board.ID)
	if board.ChannelID != "" {
		query.Set("channel_id", board.ChannelID)
	}

	return fmt.Sprintf("%s/%s/%s/boards?%s", siteURL, teamName, PluginID, query.Encode())
}

func (p *Plugin) resolveBoardTeamName(board Board) string {
	teamID := strings.TrimSpace(board.TeamID)
	if teamID == "" && board.ChannelID != "" {
		channel, appErr := p.API.GetChannel(board.ChannelID)
		if appErr == nil && channel != nil {
			teamID = channel.TeamId
		}
	}

	if teamID == "" {
		return ""
	}

	team, appErr := p.API.GetTeam(teamID)
	if appErr != nil || team == nil {
		return ""
	}

	return team.Name
}
