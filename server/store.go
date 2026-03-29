package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/mattermost/mattermost/server/public/model"
)

type kvAPI interface {
	KVGet(key string) ([]byte, *model.AppError)
	KVSet(key string, value []byte) *model.AppError
	KVDelete(key string) *model.AppError
}

type FlowStore interface {
	ListAllBoards() ([]Board, error)
	ListBoards(teamID, channelID string) ([]Board, error)
	GetBoard(boardID string) (Board, error)
	SaveBoard(board Board) error
	DeleteBoard(boardID string) error
	GetColumns(boardID string) ([]BoardColumn, error)
	SaveColumns(boardID string, columns []BoardColumn) error
	GetTemplates(boardID string) ([]CardTemplate, error)
	SaveTemplates(boardID string, templates []CardTemplate) error
	ListCards(boardID string) ([]Card, error)
	GetCard(boardID, cardID string) (Card, error)
	GetCardBoardID(cardID string) (string, error)
	SaveCard(card Card) error
	DeleteCard(boardID, cardID string) error
	ListDependencies(boardID string) ([]Dependency, error)
	SaveDependencies(boardID string, dependencies []Dependency) error
	ListActivity(boardID string) ([]Activity, error)
	AppendActivity(boardID string, activity Activity) error
	GetPreference(userID, boardID string) (Preference, error)
	SavePreference(preference Preference) error
	GetDefaultBoard(channelID string) (string, error)
	SaveDefaultBoard(channelID, boardID string) error
	GetDueSoonNotification(boardID, cardID string) (DueSoonNotification, error)
	SaveDueSoonNotification(notification DueSoonNotification) error
	DeleteDueSoonNotification(boardID, cardID string) error
}

type kvStore struct {
	api kvAPI
}

func newKVStore(api kvAPI) FlowStore {
	return &kvStore{api: api}
}

const (
	allBoardsIndexKey = "boards:index"
)

func boardKey(boardID string) string {
	return fmt.Sprintf("board:%s", boardID)
}

func boardColumnsKey(boardID string) string {
	return fmt.Sprintf("board:%s:columns", boardID)
}

func boardTemplatesKey(boardID string) string {
	return fmt.Sprintf("board:%s:templates", boardID)
}

func boardCardIDsKey(boardID string) string {
	return fmt.Sprintf("board:%s:card_ids", boardID)
}

func boardCardKey(boardID, cardID string) string {
	return fmt.Sprintf("board:%s:cards:%s", boardID, cardID)
}

func cardBoardKey(cardID string) string {
	return fmt.Sprintf("card:%s:board", cardID)
}

func boardDepsKey(boardID string) string {
	return fmt.Sprintf("board:%s:deps", boardID)
}

func boardActivityKey(boardID string) string {
	return fmt.Sprintf("board:%s:activity", boardID)
}

func teamBoardsKey(teamID string) string {
	return fmt.Sprintf("team:%s:boards", teamID)
}

func channelBoardsKey(channelID string) string {
	return fmt.Sprintf("channel:%s:boards", channelID)
}

func userPreferenceKey(userID, boardID string) string {
	return fmt.Sprintf("user:%s:prefs:%s", userID, boardID)
}

func channelDefaultBoardKey(channelID string) string {
	return fmt.Sprintf("channel:%s:defaultBoard", channelID)
}

func dueSoonNotificationKey(boardID, cardID string) string {
	return fmt.Sprintf("board:%s:due_soon:%s", boardID, cardID)
}

func (s *kvStore) ListAllBoards() ([]Board, error) {
	boardIDs, err := s.loadStringSlice(allBoardsIndexKey)
	if err != nil {
		return nil, err
	}

	boards := make([]Board, 0, len(boardIDs))
	for _, boardID := range boardIDs {
		board, err := s.GetBoard(boardID)
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return nil, err
		}
		boards = append(boards, board)
	}

	sort.Slice(boards, func(i, j int) bool {
		return boards[i].UpdatedAt > boards[j].UpdatedAt
	})

	return boards, nil
}

func (s *kvStore) ListBoards(teamID, channelID string) ([]Board, error) {
	boardIDs := make([]string, 0)

	if channelID != "" {
		ids, err := s.loadStringSlice(channelBoardsKey(channelID))
		if err != nil {
			return nil, err
		}
		boardIDs = appendAllUnique(boardIDs, ids)
	}

	if teamID != "" {
		ids, err := s.loadStringSlice(teamBoardsKey(teamID))
		if err != nil {
			return nil, err
		}
		boardIDs = appendAllUnique(boardIDs, ids)
	}

	boards := make([]Board, 0, len(boardIDs))
	for _, boardID := range boardIDs {
		board, err := s.GetBoard(boardID)
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return nil, err
		}
		boards = append(boards, board)
	}

	sort.Slice(boards, func(i, j int) bool {
		return boards[i].UpdatedAt > boards[j].UpdatedAt
	})

	return boards, nil
}

func (s *kvStore) GetBoard(boardID string) (Board, error) {
	var board Board
	if err := s.loadJSON(boardKey(boardID), &board); err != nil {
		return Board{}, err
	}
	return board, nil
}

func (s *kvStore) SaveBoard(board Board) error {
	var previous Board
	prevErr := s.loadJSON(boardKey(board.ID), &previous)
	hadPrevious := prevErr == nil
	if prevErr != nil && !isNotFound(prevErr) {
		return prevErr
	}

	if err := s.saveJSON(boardKey(board.ID), board); err != nil {
		return err
	}

	allIDs, err := s.loadStringSlice(allBoardsIndexKey)
	if err != nil {
		return err
	}
	if err := s.saveJSON(allBoardsIndexKey, appendUnique(allIDs, board.ID)); err != nil {
		return err
	}

	if board.TeamID != "" {
		ids, err := s.loadStringSlice(teamBoardsKey(board.TeamID))
		if err != nil {
			return err
		}
		if err := s.saveJSON(teamBoardsKey(board.TeamID), appendUnique(ids, board.ID)); err != nil {
			return err
		}
	}

	if board.ChannelID != "" {
		ids, err := s.loadStringSlice(channelBoardsKey(board.ChannelID))
		if err != nil {
			return err
		}
		if err := s.saveJSON(channelBoardsKey(board.ChannelID), appendUnique(ids, board.ID)); err != nil {
			return err
		}
	}

	if hadPrevious {
		if previous.TeamID != "" && previous.TeamID != board.TeamID {
			ids, err := s.loadStringSlice(teamBoardsKey(previous.TeamID))
			if err != nil {
				return err
			}
			if err := s.saveJSON(teamBoardsKey(previous.TeamID), removeString(ids, board.ID)); err != nil {
				return err
			}
		}
		if previous.ChannelID != "" && previous.ChannelID != board.ChannelID {
			ids, err := s.loadStringSlice(channelBoardsKey(previous.ChannelID))
			if err != nil {
				return err
			}
			if err := s.saveJSON(channelBoardsKey(previous.ChannelID), removeString(ids, board.ID)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *kvStore) DeleteBoard(boardID string) error {
	board, err := s.GetBoard(boardID)
	if err != nil {
		return err
	}

	cardIDs, err := s.loadStringSlice(boardCardIDsKey(boardID))
	if err != nil {
		return err
	}

	for _, cardID := range cardIDs {
		if err := s.delete(dueSoonNotificationKey(boardID, cardID)); err != nil && !isNotFound(err) {
			return err
		}
		if err := s.delete(cardBoardKey(cardID)); err != nil && !isNotFound(err) {
			return err
		}
		if err := s.delete(boardCardKey(boardID, cardID)); err != nil && !isNotFound(err) {
			return err
		}
	}

	keys := []string{
		boardKey(boardID),
		boardColumnsKey(boardID),
		boardTemplatesKey(boardID),
		boardCardIDsKey(boardID),
		boardDepsKey(boardID),
		boardActivityKey(boardID),
	}
	for _, key := range keys {
		if err := s.delete(key); err != nil && !isNotFound(err) {
			return err
		}
	}

	allIDs, err := s.loadStringSlice(allBoardsIndexKey)
	if err != nil {
		return err
	}
	if err := s.saveJSON(allBoardsIndexKey, removeString(allIDs, boardID)); err != nil {
		return err
	}

	if board.TeamID != "" {
		ids, err := s.loadStringSlice(teamBoardsKey(board.TeamID))
		if err != nil {
			return err
		}
		if err := s.saveJSON(teamBoardsKey(board.TeamID), removeString(ids, boardID)); err != nil {
			return err
		}
	}

	if board.ChannelID != "" {
		ids, err := s.loadStringSlice(channelBoardsKey(board.ChannelID))
		if err != nil {
			return err
		}
		if err := s.saveJSON(channelBoardsKey(board.ChannelID), removeString(ids, boardID)); err != nil {
			return err
		}

		defaultBoard, err := s.GetDefaultBoard(board.ChannelID)
		if err == nil && defaultBoard == boardID {
			if err := s.delete(channelDefaultBoardKey(board.ChannelID)); err != nil && !isNotFound(err) {
				return err
			}
		}
	}

	return nil
}

func (s *kvStore) GetColumns(boardID string) ([]BoardColumn, error) {
	var columns []BoardColumn
	if err := s.loadJSON(boardColumnsKey(boardID), &columns); err != nil {
		if isNotFound(err) {
			return []BoardColumn{}, nil
		}
		return nil, err
	}

	sort.Slice(columns, func(i, j int) bool {
		return columns[i].SortOrder < columns[j].SortOrder
	})
	return columns, nil
}

func (s *kvStore) SaveColumns(boardID string, columns []BoardColumn) error {
	return s.saveJSON(boardColumnsKey(boardID), columns)
}

func (s *kvStore) GetTemplates(boardID string) ([]CardTemplate, error) {
	var templates []CardTemplate
	if err := s.loadJSON(boardTemplatesKey(boardID), &templates); err != nil {
		if isNotFound(err) {
			return []CardTemplate{}, nil
		}
		return nil, err
	}
	return templates, nil
}

func (s *kvStore) SaveTemplates(boardID string, templates []CardTemplate) error {
	return s.saveJSON(boardTemplatesKey(boardID), templates)
}

func (s *kvStore) ListCards(boardID string) ([]Card, error) {
	cardIDs, err := s.loadStringSlice(boardCardIDsKey(boardID))
	if err != nil {
		return nil, err
	}

	cards := make([]Card, 0, len(cardIDs))
	for _, cardID := range cardIDs {
		card, err := s.GetCard(boardID, cardID)
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return nil, err
		}
		cards = append(cards, card)
	}

	sortCards(cards)
	return cards, nil
}

func (s *kvStore) GetCard(boardID, cardID string) (Card, error) {
	var card Card
	if err := s.loadJSON(boardCardKey(boardID, cardID), &card); err != nil {
		return Card{}, err
	}
	return card, nil
}

func (s *kvStore) GetCardBoardID(cardID string) (string, error) {
	var boardID string
	if err := s.loadJSON(cardBoardKey(cardID), &boardID); err != nil {
		return "", err
	}
	return boardID, nil
}

func (s *kvStore) SaveCard(card Card) error {
	if err := s.saveJSON(boardCardKey(card.BoardID, card.ID), card); err != nil {
		return err
	}

	cardIDs, err := s.loadStringSlice(boardCardIDsKey(card.BoardID))
	if err != nil {
		return err
	}
	if err := s.saveJSON(boardCardIDsKey(card.BoardID), appendUnique(cardIDs, card.ID)); err != nil {
		return err
	}

	return s.saveJSON(cardBoardKey(card.ID), card.BoardID)
}

func (s *kvStore) DeleteCard(boardID, cardID string) error {
	cardIDs, err := s.loadStringSlice(boardCardIDsKey(boardID))
	if err != nil {
		return err
	}
	if err := s.saveJSON(boardCardIDsKey(boardID), removeString(cardIDs, cardID)); err != nil {
		return err
	}
	if err := s.delete(boardCardKey(boardID, cardID)); err != nil && !isNotFound(err) {
		return err
	}
	if err := s.delete(cardBoardKey(cardID)); err != nil && !isNotFound(err) {
		return err
	}
	return nil
}

func (s *kvStore) ListDependencies(boardID string) ([]Dependency, error) {
	var dependencies []Dependency
	if err := s.loadJSON(boardDepsKey(boardID), &dependencies); err != nil {
		if isNotFound(err) {
			return []Dependency{}, nil
		}
		return nil, err
	}
	return dependencies, nil
}

func (s *kvStore) SaveDependencies(boardID string, dependencies []Dependency) error {
	return s.saveJSON(boardDepsKey(boardID), dependencies)
}

func (s *kvStore) ListActivity(boardID string) ([]Activity, error) {
	var activity []Activity
	if err := s.loadJSON(boardActivityKey(boardID), &activity); err != nil {
		if isNotFound(err) {
			return []Activity{}, nil
		}
		return nil, err
	}
	return activity, nil
}

func (s *kvStore) AppendActivity(boardID string, activity Activity) error {
	entries, err := s.ListActivity(boardID)
	if err != nil {
		return err
	}

	entries = append([]Activity{activity}, entries...)
	if len(entries) > 200 {
		entries = entries[:200]
	}

	return s.saveJSON(boardActivityKey(boardID), entries)
}

func (s *kvStore) GetPreference(userID, boardID string) (Preference, error) {
	var preference Preference
	if err := s.loadJSON(userPreferenceKey(userID, boardID), &preference); err != nil {
		return Preference{}, err
	}
	return preference, nil
}

func (s *kvStore) SavePreference(preference Preference) error {
	return s.saveJSON(userPreferenceKey(preference.UserID, preference.BoardID), preference)
}

func (s *kvStore) GetDefaultBoard(channelID string) (string, error) {
	var boardID string
	if err := s.loadJSON(channelDefaultBoardKey(channelID), &boardID); err != nil {
		return "", err
	}
	return boardID, nil
}

func (s *kvStore) SaveDefaultBoard(channelID, boardID string) error {
	return s.saveJSON(channelDefaultBoardKey(channelID), boardID)
}

func (s *kvStore) GetDueSoonNotification(boardID, cardID string) (DueSoonNotification, error) {
	var notification DueSoonNotification
	if err := s.loadJSON(dueSoonNotificationKey(boardID, cardID), &notification); err != nil {
		return DueSoonNotification{}, err
	}
	return notification, nil
}

func (s *kvStore) SaveDueSoonNotification(notification DueSoonNotification) error {
	return s.saveJSON(dueSoonNotificationKey(notification.BoardID, notification.CardID), notification)
}

func (s *kvStore) DeleteDueSoonNotification(boardID, cardID string) error {
	return s.delete(dueSoonNotificationKey(boardID, cardID))
}

func (s *kvStore) loadStringSlice(key string) ([]string, error) {
	var values []string
	if err := s.loadJSON(key, &values); err != nil {
		if isNotFound(err) {
			return []string{}, nil
		}
		return nil, err
	}
	return values, nil
}

func (s *kvStore) loadJSON(key string, target any) error {
	raw, appErr := s.api.KVGet(key)
	if appErr != nil {
		return fmt.Errorf("kv get %s: %w", key, appErr)
	}
	if len(raw) == 0 {
		return newNotFoundError("resource not found")
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("unmarshal %s: %w", key, err)
	}
	return nil
}

func (s *kvStore) saveJSON(key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", key, err)
	}
	if appErr := s.api.KVSet(key, raw); appErr != nil {
		return fmt.Errorf("kv set %s: %w", key, appErr)
	}
	return nil
}

func (s *kvStore) delete(key string) error {
	if appErr := s.api.KVDelete(key); appErr != nil {
		return fmt.Errorf("kv delete %s: %w", key, appErr)
	}
	return nil
}

func isNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
