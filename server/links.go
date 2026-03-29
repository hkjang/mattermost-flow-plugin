package main

import (
	"fmt"
	"net/url"
	"path"
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
	siteURL := p.siteURL()
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

func (p *Plugin) buildBoardCalendarDownloadURL(boardID string) string {
	base := p.pluginBaseURL()
	if base == "" {
		return ""
	}
	return fmt.Sprintf("%s/api/v1/boards/%s/calendar.ics", base, boardID)
}

func (p *Plugin) buildBoardCalendarSubscribeURL(boardID, token string) string {
	base := p.pluginBaseURL()
	if base == "" || strings.TrimSpace(token) == "" {
		return ""
	}

	query := url.Values{}
	query.Set("token", token)
	return fmt.Sprintf("%s/calendar/%s.ics?%s", base, boardID, query.Encode())
}

func (p *Plugin) pluginBaseURL() string {
	siteURL := p.siteURL()
	if siteURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/plugins/%s", siteURL, PluginID)
}

func (p *Plugin) siteURL() string {
	config := p.API.GetConfig()
	if config == nil || config.ServiceSettings.SiteURL == nil {
		return ""
	}

	return strings.TrimRight(strings.TrimSpace(*config.ServiceSettings.SiteURL), "/")
}

func sanitizeICSFilename(name string) string {
	cleaned := strings.TrimSpace(name)
	if cleaned == "" {
		return "mattermost-flow-board.ics"
	}

	cleaned = strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
	).Replace(cleaned)
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "mattermost-flow-board.ics"
	}

	return path.Clean(cleaned) + ".ics"
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
