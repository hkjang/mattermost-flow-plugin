package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
)

var (
	ErrNotFound   = errors.New("resource not found")
	ErrConflict   = errors.New("resource conflict")
	ErrValidation = errors.New("validation failed")
	ErrForbidden  = errors.New("forbidden")
)

type ScopeQuery struct {
	TeamID    string
	ChannelID string
}

type Board struct {
	ID          string        `json:"id"`
	TeamID      string        `json:"team_id,omitempty"`
	ChannelID   string        `json:"channel_id,omitempty"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Visibility  string        `json:"visibility"`
	AdminIDs    []string      `json:"admin_ids"`
	Settings    BoardSettings `json:"settings"`
	CreatedBy   string        `json:"created_by"`
	CreatedAt   int64         `json:"created_at"`
	UpdatedAt   int64         `json:"updated_at"`
	Version     int64         `json:"version"`
}

type BoardSettings struct {
	PostUpdates   bool   `json:"post_updates"`
	PostDueSoon   bool   `json:"post_due_soon"`
	AllowMentions bool   `json:"allow_mentions"`
	DefaultView   string `json:"default_view"`
}

type BoardColumn struct {
	ID        string `json:"id"`
	BoardID   string `json:"board_id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
	WIPLimit  int    `json:"wip_limit"`
}

type ChecklistItem struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

type AttachmentLink struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

type CardComment struct {
	ID        string `json:"id"`
	CardID    string `json:"card_id"`
	UserID    string `json:"user_id"`
	Message   string `json:"message"`
	CreatedAt int64  `json:"created_at"`
}

type Card struct {
	ID              string           `json:"id"`
	BoardID         string           `json:"board_id"`
	ColumnID        string           `json:"column_id"`
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	AssigneeIDs     []string         `json:"assignee_ids"`
	Labels          []string         `json:"labels"`
	Priority        string           `json:"priority"`
	StartDate       string           `json:"start_date,omitempty"`
	DueDate         string           `json:"due_date,omitempty"`
	Progress        int              `json:"progress"`
	Milestone       bool             `json:"milestone"`
	Checklist       []ChecklistItem  `json:"checklist"`
	AttachmentLinks []AttachmentLink `json:"attachment_links"`
	Comments        []CardComment    `json:"comments"`
	Position        int              `json:"position"`
	CreatedBy       string           `json:"created_by"`
	UpdatedBy       string           `json:"updated_by"`
	CreatedAt       int64            `json:"created_at"`
	UpdatedAt       int64            `json:"updated_at"`
	Version         int64            `json:"version"`
}

type Dependency struct {
	ID           string `json:"id"`
	BoardID      string `json:"board_id"`
	SourceCardID string `json:"source_card_id"`
	TargetCardID string `json:"target_card_id"`
	Type         string `json:"type"`
	CreatedBy    string `json:"created_by"`
	CreatedAt    int64  `json:"created_at"`
}

type Activity struct {
	ID         string          `json:"id"`
	BoardID    string          `json:"board_id"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Action     string          `json:"action"`
	ActorID    string          `json:"actor_id"`
	Before     json.RawMessage `json:"before,omitempty"`
	After      json.RawMessage `json:"after,omitempty"`
	CreatedAt  int64           `json:"created_at"`
}

type Preference struct {
	UserID    string       `json:"user_id"`
	BoardID   string       `json:"board_id"`
	ViewType  string       `json:"view_type"`
	Filters   BoardFilters `json:"filters"`
	ZoomLevel string       `json:"zoom_level"`
	UpdatedAt int64        `json:"updated_at"`
}

type FlowUser struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type DueSoonNotification struct {
	BoardID    string `json:"board_id"`
	CardID     string `json:"card_id"`
	DueDate    string `json:"due_date"`
	NotifiedAt int64  `json:"notified_at"`
}

type BoardFilters struct {
	Query      string `json:"query"`
	AssigneeID string `json:"assignee_id"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	DateFrom   string `json:"date_from"`
	DateTo     string `json:"date_to"`
}

type BoardSummary struct {
	Board          Board     `json:"board"`
	CardCount      int       `json:"card_count"`
	OverdueCount   int       `json:"overdue_count"`
	DueSoonCount   int       `json:"due_soon_count"`
	DefaultBoard   bool      `json:"default_board"`
	Columns        int       `json:"columns"`
	Assignees      []string  `json:"assignees"`
	RecentActivity *Activity `json:"recent_activity,omitempty"`
}

type BoardBundle struct {
	Board        Board         `json:"board"`
	Columns      []BoardColumn `json:"columns"`
	Cards        []Card        `json:"cards"`
	Dependencies []Dependency  `json:"dependencies"`
	Activity     []Activity    `json:"activity"`
	Preference   Preference    `json:"preference"`
	Summary      BoardSummary  `json:"summary"`
}

type GanttViewData struct {
	Board        Board         `json:"board"`
	Columns      []BoardColumn `json:"columns"`
	Tasks        []Card        `json:"tasks"`
	Dependencies []Dependency  `json:"dependencies"`
}

type CardMutationResult struct {
	Board      Board  `json:"board"`
	Card       Card   `json:"card"`
	ColumnName string `json:"column_name"`
}

type CardMoveResult struct {
	Board          Board  `json:"board"`
	Card           Card   `json:"card"`
	FromColumnName string `json:"from_column_name"`
	ToColumnName   string `json:"to_column_name"`
}

type DependencyMutationResult struct {
	Board      Board      `json:"board"`
	Dependency Dependency `json:"dependency"`
}

type CommentMutationResult struct {
	Board   Board       `json:"board"`
	Card    Card        `json:"card"`
	Comment CardComment `json:"comment"`
}

type CreateBoardRequest struct {
	TeamID       string             `json:"team_id"`
	ChannelID    string             `json:"channel_id"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Visibility   string             `json:"visibility"`
	AdminIDs     []string           `json:"admin_ids"`
	Columns      []BoardColumnInput `json:"columns"`
	Settings     *BoardSettings     `json:"settings"`
	SetAsDefault bool               `json:"set_as_default"`
}

type UpdateBoardRequest struct {
	Name        *string             `json:"name"`
	Description *string             `json:"description"`
	AdminIDs    *[]string           `json:"admin_ids"`
	Columns     *[]BoardColumnInput `json:"columns"`
	Settings    *BoardSettings      `json:"settings"`
	Version     *int64              `json:"version"`
}

type BoardColumnInput struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
	WIPLimit  int    `json:"wip_limit"`
}

type CreateCardRequest struct {
	BoardID         string           `json:"board_id"`
	ColumnID        string           `json:"column_id"`
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	AssigneeIDs     []string         `json:"assignee_ids"`
	Labels          []string         `json:"labels"`
	Priority        string           `json:"priority"`
	StartDate       string           `json:"start_date"`
	DueDate         string           `json:"due_date"`
	Progress        int              `json:"progress"`
	Milestone       bool             `json:"milestone"`
	Checklist       []ChecklistItem  `json:"checklist"`
	AttachmentLinks []AttachmentLink `json:"attachment_links"`
}

type UpdateCardRequest struct {
	Title           *string           `json:"title"`
	Description     *string           `json:"description"`
	AssigneeIDs     *[]string         `json:"assignee_ids"`
	Labels          *[]string         `json:"labels"`
	Priority        *string           `json:"priority"`
	StartDate       *string           `json:"start_date"`
	DueDate         *string           `json:"due_date"`
	Progress        *int              `json:"progress"`
	Milestone       *bool             `json:"milestone"`
	Checklist       *[]ChecklistItem  `json:"checklist"`
	AttachmentLinks *[]AttachmentLink `json:"attachment_links"`
	Version         *int64            `json:"version"`
}

type MoveCardRequest struct {
	TargetColumnID string `json:"target_column_id"`
	TargetIndex    int    `json:"target_index"`
	Version        int64  `json:"version"`
}

type AddDependencyRequest struct {
	TargetCardID string `json:"target_card_id"`
	Type         string `json:"type"`
}

type AddCommentRequest struct {
	Message string `json:"message"`
}

type SavePreferenceRequest struct {
	ViewType  string       `json:"view_type"`
	Filters   BoardFilters `json:"filters"`
	ZoomLevel string       `json:"zoom_level"`
}

type CardActionResponse struct {
	Action            string              `json:"action"`
	EventAction       string              `json:"event_action"`
	Status            string              `json:"status"`
	Message           string              `json:"message"`
	BoardID           string              `json:"board_id"`
	Board             Board               `json:"board"`
	Summary           BoardSummary        `json:"summary"`
	Card              Card                `json:"card"`
	ColumnCardIDs     map[string][]string `json:"column_card_ids,omitempty"`
	CurrentColumnName string              `json:"current_column_name"`
	NextColumnName    string              `json:"next_column_name,omitempty"`
	HasNextColumn     bool                `json:"has_next_column"`
	DoneColumnName    string              `json:"done_column_name,omitempty"`
	HasDoneColumn     bool                `json:"has_done_column"`
	InDoneColumn      bool                `json:"in_done_column"`
}

type BoardStreamEvent struct {
	Type          string              `json:"type"`
	BoardID       string              `json:"board_id"`
	EntityType    string              `json:"entity_type"`
	Action        string              `json:"action"`
	ActorID       string              `json:"actor_id,omitempty"`
	CardID        string              `json:"card_id,omitempty"`
	OccurredAt    int64               `json:"occurred_at"`
	ColumnCardIDs map[string][]string `json:"column_card_ids,omitempty"`
	Board         *Board              `json:"board,omitempty"`
	Card          *Card               `json:"card,omitempty"`
	Dependency    *Dependency         `json:"dependency,omitempty"`
	Comment       *CardComment        `json:"comment,omitempty"`
	Activity      *Activity           `json:"activity,omitempty"`
}

type BoardSummaryStreamEvent struct {
	Type       string        `json:"type"`
	BoardID    string        `json:"board_id"`
	Action     string        `json:"action"`
	OccurredAt int64         `json:"occurred_at"`
	Summary    *BoardSummary `json:"summary,omitempty"`
}

func newValidationError(message string) error {
	return fmt.Errorf("%w: %s", ErrValidation, message)
}

func newConflictError(message string) error {
	return fmt.Errorf("%w: %s", ErrConflict, message)
}

func newNotFoundError(message string) error {
	return fmt.Errorf("%w: %s", ErrNotFound, message)
}

func newForbiddenError(message string) error {
	return fmt.Errorf("%w: %s", ErrForbidden, message)
}

func nowMillis() int64 {
	return model.GetMillis()
}
