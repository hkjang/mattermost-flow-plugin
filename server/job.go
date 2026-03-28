package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

func (p *Plugin) runJob() {
	if p.service == nil {
		return
	}

	boards, err := p.service.ListAllBoards()
	if err != nil {
		p.API.LogError("flow background job failed to list boards", "error", err.Error())
		return
	}

	now := time.Now().UTC()
	for _, board := range boards {
		if board.ChannelID == "" || !board.Settings.PostDueSoon {
			continue
		}

		cards, err := p.service.ListCards(board.ID)
		if err != nil {
			p.API.LogError("flow background job failed to list cards", "board_id", board.ID, "error", err.Error())
			continue
		}

		for _, card := range cards {
			if err := p.processDueSoonNotification(board, card, now); err != nil {
				p.API.LogError("flow background job failed to process due soon notification", "board_id", board.ID, "card_id", card.ID, "error", err.Error())
			}
		}
	}
}

func (p *Plugin) processDueSoonNotification(board Board, card Card, now time.Time) error {
	if card.Progress >= 100 || strings.TrimSpace(card.DueDate) == "" {
		if err := p.service.store.DeleteDueSoonNotification(board.ID, card.ID); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		return nil
	}

	dueDate, ok := parseDay(card.DueDate)
	if !ok {
		return nil
	}

	notification, err := p.service.store.GetDueSoonNotification(board.ID, card.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}

	if !isDueSoonDate(dueDate, now) {
		if err == nil {
			if deleteErr := p.service.store.DeleteDueSoonNotification(board.ID, card.ID); deleteErr != nil && !errors.Is(deleteErr, ErrNotFound) {
				return deleteErr
			}
		}
		return nil
	}

	if err == nil && notification.DueDate == card.DueDate {
		return nil
	}

	if err := p.postDueSoonAlert(board, card); err != nil {
		return err
	}

	return p.service.store.SaveDueSoonNotification(DueSoonNotification{
		BoardID:    board.ID,
		CardID:     card.ID,
		DueDate:    card.DueDate,
		NotifiedAt: nowMillis(),
	})
}

func isDueSoonDate(dueDate, now time.Time) bool {
	today := startOfDay(now)
	horizon := today.Add(48 * time.Hour)
	return !dueDate.Before(today) && !dueDate.After(horizon)
}

func (p *Plugin) postDueSoonAlert(board Board, card Card) error {
	if p.botUserID == "" {
		return fmt.Errorf("flow bot user is not configured")
	}

	message := fmt.Sprintf("[Flow] Due soon: **%s** is due on **%s** in **%s**.", card.Title, card.DueDate, board.Name)
	if board.Settings.AllowMentions && len(card.AssigneeIDs) > 0 {
		mentions := make([]string, 0, len(card.AssigneeIDs))
		for _, assigneeID := range card.AssigneeIDs {
			mentions = append(mentions, fmt.Sprintf("<@%s>", assigneeID))
		}
		message += "\n" + strings.Join(mentions, " ")
	}

	props := p.buildFlowCardPostProps(board, card)
	ganttLinkURL, _ := props["gantt_link_url"].(string)
	if ganttLinkURL != "" {
		message += fmt.Sprintf("\n[Open in gantt](%s)", ganttLinkURL)
	}

	props["flow_type"] = "due_soon"
	props["due_date"] = card.DueDate
	props["summary"] = fmt.Sprintf("Due on %s in %s", card.DueDate, board.Name)
	props["link_url"] = ganttLinkURL

	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: board.ChannelID,
		Message:   message,
		Type:      FlowPostTypeDueSoon,
		Props:     props,
	}

	if _, appErr := p.API.CreatePost(post); appErr != nil {
		return fmt.Errorf("create due soon post: %w", appErr)
	}

	return nil
}
