package main

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

type FlowService struct {
	store FlowStore
}

func NewFlowService(store FlowStore) *FlowService {
	return &FlowService{store: store}
}

func (s *FlowService) ListAllBoards() ([]Board, error) {
	return s.store.ListAllBoards()
}

func (s *FlowService) ListBoards(scope ScopeQuery) ([]BoardSummary, error) {
	boards, err := s.store.ListBoards(scope.TeamID, scope.ChannelID)
	if err != nil {
		return nil, err
	}

	defaultBoardID := ""
	if scope.ChannelID != "" {
		if boardID, err := s.store.GetDefaultBoard(scope.ChannelID); err == nil {
			defaultBoardID = boardID
		}
	}

	summaries := make([]BoardSummary, 0, len(boards))
	for _, board := range boards {
		cards, err := s.store.ListCards(board.ID)
		if err != nil {
			return nil, err
		}
		columns, err := s.store.GetColumns(board.ID)
		if err != nil {
			return nil, err
		}

		activity, err := s.store.ListActivity(board.ID)
		if err != nil {
			return nil, err
		}

		summaries = append(summaries, buildBoardSummary(board, cards, columns, board.ID == defaultBoardID, latestActivity(activity)))
	}

	return summaries, nil
}

func (s *FlowService) GetBoard(boardID string) (Board, error) {
	return s.store.GetBoard(boardID)
}

func (s *FlowService) GetBoardBundle(boardID, userID string) (*BoardBundle, error) {
	board, err := s.store.GetBoard(boardID)
	if err != nil {
		return nil, err
	}

	columns, err := s.store.GetColumns(boardID)
	if err != nil {
		return nil, err
	}

	templates, err := s.store.GetTemplates(boardID)
	if err != nil {
		return nil, err
	}

	cards, err := s.store.ListCards(boardID)
	if err != nil {
		return nil, err
	}

	dependencies, err := s.store.ListDependencies(boardID)
	if err != nil {
		return nil, err
	}

	activity, err := s.store.ListActivity(boardID)
	if err != nil {
		return nil, err
	}

	preference, err := s.store.GetPreference(userID, boardID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if errors.Is(err, ErrNotFound) {
		preference = Preference{
			UserID:    userID,
			BoardID:   boardID,
			ViewType:  board.Settings.DefaultView,
			ZoomLevel: "week",
		}
	}

	summary := buildBoardSummary(board, cards, columns, false, latestActivity(activity))
	return &BoardBundle{
		Board:        board,
		Columns:      columns,
		Templates:    templates,
		Cards:        cards,
		Dependencies: dependencies,
		Activity:     activity,
		Preference:   preference,
		Summary:      summary,
	}, nil
}

func (s *FlowService) CreateBoard(actorID string, req CreateBoardRequest) (*BoardBundle, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, newValidationError("board name is required")
	}
	if req.TeamID == "" && req.ChannelID == "" {
		return nil, newValidationError("team_id or channel_id is required")
	}

	now := nowMillis()
	board := Board{
		ID:          model.NewId(),
		TeamID:      strings.TrimSpace(req.TeamID),
		ChannelID:   strings.TrimSpace(req.ChannelID),
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Visibility:  normalizeVisibility(req.Visibility, req.ChannelID),
		AdminIDs:    appendUnique(normalizeUserIDs(req.AdminIDs), actorID),
		Settings:    normalizeBoardSettings(req.Settings),
		CreatedBy:   actorID,
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     1,
	}

	columns := normalizeColumns(req.Columns, board.ID)
	if len(columns) == 0 {
		columns = defaultColumns(board.ID)
	}

	if err := s.store.SaveBoard(board); err != nil {
		return nil, err
	}
	if board.Settings.CalendarFeedEnabled {
		if _, err := s.ensureBoardCalendarFeed(board.ID, actorID); err != nil {
			return nil, err
		}
	}
	if err := s.store.SaveColumns(board.ID, columns); err != nil {
		return nil, err
	}
	if len(req.Templates) > 0 {
		templates, err := normalizeTemplates(req.Templates, board.ID, actorID, nil)
		if err != nil {
			return nil, err
		}
		if err := s.store.SaveTemplates(board.ID, templates); err != nil {
			return nil, err
		}
	}
	if board.ChannelID != "" && req.SetAsDefault {
		if err := s.store.SaveDefaultBoard(board.ChannelID, board.ID); err != nil {
			return nil, err
		}
	}

	if err := s.store.AppendActivity(board.ID, newActivity(board.ID, "board", board.ID, "board.created", actorID, nil, board)); err != nil {
		return nil, err
	}

	return s.GetBoardBundle(board.ID, actorID)
}

func (s *FlowService) UpdateBoard(actorID, boardID string, req UpdateBoardRequest) (*BoardBundle, error) {
	board, err := s.store.GetBoard(boardID)
	if err != nil {
		return nil, err
	}

	if req.Version != nil && *req.Version != board.Version {
		return nil, newConflictError("board version mismatch")
	}

	before := board
	if req.Name != nil {
		board.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		board.Description = strings.TrimSpace(*req.Description)
	}
	if req.AdminIDs != nil {
		board.AdminIDs = normalizeUserIDs(*req.AdminIDs)
	}
	if req.Settings != nil {
		board.Settings = normalizeBoardSettings(req.Settings)
	}

	board.Version++
	board.UpdatedAt = nowMillis()

	if strings.TrimSpace(board.Name) == "" {
		return nil, newValidationError("board name is required")
	}

	if err := s.store.SaveBoard(board); err != nil {
		return nil, err
	}
	if board.Settings.CalendarFeedEnabled {
		if _, err := s.ensureBoardCalendarFeed(board.ID, actorID); err != nil {
			return nil, err
		}
	}

	if req.Columns != nil {
		if err := s.updateColumns(board.ID, *req.Columns); err != nil {
			return nil, err
		}
	}
	if req.Templates != nil {
		if err := s.updateTemplates(board.ID, *req.Templates, actorID); err != nil {
			return nil, err
		}
	}

	if err := s.store.AppendActivity(board.ID, newActivity(board.ID, "board", board.ID, "board.updated", actorID, before, board)); err != nil {
		return nil, err
	}

	return s.GetBoardBundle(board.ID, actorID)
}

func (s *FlowService) DeleteBoard(actorID, boardID string) error {
	board, err := s.store.GetBoard(boardID)
	if err != nil {
		return err
	}

	if err := s.store.AppendActivity(boardID, newActivity(board.ID, "board", board.ID, "board.deleted", actorID, board, nil)); err != nil {
		return err
	}

	return s.store.DeleteBoard(boardID)
}

func (s *FlowService) ListCards(boardID string) ([]Card, error) {
	return s.store.ListCards(boardID)
}

func (s *FlowService) GetCard(cardID string) (Card, Board, error) {
	boardID, err := s.store.GetCardBoardID(cardID)
	if err != nil {
		return Card{}, Board{}, err
	}

	card, err := s.store.GetCard(boardID, cardID)
	if err != nil {
		return Card{}, Board{}, err
	}

	board, err := s.store.GetBoard(boardID)
	if err != nil {
		return Card{}, Board{}, err
	}

	return card, board, nil
}

func (s *FlowService) CreateCard(actorID string, req CreateCardRequest) (*CardMutationResult, error) {
	board, err := s.store.GetBoard(req.BoardID)
	if err != nil {
		return nil, err
	}

	columns, err := s.store.GetColumns(req.BoardID)
	if err != nil {
		return nil, err
	}
	if !columnExists(columns, req.ColumnID) {
		return nil, newValidationError("column_id is invalid")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, newValidationError("card title is required")
	}

	if err := validateDates(req.StartDate, req.DueDate); err != nil {
		return nil, err
	}

	now := nowMillis()
	cards, err := s.store.ListCards(req.BoardID)
	if err != nil {
		return nil, err
	}

	card := Card{
		ID:              model.NewId(),
		BoardID:         req.BoardID,
		ColumnID:        req.ColumnID,
		Title:           title,
		Description:     strings.TrimSpace(req.Description),
		AssigneeIDs:     normalizeUserIDs(req.AssigneeIDs),
		Labels:          normalizeLabels(req.Labels),
		Priority:        normalizePriority(req.Priority),
		StartDate:       normalizeDate(req.StartDate),
		DueDate:         normalizeDate(req.DueDate),
		Progress:        clampProgress(req.Progress),
		Milestone:       req.Milestone,
		Checklist:       normalizeChecklist(req.Checklist),
		AttachmentLinks: normalizeAttachmentLinks(req.AttachmentLinks),
		Comments:        []CardComment{},
		Position:        nextPosition(cards, req.ColumnID),
		CreatedBy:       actorID,
		UpdatedBy:       actorID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Version:         1,
	}

	if card.Progress == 0 && looksDoneColumn(columns, card.ColumnID) {
		card.Progress = 100
	}

	if err := s.store.SaveCard(card); err != nil {
		return nil, err
	}

	board.UpdatedAt = now
	board.Version++
	if err := s.store.SaveBoard(board); err != nil {
		return nil, err
	}

	if err := s.store.AppendActivity(board.ID, newActivity(board.ID, "card", card.ID, "card.created", actorID, nil, card)); err != nil {
		return nil, err
	}

	return &CardMutationResult{
		Board:      board,
		Card:       card,
		ColumnName: columnName(columns, card.ColumnID),
	}, nil
}

func (s *FlowService) UpdateCard(actorID, cardID string, req UpdateCardRequest) (*CardMutationResult, error) {
	card, board, err := s.GetCard(cardID)
	if err != nil {
		return nil, err
	}

	if req.Version != nil && *req.Version != card.Version {
		return nil, newConflictError("card version mismatch")
	}

	before := card
	if req.Title != nil {
		card.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		card.Description = strings.TrimSpace(*req.Description)
	}
	if req.AssigneeIDs != nil {
		card.AssigneeIDs = normalizeUserIDs(*req.AssigneeIDs)
	}
	if req.Labels != nil {
		card.Labels = normalizeLabels(*req.Labels)
	}
	if req.Priority != nil {
		card.Priority = normalizePriority(*req.Priority)
	}
	if req.StartDate != nil {
		card.StartDate = normalizeDate(*req.StartDate)
	}
	if req.DueDate != nil {
		card.DueDate = normalizeDate(*req.DueDate)
	}
	if err := validateDates(card.StartDate, card.DueDate); err != nil {
		return nil, err
	}
	if req.Progress != nil {
		card.Progress = clampProgress(*req.Progress)
	}
	if req.Milestone != nil {
		card.Milestone = *req.Milestone
	}
	if req.Checklist != nil {
		card.Checklist = normalizeChecklist(*req.Checklist)
	}
	if req.AttachmentLinks != nil {
		card.AttachmentLinks = normalizeAttachmentLinks(*req.AttachmentLinks)
	}
	if strings.TrimSpace(card.Title) == "" {
		return nil, newValidationError("card title is required")
	}

	card.Version++
	card.UpdatedBy = actorID
	card.UpdatedAt = nowMillis()

	if err := s.store.SaveCard(card); err != nil {
		return nil, err
	}

	board.Version++
	board.UpdatedAt = card.UpdatedAt
	if err := s.store.SaveBoard(board); err != nil {
		return nil, err
	}

	if err := s.store.AppendActivity(board.ID, newActivity(board.ID, "card", card.ID, "card.updated", actorID, before, card)); err != nil {
		return nil, err
	}

	columns, err := s.store.GetColumns(board.ID)
	if err != nil {
		return nil, err
	}

	return &CardMutationResult{
		Board:      board,
		Card:       card,
		ColumnName: columnName(columns, card.ColumnID),
	}, nil
}

func (s *FlowService) MoveCard(actorID, cardID string, req MoveCardRequest) (*CardMoveResult, error) {
	card, board, err := s.GetCard(cardID)
	if err != nil {
		return nil, err
	}
	if req.Version != 0 && req.Version != card.Version {
		return nil, newConflictError("card version mismatch")
	}

	columns, err := s.store.GetColumns(board.ID)
	if err != nil {
		return nil, err
	}
	if !columnExists(columns, req.TargetColumnID) {
		return nil, newValidationError("target_column_id is invalid")
	}

	cards, err := s.store.ListCards(board.ID)
	if err != nil {
		return nil, err
	}

	before := card
	fromColumnName := columnName(columns, card.ColumnID)
	toColumnName := columnName(columns, req.TargetColumnID)

	reordered, movedCard, err := moveCardInList(cards, card.ID, req.TargetColumnID, req.TargetIndex)
	if err != nil {
		return nil, err
	}

	if looksDoneColumn(columns, req.TargetColumnID) && movedCard.Progress < 100 {
		movedCard.Progress = 100
	}
	movedCard.Version++
	movedCard.UpdatedAt = nowMillis()
	movedCard.UpdatedBy = actorID

	for index, current := range reordered {
		if current.ID == movedCard.ID {
			reordered[index] = movedCard
			current = movedCard
		}
		if err := s.store.SaveCard(current); err != nil {
			return nil, err
		}
	}

	board.Version++
	board.UpdatedAt = movedCard.UpdatedAt
	if err := s.store.SaveBoard(board); err != nil {
		return nil, err
	}

	if err := s.store.AppendActivity(board.ID, newActivity(board.ID, "card", movedCard.ID, "card.moved", actorID, before, movedCard)); err != nil {
		return nil, err
	}

	return &CardMoveResult{
		Board:          board,
		Card:           movedCard,
		FromColumnName: fromColumnName,
		ToColumnName:   toColumnName,
	}, nil
}

func (s *FlowService) AssignCardToUser(actorID, cardID, assigneeID string) (*CardMutationResult, bool, error) {
	card, board, err := s.GetCard(cardID)
	if err != nil {
		return nil, false, err
	}

	assigneeID = strings.TrimSpace(assigneeID)
	if assigneeID == "" {
		return nil, false, newValidationError("assignee_id is required")
	}
	if containsString(card.AssigneeIDs, assigneeID) {
		columns, columnErr := s.store.GetColumns(board.ID)
		if columnErr != nil {
			return nil, false, columnErr
		}
		return &CardMutationResult{
			Board:      board,
			Card:       card,
			ColumnName: columnName(columns, card.ColumnID),
		}, false, nil
	}

	before := card
	card.AssigneeIDs = appendUnique(card.AssigneeIDs, assigneeID)
	card.Version++
	card.UpdatedBy = actorID
	card.UpdatedAt = nowMillis()

	if err := s.store.SaveCard(card); err != nil {
		return nil, false, err
	}

	board.Version++
	board.UpdatedAt = card.UpdatedAt
	if err := s.store.SaveBoard(board); err != nil {
		return nil, false, err
	}

	if err := s.store.AppendActivity(board.ID, newActivity(board.ID, "card", card.ID, "card.assignee_added", actorID, before, card)); err != nil {
		return nil, false, err
	}

	columns, err := s.store.GetColumns(board.ID)
	if err != nil {
		return nil, false, err
	}

	return &CardMutationResult{
		Board:      board,
		Card:       card,
		ColumnName: columnName(columns, card.ColumnID),
	}, true, nil
}

func (s *FlowService) SetCardDueDate(actorID, cardID, dueDate string) (*CardMutationResult, bool, error) {
	card, board, err := s.GetCard(cardID)
	if err != nil {
		return nil, false, err
	}

	normalizedDueDate := normalizeDate(dueDate)
	if normalizedDueDate == "" {
		return nil, false, newValidationError("due_date is required")
	}
	if card.DueDate == normalizedDueDate {
		columns, columnErr := s.store.GetColumns(board.ID)
		if columnErr != nil {
			return nil, false, columnErr
		}
		return &CardMutationResult{
			Board:      board,
			Card:       card,
			ColumnName: columnName(columns, card.ColumnID),
		}, false, nil
	}

	before := card
	card.DueDate = normalizedDueDate
	if err := validateDates(card.StartDate, card.DueDate); err != nil {
		return nil, false, err
	}

	card.Version++
	card.UpdatedBy = actorID
	card.UpdatedAt = nowMillis()

	if err := s.store.SaveCard(card); err != nil {
		return nil, false, err
	}

	board.Version++
	board.UpdatedAt = card.UpdatedAt
	if err := s.store.SaveBoard(board); err != nil {
		return nil, false, err
	}

	if err := s.store.AppendActivity(board.ID, newActivity(board.ID, "card", card.ID, "card.due_date_updated", actorID, before, card)); err != nil {
		return nil, false, err
	}

	columns, err := s.store.GetColumns(board.ID)
	if err != nil {
		return nil, false, err
	}

	return &CardMutationResult{
		Board:      board,
		Card:       card,
		ColumnName: columnName(columns, card.ColumnID),
	}, true, nil
}

func (s *FlowService) CompleteNextChecklistItem(actorID, cardID string) (*CardMutationResult, string, bool, error) {
	card, board, err := s.GetCard(cardID)
	if err != nil {
		return nil, "", false, err
	}

	nextIndex := -1
	nextItemText := ""
	for index, item := range card.Checklist {
		if item.Completed {
			continue
		}
		nextIndex = index
		nextItemText = item.Text
		break
	}

	columns, err := s.store.GetColumns(board.ID)
	if err != nil {
		return nil, "", false, err
	}

	if nextIndex == -1 {
		return &CardMutationResult{
			Board:      board,
			Card:       card,
			ColumnName: columnName(columns, card.ColumnID),
		}, "", false, nil
	}

	before := card
	card.Checklist[nextIndex].Completed = true

	completedCount := 0
	for _, item := range card.Checklist {
		if item.Completed {
			completedCount++
		}
	}
	if len(card.Checklist) > 0 {
		derivedProgress := int(float64(completedCount*100) / float64(len(card.Checklist)))
		if derivedProgress > card.Progress {
			card.Progress = derivedProgress
		}
	}

	card.Version++
	card.UpdatedBy = actorID
	card.UpdatedAt = nowMillis()

	if err := s.store.SaveCard(card); err != nil {
		return nil, "", false, err
	}

	board.Version++
	board.UpdatedAt = card.UpdatedAt
	if err := s.store.SaveBoard(board); err != nil {
		return nil, "", false, err
	}

	if err := s.store.AppendActivity(board.ID, newActivity(board.ID, "card", card.ID, "card.checklist_item_completed", actorID, before, card)); err != nil {
		return nil, "", false, err
	}

	return &CardMutationResult{
		Board:      board,
		Card:       card,
		ColumnName: columnName(columns, card.ColumnID),
	}, nextItemText, true, nil
}

func (s *FlowService) CompleteCard(actorID, cardID string) (*CardMutationResult, bool, error) {
	card, board, err := s.GetCard(cardID)
	if err != nil {
		return nil, false, err
	}

	columns, err := s.store.GetColumns(board.ID)
	if err != nil {
		return nil, false, err
	}

	if doneColumn, ok := findDoneColumn(columns); ok && card.ColumnID != doneColumn.ID {
		cards, listErr := s.store.ListCards(board.ID)
		if listErr != nil {
			return nil, false, listErr
		}

		moveResult, moveErr := s.MoveCard(actorID, cardID, MoveCardRequest{
			TargetColumnID: doneColumn.ID,
			TargetIndex:    nextPosition(cards, doneColumn.ID),
		})
		if moveErr != nil {
			return nil, false, moveErr
		}

		return &CardMutationResult{
			Board:      moveResult.Board,
			Card:       moveResult.Card,
			ColumnName: moveResult.ToColumnName,
		}, true, nil
	}

	if card.Progress >= 100 {
		return &CardMutationResult{
			Board:      board,
			Card:       card,
			ColumnName: columnName(columns, card.ColumnID),
		}, false, nil
	}

	progress := 100
	result, updateErr := s.UpdateCard(actorID, cardID, UpdateCardRequest{
		Progress: &progress,
	})
	if updateErr != nil {
		return nil, false, updateErr
	}

	return result, true, nil
}

func (s *FlowService) MoveCardToNextColumn(actorID, cardID string) (*CardMoveResult, error) {
	card, board, err := s.GetCard(cardID)
	if err != nil {
		return nil, err
	}

	columns, err := s.store.GetColumns(board.ID)
	if err != nil {
		return nil, err
	}

	currentIndex := -1
	for index, column := range columns {
		if column.ID == card.ColumnID {
			currentIndex = index
			break
		}
	}
	if currentIndex == -1 {
		return nil, newValidationError("card column is invalid")
	}
	if currentIndex >= len(columns)-1 {
		return nil, newValidationError("card is already in the last column")
	}

	cards, err := s.store.ListCards(board.ID)
	if err != nil {
		return nil, err
	}

	targetColumnID := columns[currentIndex+1].ID
	return s.MoveCard(actorID, cardID, MoveCardRequest{
		TargetColumnID: targetColumnID,
		TargetIndex:    nextPosition(cards, targetColumnID),
	})
}

func (s *FlowService) AddDependency(actorID, sourceCardID string, req AddDependencyRequest) (*DependencyMutationResult, error) {
	sourceCard, board, err := s.GetCard(sourceCardID)
	if err != nil {
		return nil, err
	}
	targetCard, targetBoard, err := s.GetCard(strings.TrimSpace(req.TargetCardID))
	if err != nil {
		return nil, err
	}
	if targetBoard.ID != board.ID {
		return nil, newValidationError("target card must be in the same board")
	}
	if sourceCard.ID == targetCard.ID {
		return nil, newValidationError("dependency cannot point to the same card")
	}

	dependencies, err := s.store.ListDependencies(board.ID)
	if err != nil {
		return nil, err
	}

	for _, dependency := range dependencies {
		if dependency.SourceCardID == sourceCard.ID && dependency.TargetCardID == targetCard.ID {
			return nil, newConflictError("dependency already exists")
		}
	}

	dependency := Dependency{
		ID:           model.NewId(),
		BoardID:      board.ID,
		SourceCardID: sourceCard.ID,
		TargetCardID: targetCard.ID,
		Type:         normalizeDependencyType(req.Type),
		CreatedBy:    actorID,
		CreatedAt:    nowMillis(),
	}

	dependencies = append(dependencies, dependency)
	if err := s.store.SaveDependencies(board.ID, dependencies); err != nil {
		return nil, err
	}
	if err := s.store.AppendActivity(board.ID, newActivity(board.ID, "dependency", dependency.ID, "dependency.created", actorID, nil, dependency)); err != nil {
		return nil, err
	}

	return &DependencyMutationResult{Board: board, Dependency: dependency}, nil
}

func (s *FlowService) AddComment(actorID, cardID string, req AddCommentRequest) (*CommentMutationResult, error) {
	card, board, err := s.GetCard(cardID)
	if err != nil {
		return nil, err
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		return nil, newValidationError("comment message is required")
	}

	comment := CardComment{
		ID:        model.NewId(),
		CardID:    card.ID,
		UserID:    actorID,
		Message:   message,
		CreatedAt: nowMillis(),
	}

	card.Comments = append(card.Comments, comment)
	card.Version++
	card.UpdatedBy = actorID
	card.UpdatedAt = comment.CreatedAt

	if err := s.store.SaveCard(card); err != nil {
		return nil, err
	}

	board.Version++
	board.UpdatedAt = card.UpdatedAt
	if err := s.store.SaveBoard(board); err != nil {
		return nil, err
	}

	if err := s.store.AppendActivity(board.ID, newActivity(board.ID, "comment", comment.ID, "comment.created", actorID, nil, comment)); err != nil {
		return nil, err
	}

	return &CommentMutationResult{
		Board:   board,
		Card:    card,
		Comment: comment,
	}, nil
}

func (s *FlowService) GetGantt(boardID string) (*GanttViewData, error) {
	board, err := s.store.GetBoard(boardID)
	if err != nil {
		return nil, err
	}
	columns, err := s.store.GetColumns(boardID)
	if err != nil {
		return nil, err
	}
	cards, err := s.store.ListCards(boardID)
	if err != nil {
		return nil, err
	}
	dependencies, err := s.store.ListDependencies(boardID)
	if err != nil {
		return nil, err
	}

	return &GanttViewData{
		Board:        board,
		Columns:      columns,
		Tasks:        cards,
		Dependencies: dependencies,
	}, nil
}

func (s *FlowService) ListActivity(boardID string) ([]Activity, error) {
	return s.store.ListActivity(boardID)
}

func (s *FlowService) SavePreference(userID, boardID string, req SavePreferenceRequest) (*Preference, error) {
	board, err := s.store.GetBoard(boardID)
	if err != nil {
		return nil, err
	}

	preference := Preference{
		UserID:    userID,
		BoardID:   boardID,
		ViewType:  normalizeViewType(req.ViewType, board.Settings.DefaultView),
		Filters:   req.Filters,
		ZoomLevel: normalizeZoomLevel(req.ZoomLevel),
		UpdatedAt: nowMillis(),
	}

	if err := s.store.SavePreference(preference); err != nil {
		return nil, err
	}
	return &preference, nil
}

func (s *FlowService) GetBoardSummary(boardID string) (BoardSummary, error) {
	board, err := s.store.GetBoard(boardID)
	if err != nil {
		return BoardSummary{}, err
	}

	cards, err := s.store.ListCards(boardID)
	if err != nil {
		return BoardSummary{}, err
	}

	columns, err := s.store.GetColumns(boardID)
	if err != nil {
		return BoardSummary{}, err
	}

	isDefault := false
	if board.ChannelID != "" {
		defaultBoardID, defaultErr := s.store.GetDefaultBoard(board.ChannelID)
		if defaultErr == nil {
			isDefault = defaultBoardID == boardID
		}
	}

	activity, err := s.store.ListActivity(boardID)
	if err != nil {
		return BoardSummary{}, err
	}

	return buildBoardSummary(board, cards, columns, isDefault, latestActivity(activity)), nil
}

func (s *FlowService) GetBoardCalendarFeed(boardID string) (BoardCalendarFeed, error) {
	return s.store.GetCalendarFeed(boardID)
}

func (s *FlowService) EnsureBoardCalendarFeed(boardID, actorID string) (BoardCalendarFeed, error) {
	return s.ensureBoardCalendarFeed(boardID, actorID)
}

func (s *FlowService) RotateBoardCalendarFeed(boardID, actorID string) (BoardCalendarFeed, error) {
	board, err := s.store.GetBoard(boardID)
	if err != nil {
		return BoardCalendarFeed{}, err
	}
	if !board.Settings.CalendarFeedEnabled {
		return BoardCalendarFeed{}, newValidationError("calendar feed is disabled")
	}

	feed := BoardCalendarFeed{
		BoardID:   boardID,
		Token:     model.NewId(),
		UpdatedBy: actorID,
		UpdatedAt: nowMillis(),
	}
	if err := s.store.SaveCalendarFeed(feed); err != nil {
		return BoardCalendarFeed{}, err
	}

	if err := s.store.AppendActivity(boardID, newActivity(boardID, "board", boardID, "board.calendar.rotated", actorID, nil, map[string]any{
		"calendar_feed_enabled": true,
	})); err != nil {
		return BoardCalendarFeed{}, err
	}

	return feed, nil
}

func (s *FlowService) BuildColumnCardIDs(boardID string) (map[string][]string, error) {
	columns, err := s.store.GetColumns(boardID)
	if err != nil {
		return nil, err
	}

	cards, err := s.store.ListCards(boardID)
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]Card, len(columns))
	for _, column := range columns {
		grouped[column.ID] = []Card{}
	}

	for _, card := range cards {
		grouped[card.ColumnID] = append(grouped[card.ColumnID], card)
	}

	snapshot := make(map[string][]string, len(grouped))
	for _, column := range columns {
		columnCards := grouped[column.ID]
		sort.Slice(columnCards, func(i, j int) bool {
			if columnCards[i].Position == columnCards[j].Position {
				return columnCards[i].CreatedAt < columnCards[j].CreatedAt
			}
			return columnCards[i].Position < columnCards[j].Position
		})

		ids := make([]string, 0, len(columnCards))
		for _, card := range columnCards {
			ids = append(ids, card.ID)
		}
		snapshot[column.ID] = ids
	}

	return snapshot, nil
}

func (s *FlowService) updateColumns(boardID string, inputs []BoardColumnInput) error {
	existingColumns, err := s.store.GetColumns(boardID)
	if err != nil {
		return err
	}
	if len(inputs) == 0 {
		return newValidationError("at least one column is required")
	}

	nextColumns := normalizeColumns(inputs, boardID)
	cards, err := s.store.ListCards(boardID)
	if err != nil {
		return err
	}

	fallbackColumnID := nextColumns[0].ID
	for index, card := range cards {
		if !columnExists(nextColumns, card.ColumnID) {
			cards[index].ColumnID = fallbackColumnID
		}
	}

	reindexCards(cards)
	for _, card := range cards {
		if err := s.store.SaveCard(card); err != nil {
			return err
		}
	}

	if len(existingColumns) != len(nextColumns) || !sameColumnShape(existingColumns, nextColumns) {
		if err := s.store.SaveColumns(boardID, nextColumns); err != nil {
			return err
		}
	}

	return nil
}

func (s *FlowService) updateTemplates(boardID string, inputs []CardTemplateInput, actorID string) error {
	existingTemplates, err := s.store.GetTemplates(boardID)
	if err != nil {
		return err
	}

	nextTemplates, err := normalizeTemplates(inputs, boardID, actorID, existingTemplates)
	if err != nil {
		return err
	}

	return s.store.SaveTemplates(boardID, nextTemplates)
}

func buildBoardSummary(board Board, cards []Card, columns []BoardColumn, isDefault bool, recentActivity *Activity) BoardSummary {
	now := time.Now().UTC()
	assignees := make([]string, 0)
	overdue := 0
	dueSoon := 0

	for _, card := range cards {
		assignees = appendAllUnique(assignees, card.AssigneeIDs)
		if dueDate, ok := parseDay(card.DueDate); ok && card.Progress < 100 {
			if dueDate.Before(startOfDay(now)) {
				overdue++
			}
			if !dueDate.Before(startOfDay(now)) && dueDate.Before(startOfDay(now).Add(72*time.Hour)) {
				dueSoon++
			}
		}
	}

	return BoardSummary{
		Board:          board,
		CardCount:      len(cards),
		OverdueCount:   overdue,
		DueSoonCount:   dueSoon,
		DefaultBoard:   isDefault,
		Columns:        len(columns),
		Assignees:      assignees,
		RecentActivity: recentActivity,
	}
}

func latestActivity(activity []Activity) *Activity {
	if len(activity) == 0 {
		return nil
	}

	item := activity[0]
	return &item
}

func defaultColumns(boardID string) []BoardColumn {
	names := []string{"Todo", "In Progress", "Review", "Done"}
	columns := make([]BoardColumn, 0, len(names))
	for index, name := range names {
		columns = append(columns, BoardColumn{
			ID:        model.NewId(),
			BoardID:   boardID,
			Name:      name,
			SortOrder: index,
		})
	}
	return columns
}

func normalizeColumns(inputs []BoardColumnInput, boardID string) []BoardColumn {
	if len(inputs) == 0 {
		return nil
	}
	columns := make([]BoardColumn, 0, len(inputs))
	for index, input := range inputs {
		name := strings.TrimSpace(input.Name)
		if name == "" {
			continue
		}
		columnID := strings.TrimSpace(input.ID)
		if columnID == "" {
			columnID = model.NewId()
		}
		sortOrder := input.SortOrder
		if sortOrder == 0 && index > 0 {
			sortOrder = index
		}
		columns = append(columns, BoardColumn{
			ID:        columnID,
			BoardID:   boardID,
			Name:      name,
			SortOrder: sortOrder,
			WIPLimit:  input.WIPLimit,
		})
	}
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].SortOrder < columns[j].SortOrder
	})
	for index := range columns {
		columns[index].SortOrder = index
	}
	return columns
}

func normalizeTemplates(inputs []CardTemplateInput, boardID, actorID string, existing []CardTemplate) ([]CardTemplate, error) {
	if len(inputs) == 0 {
		return []CardTemplate{}, nil
	}

	existingByID := make(map[string]CardTemplate, len(existing))
	for _, template := range existing {
		existingByID[template.ID] = template
	}

	now := nowMillis()
	templates := make([]CardTemplate, 0, len(inputs))

	for _, input := range inputs {
		name := strings.TrimSpace(input.Name)
		if name == "" {
			continue
		}

		startOffsetDays, err := normalizeTemplateOffset(input.StartOffsetDays)
		if err != nil {
			return nil, err
		}

		dueOffsetDays, err := normalizeTemplateOffset(input.DueOffsetDays)
		if err != nil {
			return nil, err
		}

		if startOffsetDays != nil && dueOffsetDays != nil && *dueOffsetDays < *startOffsetDays {
			return nil, newValidationError("template due_offset_days must be on or after start_offset_days")
		}

		templateID := strings.TrimSpace(input.ID)
		createdBy := actorID
		createdAt := now
		if previous, ok := existingByID[templateID]; ok {
			createdBy = previous.CreatedBy
			createdAt = previous.CreatedAt
		}
		if templateID == "" {
			templateID = model.NewId()
		}

		templates = append(templates, CardTemplate{
			ID:              templateID,
			BoardID:         boardID,
			Name:            name,
			Title:           strings.TrimSpace(input.Title),
			Description:     strings.TrimSpace(input.Description),
			Labels:          normalizeLabels(input.Labels),
			Priority:        normalizePriority(input.Priority),
			StartOffsetDays: startOffsetDays,
			DueOffsetDays:   dueOffsetDays,
			Milestone:       input.Milestone,
			Checklist:       normalizeChecklist(input.Checklist),
			AttachmentLinks: normalizeAttachmentLinks(input.AttachmentLinks),
			CreatedBy:       createdBy,
			CreatedAt:       createdAt,
			UpdatedAt:       now,
		})
	}

	sort.Slice(templates, func(i, j int) bool {
		if templates[i].UpdatedAt == templates[j].UpdatedAt {
			return templates[i].Name < templates[j].Name
		}
		return templates[i].UpdatedAt > templates[j].UpdatedAt
	})

	return templates, nil
}

func validateDates(startDate, dueDate string) error {
	start, okStart := parseDay(startDate)
	due, okDue := parseDay(dueDate)
	if !okStart || !okDue {
		return nil
	}
	if due.Before(start) {
		return newValidationError("due_date must be on or after start_date")
	}
	return nil
}

func normalizeDate(value string) string {
	date := strings.TrimSpace(value)
	if date == "" {
		return ""
	}
	if parsed, err := time.Parse("2006-01-02", date); err == nil {
		return parsed.Format("2006-01-02")
	}
	return ""
}

func normalizeTemplateOffset(value *int) (*int, error) {
	if value == nil {
		return nil, nil
	}

	if *value < 0 {
		return nil, newValidationError("template offsets cannot be negative")
	}

	offset := *value
	if offset > 365 {
		offset = 365
	}
	return &offset, nil
}

func parseDay(value string) (time.Time, bool) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func normalizeVisibility(value, channelID string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "team", "channel":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		if strings.TrimSpace(channelID) != "" {
			return "channel"
		}
		return "team"
	}
}

func normalizeBoardSettings(settings *BoardSettings) BoardSettings {
	defaults := BoardSettings{
		PostUpdates:         true,
		PostDueSoon:         false,
		AllowMentions:       true,
		DefaultView:         "board",
		CalendarFeedEnabled: false,
	}
	if settings == nil {
		return defaults
	}
	defaults.PostUpdates = settings.PostUpdates
	defaults.PostDueSoon = settings.PostDueSoon
	defaults.AllowMentions = settings.AllowMentions
	defaults.DefaultView = normalizeViewType(settings.DefaultView, defaults.DefaultView)
	defaults.CalendarFeedEnabled = settings.CalendarFeedEnabled
	return defaults
}

func (s *FlowService) ensureBoardCalendarFeed(boardID, actorID string) (BoardCalendarFeed, error) {
	board, err := s.store.GetBoard(boardID)
	if err != nil {
		return BoardCalendarFeed{}, err
	}
	if !board.Settings.CalendarFeedEnabled {
		return BoardCalendarFeed{}, newValidationError("calendar feed is disabled")
	}

	feed, err := s.store.GetCalendarFeed(boardID)
	switch {
	case err == nil && strings.TrimSpace(feed.Token) != "":
		return feed, nil
	case err != nil && !errors.Is(err, ErrNotFound):
		return BoardCalendarFeed{}, err
	}

	feed = BoardCalendarFeed{
		BoardID:   boardID,
		Token:     model.NewId(),
		UpdatedBy: actorID,
		UpdatedAt: nowMillis(),
	}
	if err := s.store.SaveCalendarFeed(feed); err != nil {
		return BoardCalendarFeed{}, err
	}
	return feed, nil
}

func normalizeViewType(value, fallback string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "board", "gantt", "dashboard":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		if fallback == "gantt" {
			return "gantt"
		}
		if fallback == "dashboard" {
			return "dashboard"
		}
		return "board"
	}
}

func normalizeZoomLevel(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "day", "week", "month":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "week"
	}
}

func normalizePriority(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "low", "normal", "high", "urgent":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "normal"
	}
}

func normalizeDependencyType(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "finish_to_start", "start_to_start", "finish_to_finish":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "finish_to_start"
	}
}

func normalizeLabels(values []string) []string {
	labels := make([]string, 0, len(values))
	for _, value := range values {
		label := strings.TrimSpace(value)
		if label == "" {
			continue
		}
		labels = appendUnique(labels, label)
	}
	sort.Strings(labels)
	return labels
}

func normalizeUserIDs(values []string) []string {
	userIDs := make([]string, 0, len(values))
	for _, value := range values {
		userID := strings.TrimSpace(value)
		if userID == "" {
			continue
		}
		userIDs = appendUnique(userIDs, userID)
	}
	sort.Strings(userIDs)
	return userIDs
}

func normalizeChecklist(items []ChecklistItem) []ChecklistItem {
	next := make([]ChecklistItem, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}
		if strings.TrimSpace(item.ID) == "" {
			item.ID = model.NewId()
		}
		item.Text = text
		next = append(next, item)
	}
	return next
}

func normalizeAttachmentLinks(links []AttachmentLink) []AttachmentLink {
	next := make([]AttachmentLink, 0, len(links))
	for _, link := range links {
		url := strings.TrimSpace(link.URL)
		if url == "" {
			continue
		}
		if strings.TrimSpace(link.ID) == "" {
			link.ID = model.NewId()
		}
		link.URL = url
		link.Title = strings.TrimSpace(link.Title)
		next = append(next, link)
	}
	return next
}

func clampProgress(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func nextPosition(cards []Card, columnID string) int {
	maxPosition := -1
	for _, card := range cards {
		if card.ColumnID == columnID && card.Position > maxPosition {
			maxPosition = card.Position
		}
	}
	return maxPosition + 1
}

func sortCards(cards []Card) {
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].ColumnID == cards[j].ColumnID {
			if cards[i].Position == cards[j].Position {
				return cards[i].CreatedAt < cards[j].CreatedAt
			}
			return cards[i].Position < cards[j].Position
		}
		if cards[i].UpdatedAt == cards[j].UpdatedAt {
			return cards[i].CreatedAt < cards[j].CreatedAt
		}
		return cards[i].UpdatedAt > cards[j].UpdatedAt
	})
}

func reindexCards(cards []Card) {
	grouped := map[string][]int{}
	for index, card := range cards {
		grouped[card.ColumnID] = append(grouped[card.ColumnID], index)
	}
	for _, indexes := range grouped {
		sort.Slice(indexes, func(i, j int) bool {
			return cards[indexes[i]].Position < cards[indexes[j]].Position
		})
		for position, cardIndex := range indexes {
			cards[cardIndex].Position = position
		}
	}
}

func columnExists(columns []BoardColumn, columnID string) bool {
	for _, column := range columns {
		if column.ID == columnID {
			return true
		}
	}
	return false
}

func columnName(columns []BoardColumn, columnID string) string {
	for _, column := range columns {
		if column.ID == columnID {
			return column.Name
		}
	}
	return "Column"
}

func looksDoneColumn(columns []BoardColumn, columnID string) bool {
	return strings.Contains(strings.ToLower(columnName(columns, columnID)), "done")
}

func findDoneColumn(columns []BoardColumn) (BoardColumn, bool) {
	for _, column := range columns {
		if strings.Contains(strings.ToLower(column.Name), "done") {
			return column, true
		}
	}

	return BoardColumn{}, false
}

func moveCardInList(cards []Card, cardID, targetColumnID string, targetIndex int) ([]Card, Card, error) {
	next := make([]Card, len(cards))
	copy(next, cards)
	reindexCards(next)

	foundIndex := -1
	movedCard := Card{}
	for index, card := range next {
		if card.ID == cardID {
			foundIndex = index
			movedCard = card
			break
		}
	}
	if foundIndex == -1 {
		return nil, Card{}, newNotFoundError("card not found")
	}

	movedCard.ColumnID = targetColumnID

	remaining := make([]Card, 0, len(next)-1)
	targetCards := make([]Card, 0)
	for _, card := range next {
		if card.ID == cardID {
			continue
		}
		if card.ColumnID == targetColumnID {
			targetCards = append(targetCards, card)
		} else {
			remaining = append(remaining, card)
		}
	}

	sort.Slice(targetCards, func(i, j int) bool {
		return targetCards[i].Position < targetCards[j].Position
	})

	if targetIndex < 0 {
		targetIndex = 0
	}
	if targetIndex > len(targetCards) {
		targetIndex = len(targetCards)
	}

	inserted := make([]Card, 0, len(targetCards)+1)
	inserted = append(inserted, targetCards[:targetIndex]...)
	inserted = append(inserted, movedCard)
	inserted = append(inserted, targetCards[targetIndex:]...)

	result := make([]Card, 0, len(next))
	result = append(result, remaining...)
	result = append(result, inserted...)
	reindexCards(result)

	for _, card := range result {
		if card.ID == movedCard.ID {
			movedCard = card
			break
		}
	}

	return result, movedCard, nil
}

func sameColumnShape(left, right []BoardColumn) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].ID != right[index].ID || left[index].Name != right[index].Name || left[index].WIPLimit != right[index].WIPLimit {
			return false
		}
	}
	return true
}

func newActivity(boardID, entityType, entityID, action, actorID string, before, after any) Activity {
	return Activity{
		ID:         model.NewId(),
		BoardID:    boardID,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		ActorID:    actorID,
		Before:     marshalJSON(before),
		After:      marshalJSON(after),
		CreatedAt:  nowMillis(),
	}
}

func marshalJSON(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return raw
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func appendAllUnique(target []string, values []string) []string {
	for _, value := range values {
		target = appendUnique(target, value)
	}
	return target
}

func removeString(values []string, target string) []string {
	next := make([]string, 0, len(values))
	for _, value := range values {
		if value != target {
			next = append(next, value)
		}
	}
	return next
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
