package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

func (p *Plugin) initRouter() *mux.Router {
	router := mux.NewRouter()
	router.Use(p.MattermostAuthorizationRequired)

	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.HandleFunc("/ping", p.handlePing).Methods(http.MethodGet)
	apiRouter.HandleFunc("/boards", p.handleListBoards).Methods(http.MethodGet)
	apiRouter.HandleFunc("/boards/summary/stream", p.handleBoardSummaryStream).Methods(http.MethodGet)
	apiRouter.HandleFunc("/boards", p.handleCreateBoard).Methods(http.MethodPost)
	apiRouter.HandleFunc("/boards/{id}", p.handleGetBoard).Methods(http.MethodGet)
	apiRouter.HandleFunc("/boards/{id}", p.handleUpdateBoard).Methods(http.MethodPatch)
	apiRouter.HandleFunc("/boards/{id}", p.handleDeleteBoard).Methods(http.MethodDelete)
	apiRouter.HandleFunc("/boards/{id}/stream", p.handleBoardStream).Methods(http.MethodGet)
	apiRouter.HandleFunc("/boards/{id}/cards", p.handleListCards).Methods(http.MethodGet)
	apiRouter.HandleFunc("/boards/{id}/gantt", p.handleGetGantt).Methods(http.MethodGet)
	apiRouter.HandleFunc("/boards/{id}/activity", p.handleListActivity).Methods(http.MethodGet)
	apiRouter.HandleFunc("/boards/{id}/preferences", p.handleSavePreference).Methods(http.MethodPut)
	apiRouter.HandleFunc("/boards/{id}/users", p.handleListBoardUsers).Methods(http.MethodGet)
	apiRouter.HandleFunc("/cards", p.handleCreateCard).Methods(http.MethodPost)
	apiRouter.HandleFunc("/cards/{id}", p.handleUpdateCard).Methods(http.MethodPatch)
	apiRouter.HandleFunc("/cards/{id}/move", p.handleMoveCard).Methods(http.MethodPost)
	apiRouter.HandleFunc("/cards/{id}/actions/{action}", p.handleCardAction).Methods(http.MethodPost)
	apiRouter.HandleFunc("/cards/{id}/dependencies", p.handleAddDependency).Methods(http.MethodPost)
	apiRouter.HandleFunc("/cards/{id}/comments", p.handleAddComment).Methods(http.MethodPost)

	return router
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.router.ServeHTTP(w, r)
}

func (p *Plugin) MattermostAuthorizationRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p.service == nil {
			http.Error(w, "Plugin not initialized", http.StatusServiceUnavailable)
			return
		}

		if r.Header.Get("Mattermost-User-ID") == "" {
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (p *Plugin) handlePing(w http.ResponseWriter, r *http.Request) {
	p.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (p *Plugin) handleListBoards(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	teamID := strings.TrimSpace(r.URL.Query().Get("team_id"))
	channelID := strings.TrimSpace(r.URL.Query().Get("channel_id"))

	if teamID == "" && channelID == "" {
		p.writeError(w, newValidationError("team_id or channel_id is required"))
		return
	}

	if err := p.authorizeScope(userID, teamID, channelID); err != nil {
		p.writeError(w, err)
		return
	}

	boards, err := p.service.ListBoards(ScopeQuery{TeamID: teamID, ChannelID: channelID})
	if err != nil {
		p.writeError(w, err)
		return
	}

	p.writeJSON(w, http.StatusOK, map[string]any{"items": boards})
}

func (p *Plugin) handleCreateBoard(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)

	var req CreateBoardRequest
	if err := decodeJSON(r, &req); err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeScopeAdmin(userID, req.TeamID, req.ChannelID); err != nil {
		p.writeError(w, err)
		return
	}

	board, err := p.service.CreateBoard(userID, req)
	if err != nil {
		p.writeError(w, err)
		return
	}

	p.publishBoardEvent(BoardStreamEvent{
		BoardID:    board.Board.ID,
		EntityType: "board",
		Action:     "board.created",
		ActorID:    userID,
		Board:      &board.Board,
	})
	p.writeJSON(w, http.StatusCreated, board)
}

func (p *Plugin) handleGetBoard(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	boardID := mux.Vars(r)["id"]

	board, err := p.service.GetBoardBundle(boardID, userID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeBoard(userID, board.Board, false); err != nil {
		p.writeError(w, err)
		return
	}

	p.writeJSON(w, http.StatusOK, board)
}

func (p *Plugin) handleUpdateBoard(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	boardID := mux.Vars(r)["id"]

	existing, err := p.service.GetBoardBundle(boardID, userID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeBoard(userID, existing.Board, true); err != nil {
		p.writeError(w, err)
		return
	}

	var req UpdateBoardRequest
	if err := decodeJSON(r, &req); err != nil {
		p.writeError(w, err)
		return
	}

	board, err := p.service.UpdateBoard(userID, boardID, req)
	if err != nil {
		p.writeError(w, err)
		return
	}

	p.publishBoardEvent(BoardStreamEvent{
		BoardID:    board.Board.ID,
		EntityType: "board",
		Action:     "board.updated",
		ActorID:    userID,
		Board:      &board.Board,
	})
	p.writeJSON(w, http.StatusOK, board)
}

func (p *Plugin) handleDeleteBoard(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	boardID := mux.Vars(r)["id"]

	board, err := p.service.GetBoard(boardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeBoard(userID, board, true); err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.service.DeleteBoard(userID, boardID); err != nil {
		p.writeError(w, err)
		return
	}

	p.publishBoardEvent(BoardStreamEvent{
		BoardID:    boardID,
		EntityType: "board",
		Action:     "board.deleted",
		ActorID:    userID,
		Board:      &board,
	})
	p.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (p *Plugin) handleBoardSummaryStream(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	teamID := strings.TrimSpace(r.URL.Query().Get("team_id"))
	channelID := strings.TrimSpace(r.URL.Query().Get("channel_id"))

	if teamID == "" && channelID == "" {
		p.writeError(w, newValidationError("team_id or channel_id is required"))
		return
	}

	if err := p.authorizeScope(userID, teamID, channelID); err != nil {
		p.writeError(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		p.writeError(w, fmt.Errorf("streaming is not supported"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	scopeKeys := uniqueStrings([]string{
		scopeKeyForChannel(channelID),
		scopeKeyForTeam(teamID),
	})
	events, cancel := p.eventBroker.SubscribeSummary(scopeKeys)
	defer cancel()

	readyPayload, _ := json.Marshal(BoardSummaryStreamEvent{
		Type:       "board_summary_ready",
		Action:     "board.summary.ready",
		OccurredAt: nowMillis(),
	})
	fmt.Fprintf(w, "event: summary_ready\ndata: %s\n\n", readyPayload)
	flusher.Flush()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			payload, marshalErr := json.Marshal(event)
			if marshalErr != nil {
				continue
			}
			if _, writeErr := fmt.Fprintf(w, "event: board_summary_event\ndata: %s\n\n", payload); writeErr != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, writeErr := fmt.Fprint(w, ": keep-alive\n\n"); writeErr != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (p *Plugin) handleBoardStream(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	boardID := mux.Vars(r)["id"]

	board, err := p.service.GetBoard(boardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeBoard(userID, board, false); err != nil {
		p.writeError(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		p.writeError(w, fmt.Errorf("streaming is not supported"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	events, cancel := p.eventBroker.Subscribe(boardID)
	defer cancel()

	readyPayload, _ := json.Marshal(BoardStreamEvent{
		Type:       "board_ready",
		BoardID:    boardID,
		EntityType: "board",
		Action:     "board.ready",
		OccurredAt: nowMillis(),
	})
	fmt.Fprintf(w, "event: ready\ndata: %s\n\n", readyPayload)
	flusher.Flush()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			payload, marshalErr := json.Marshal(event)
			if marshalErr != nil {
				continue
			}
			if _, writeErr := fmt.Fprintf(w, "event: board_event\ndata: %s\n\n", payload); writeErr != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, writeErr := fmt.Fprint(w, ": keep-alive\n\n"); writeErr != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (p *Plugin) handleListCards(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	boardID := mux.Vars(r)["id"]

	board, err := p.service.GetBoard(boardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeBoard(userID, board, false); err != nil {
		p.writeError(w, err)
		return
	}

	cards, err := p.service.ListCards(boardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	p.writeJSON(w, http.StatusOK, map[string]any{"items": cards})
}

func (p *Plugin) handleGetGantt(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	boardID := mux.Vars(r)["id"]

	board, err := p.service.GetBoard(boardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeBoard(userID, board, false); err != nil {
		p.writeError(w, err)
		return
	}

	ganttData, err := p.service.GetGantt(boardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	p.writeJSON(w, http.StatusOK, ganttData)
}

func (p *Plugin) handleListActivity(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	boardID := mux.Vars(r)["id"]

	board, err := p.service.GetBoard(boardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeBoard(userID, board, false); err != nil {
		p.writeError(w, err)
		return
	}

	activity, err := p.service.ListActivity(boardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	p.writeJSON(w, http.StatusOK, map[string]any{"items": activity})
}

func (p *Plugin) handleSavePreference(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	boardID := mux.Vars(r)["id"]

	board, err := p.service.GetBoard(boardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeBoard(userID, board, false); err != nil {
		p.writeError(w, err)
		return
	}

	var req SavePreferenceRequest
	if err := decodeJSON(r, &req); err != nil {
		p.writeError(w, err)
		return
	}

	preference, err := p.service.SavePreference(userID, boardID, req)
	if err != nil {
		p.writeError(w, err)
		return
	}

	p.writeJSON(w, http.StatusOK, preference)
}

func (p *Plugin) handleListBoardUsers(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	boardID := mux.Vars(r)["id"]

	board, err := p.service.GetBoard(boardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeBoard(userID, board, false); err != nil {
		p.writeError(w, err)
		return
	}

	term := strings.TrimSpace(r.URL.Query().Get("term"))
	ids := splitCSV(strings.TrimSpace(r.URL.Query().Get("ids")))
	limit := parseLimit(r.URL.Query().Get("limit"), 20, 100)

	users, err := p.listBoardUsers(board, term, ids, limit)
	if err != nil {
		p.writeError(w, err)
		return
	}

	p.writeJSON(w, http.StatusOK, map[string]any{"items": users})
}

func (p *Plugin) handleCreateCard(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)

	var req CreateCardRequest
	if err := decodeJSON(r, &req); err != nil {
		p.writeError(w, err)
		return
	}

	board, err := p.service.GetBoard(req.BoardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeBoard(userID, board, false); err != nil {
		p.writeError(w, err)
		return
	}

	req.AssigneeIDs = p.normalizeAssignees(req.AssigneeIDs)

	result, err := p.service.CreateCard(userID, req)
	if err != nil {
		p.writeError(w, err)
		return
	}

	p.postCardUpdate(result.Board, result.Card, userID, fmt.Sprintf("created card **%s** in **%s**", result.Card.Title, result.ColumnName))
	p.publishBoardEvent(BoardStreamEvent{
		BoardID:    result.Board.ID,
		EntityType: "card",
		Action:     "card.created",
		ActorID:    userID,
		CardID:     result.Card.ID,
		Board:      &result.Board,
		Card:       &result.Card,
	})
	p.writeJSON(w, http.StatusCreated, result)
}

func (p *Plugin) handleUpdateCard(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	cardID := mux.Vars(r)["id"]

	card, board, err := p.service.GetCard(cardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeBoard(userID, board, false); err != nil {
		p.writeError(w, err)
		return
	}

	var req UpdateCardRequest
	if err := decodeJSON(r, &req); err != nil {
		p.writeError(w, err)
		return
	}

	if req.AssigneeIDs != nil {
		normalized := p.normalizeAssignees(*req.AssigneeIDs)
		req.AssigneeIDs = &normalized
	}

	result, err := p.service.UpdateCard(userID, card.ID, req)
	if err != nil {
		p.writeError(w, err)
		return
	}

	p.postCardUpdate(result.Board, result.Card, userID, fmt.Sprintf("updated card **%s**", result.Card.Title))
	p.publishBoardEvent(BoardStreamEvent{
		BoardID:    result.Board.ID,
		EntityType: "card",
		Action:     "card.updated",
		ActorID:    userID,
		CardID:     result.Card.ID,
		Board:      &result.Board,
		Card:       &result.Card,
	})
	p.writeJSON(w, http.StatusOK, result)
}

func (p *Plugin) handleMoveCard(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	cardID := mux.Vars(r)["id"]

	_, board, err := p.service.GetCard(cardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeBoard(userID, board, false); err != nil {
		p.writeError(w, err)
		return
	}

	var req MoveCardRequest
	if err := decodeJSON(r, &req); err != nil {
		p.writeError(w, err)
		return
	}

	result, err := p.service.MoveCard(userID, cardID, req)
	if err != nil {
		p.writeError(w, err)
		return
	}

	p.postCardUpdate(result.Board, result.Card, userID, fmt.Sprintf("moved card **%s** from **%s** to **%s**", result.Card.Title, result.FromColumnName, result.ToColumnName))
	p.publishBoardEvent(BoardStreamEvent{
		BoardID:    result.Board.ID,
		EntityType: "card",
		Action:     "card.moved",
		ActorID:    userID,
		CardID:     result.Card.ID,
		Board:      &result.Board,
		Card:       &result.Card,
	})
	p.writeJSON(w, http.StatusOK, result)
}

func (p *Plugin) handleCardAction(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	cardID := mux.Vars(r)["id"]
	action := strings.TrimSpace(mux.Vars(r)["action"])

	card, board, err := p.service.GetCard(cardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeBoard(userID, board, false); err != nil {
		p.writeError(w, err)
		return
	}

	switch action {
	case "assign-self":
		result, changed, actionErr := p.service.AssignCardToUser(userID, cardID, userID)
		if actionErr != nil {
			p.writeError(w, actionErr)
			return
		}
		if changed {
			p.postCardUpdate(result.Board, result.Card, userID, fmt.Sprintf("assigned themselves to **%s**", result.Card.Title))
			p.publishBoardEvent(BoardStreamEvent{
				BoardID:    result.Board.ID,
				EntityType: "card",
				Action:     "card.assignee_added",
				ActorID:    userID,
				CardID:     result.Card.ID,
				Board:      &result.Board,
				Card:       &result.Card,
			})
		}

		status := "noop"
		message := fmt.Sprintf("You are already assigned to %s.", result.Card.Title)
		if changed {
			status = "applied"
			message = fmt.Sprintf("Assigned you to %s.", result.Card.Title)
		}

		p.writeJSON(w, http.StatusOK, p.newCardActionResponse(action, "card.assignee_added", status, message, result.Board, result.Card))
	case "move-next":
		result, actionErr := p.service.MoveCardToNextColumn(userID, cardID)
		if actionErr != nil {
			p.writeError(w, actionErr)
			return
		}

		p.postCardUpdate(result.Board, result.Card, userID, fmt.Sprintf("moved card **%s** from **%s** to **%s**", result.Card.Title, result.FromColumnName, result.ToColumnName))
		p.publishBoardEvent(BoardStreamEvent{
			BoardID:    result.Board.ID,
			EntityType: "card",
			Action:     "card.moved",
			ActorID:    userID,
			CardID:     result.Card.ID,
			Board:      &result.Board,
			Card:       &result.Card,
		})
		p.writeJSON(w, http.StatusOK, p.newCardActionResponse(action, "card.moved", "applied", fmt.Sprintf("Moved %s to %s.", result.Card.Title, result.ToColumnName), result.Board, result.Card))
	case "push-1d", "push-7d":
		days := 1
		if action == "push-7d" {
			days = 7
		}

		nextDueDate := nextCardDueDate(card.DueDate, days)
		result, changed, actionErr := p.service.SetCardDueDate(userID, cardID, nextDueDate)
		if actionErr != nil {
			p.writeError(w, actionErr)
			return
		}
		if changed {
			p.postCardUpdate(result.Board, result.Card, userID, fmt.Sprintf("updated due date for **%s** to **%s**", result.Card.Title, result.Card.DueDate))
			p.publishBoardEvent(BoardStreamEvent{
				BoardID:    result.Board.ID,
				EntityType: "card",
				Action:     "card.due_date_updated",
				ActorID:    userID,
				CardID:     result.Card.ID,
				Board:      &result.Board,
				Card:       &result.Card,
			})
		}

		status := "noop"
		message := fmt.Sprintf("Due date is already %s.", result.Card.DueDate)
		if changed {
			status = "applied"
			message = fmt.Sprintf("Due date moved to %s.", result.Card.DueDate)
		}

		p.writeJSON(w, http.StatusOK, p.newCardActionResponse(action, "card.due_date_updated", status, message, result.Board, result.Card))
	case "complete-next-checklist":
		result, itemText, changed, actionErr := p.service.CompleteNextChecklistItem(userID, cardID)
		if actionErr != nil {
			p.writeError(w, actionErr)
			return
		}
		if changed {
			p.postCardUpdate(result.Board, result.Card, userID, fmt.Sprintf("completed checklist item **%s** on **%s**", itemText, result.Card.Title))
			p.publishBoardEvent(BoardStreamEvent{
				BoardID:    result.Board.ID,
				EntityType: "card",
				Action:     "card.checklist_item_completed",
				ActorID:    userID,
				CardID:     result.Card.ID,
				Board:      &result.Board,
				Card:       &result.Card,
			})
		}

		status := "noop"
		message := fmt.Sprintf("No open checklist items left on %s.", result.Card.Title)
		if changed {
			status = "applied"
			message = fmt.Sprintf("Completed checklist item %q.", itemText)
		}

		p.writeJSON(w, http.StatusOK, p.newCardActionResponse(action, "card.checklist_item_completed", status, message, result.Board, result.Card))
	case "complete-card":
		result, changed, actionErr := p.service.CompleteCard(userID, cardID)
		if actionErr != nil {
			p.writeError(w, actionErr)
			return
		}
		if changed {
			p.postCardUpdate(result.Board, result.Card, userID, fmt.Sprintf("marked **%s** done", result.Card.Title))
			p.publishBoardEvent(BoardStreamEvent{
				BoardID:    result.Board.ID,
				EntityType: "card",
				Action:     "card.completed",
				ActorID:    userID,
				CardID:     result.Card.ID,
				Board:      &result.Board,
				Card:       &result.Card,
			})
		}

		status := "noop"
		message := fmt.Sprintf("%s is already done.", result.Card.Title)
		if changed {
			status = "applied"
			message = fmt.Sprintf("Marked %s done.", result.Card.Title)
		}

		p.writeJSON(w, http.StatusOK, p.newCardActionResponse(action, "card.completed", status, message, result.Board, result.Card))
	default:
		p.writeError(w, newValidationError("unknown card action"))
	}
}

func (p *Plugin) newCardActionResponse(action, eventAction, status, message string, board Board, card Card) CardActionResponse {
	details := p.describeCardColumns(board.ID, card.ColumnID)
	var columnCardIDs map[string][]string
	summary := BoardSummary{Board: board}
	if status == "applied" && needsColumnCardSnapshot(eventAction) && p.service != nil {
		snapshot, err := p.service.BuildColumnCardIDs(board.ID)
		if err == nil {
			columnCardIDs = snapshot
		}
	}
	if p.service != nil {
		nextSummary, err := p.service.GetBoardSummary(board.ID)
		if err == nil {
			summary = nextSummary
		}
	}

	return CardActionResponse{
		Action:            action,
		EventAction:       eventAction,
		Status:            status,
		Message:           message,
		BoardID:           board.ID,
		Board:             board,
		Summary:           summary,
		Card:              card,
		ColumnCardIDs:     columnCardIDs,
		CurrentColumnName: details.currentName,
		NextColumnName:    details.nextName,
		HasNextColumn:     details.hasNext,
		DoneColumnName:    details.doneName,
		HasDoneColumn:     details.hasDone,
		InDoneColumn:      details.inDone,
	}
}

func nextCardDueDate(currentDueDate string, days int) string {
	baseDate, ok := parseDay(currentDueDate)
	if !ok {
		baseDate = startOfDay(time.Now().UTC())
	}

	return baseDate.AddDate(0, 0, days).Format("2006-01-02")
}

func (p *Plugin) handleAddDependency(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	cardID := mux.Vars(r)["id"]

	_, board, err := p.service.GetCard(cardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeBoard(userID, board, false); err != nil {
		p.writeError(w, err)
		return
	}

	var req AddDependencyRequest
	if err := decodeJSON(r, &req); err != nil {
		p.writeError(w, err)
		return
	}

	result, err := p.service.AddDependency(userID, cardID, req)
	if err != nil {
		p.writeError(w, err)
		return
	}

	p.publishBoardEvent(BoardStreamEvent{
		BoardID:    result.Board.ID,
		EntityType: "dependency",
		Action:     "dependency.created",
		ActorID:    userID,
		CardID:     cardID,
		Board:      &result.Board,
		Dependency: &result.Dependency,
	})
	p.writeJSON(w, http.StatusCreated, result)
}

func (p *Plugin) handleAddComment(w http.ResponseWriter, r *http.Request) {
	userID := p.getUserID(r)
	cardID := mux.Vars(r)["id"]

	_, board, err := p.service.GetCard(cardID)
	if err != nil {
		p.writeError(w, err)
		return
	}

	if err := p.authorizeBoard(userID, board, false); err != nil {
		p.writeError(w, err)
		return
	}

	var req AddCommentRequest
	if err := decodeJSON(r, &req); err != nil {
		p.writeError(w, err)
		return
	}

	result, err := p.service.AddComment(userID, cardID, req)
	if err != nil {
		p.writeError(w, err)
		return
	}

	p.publishBoardEvent(BoardStreamEvent{
		BoardID:    result.Board.ID,
		EntityType: "comment",
		Action:     "comment.created",
		ActorID:    userID,
		CardID:     result.Card.ID,
		Board:      &result.Board,
		Card:       &result.Card,
		Comment:    &result.Comment,
	})
	p.writeJSON(w, http.StatusCreated, result)
}

func (p *Plugin) authorizeScope(userID, teamID, channelID string) error {
	if channelID != "" {
		if _, appErr := p.API.GetChannelMember(channelID, userID); appErr != nil {
			return newForbiddenError("user cannot access channel")
		}
	}

	if teamID != "" {
		if _, appErr := p.API.GetTeamMember(teamID, userID); appErr != nil {
			return newForbiddenError("user cannot access team")
		}
	}

	return nil
}

func (p *Plugin) authorizeScopeAdmin(userID, teamID, channelID string) error {
	if err := p.authorizeScope(userID, teamID, channelID); err != nil {
		return err
	}

	if p.isSystemAdmin(userID) {
		return nil
	}

	if teamID != "" && p.isTeamAdmin(userID, teamID) {
		return nil
	}

	if teamID == "" && channelID != "" {
		channel, appErr := p.API.GetChannel(channelID)
		if appErr == nil && channel != nil && channel.TeamId != "" && p.isTeamAdmin(userID, channel.TeamId) {
			return nil
		}
	}

	return newForbiddenError("admin permission is required")
}

func (p *Plugin) authorizeBoard(userID string, board Board, adminOnly bool) error {
	if err := p.authorizeScope(userID, board.TeamID, board.ChannelID); err != nil {
		return err
	}

	if !adminOnly {
		return nil
	}

	if p.isSystemAdmin(userID) || containsString(board.AdminIDs, userID) {
		return nil
	}

	if board.TeamID != "" && p.isTeamAdmin(userID, board.TeamID) {
		return nil
	}

	if board.TeamID == "" && board.ChannelID != "" {
		channel, appErr := p.API.GetChannel(board.ChannelID)
		if appErr == nil && channel != nil && channel.TeamId != "" && p.isTeamAdmin(userID, channel.TeamId) {
			return nil
		}
	}

	return newForbiddenError("board admin permission is required")
}

func (p *Plugin) isSystemAdmin(userID string) bool {
	user, appErr := p.API.GetUser(userID)
	if appErr != nil || user == nil {
		return false
	}

	return strings.Contains(user.Roles, "system_admin")
}

func (p *Plugin) isTeamAdmin(userID, teamID string) bool {
	if teamID == "" {
		return false
	}

	member, appErr := p.API.GetTeamMember(teamID, userID)
	if appErr != nil || member == nil {
		return false
	}

	return strings.Contains(member.Roles, "team_admin")
}

func (p *Plugin) getUserID(r *http.Request) string {
	return r.Header.Get("Mattermost-User-ID")
}

func (p *Plugin) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		p.API.LogError("failed to write json response", "error", err.Error())
	}
}

func (p *Plugin) writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal error"

	switch {
	case errors.Is(err, ErrValidation):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
		message = err.Error()
	case errors.Is(err, ErrConflict):
		status = http.StatusConflict
		message = err.Error()
	case errors.Is(err, ErrForbidden):
		status = http.StatusForbidden
		message = err.Error()
	default:
		if err != nil {
			message = err.Error()
		}
	}

	p.writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"message": message,
			"status":  status,
		},
	})
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return newValidationError(fmt.Sprintf("invalid request body: %s", err.Error()))
	}

	return nil
}

func (p *Plugin) normalizeAssignees(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		candidate := strings.TrimSpace(strings.TrimPrefix(value, "@"))
		if candidate == "" {
			continue
		}

		if user, appErr := p.API.GetUser(candidate); appErr == nil && user != nil {
			normalized = appendUnique(normalized, user.Id)
			continue
		}

		if user, appErr := p.API.GetUserByUsername(candidate); appErr == nil && user != nil {
			normalized = appendUnique(normalized, user.Id)
			continue
		}

		normalized = appendUnique(normalized, candidate)
	}

	return normalized
}

func (p *Plugin) listBoardUsers(board Board, term string, ids []string, limit int) ([]FlowUser, error) {
	if len(ids) > 0 {
		users, err := p.client.User.ListByUserIDs(ids)
		if err != nil {
			return nil, err
		}

		usersByID := make(map[string]*model.User, len(users))
		for _, user := range users {
			if user == nil || !p.isUserInBoardScope(user.Id, board) {
				continue
			}
			usersByID[user.Id] = user
		}

		items := make([]FlowUser, 0, len(ids))
		for _, id := range ids {
			if user, ok := usersByID[id]; ok {
				items = append(items, newFlowUser(user))
			}
		}

		return items, nil
	}

	if limit <= 0 {
		limit = 20
	}

	var (
		users []*model.User
		err   error
	)

	switch {
	case board.ChannelID != "":
		if term == "" {
			users, err = p.client.User.ListInChannel(board.ChannelID, "username", 0, limit)
		} else {
			users, err = p.client.User.Search(&model.UserSearch{
				Term:        term,
				InChannelId: board.ChannelID,
				Limit:       limit,
			})
		}
	case board.TeamID != "":
		if term == "" {
			users, err = p.client.User.ListInTeam(board.TeamID, 0, limit)
		} else {
			users, err = p.client.User.Search(&model.UserSearch{
				Term:   term,
				TeamId: board.TeamID,
				Limit:  limit,
			})
		}
	default:
		if term == "" {
			return []FlowUser{}, nil
		}
		users, err = p.client.User.Search(&model.UserSearch{
			Term:  term,
			Limit: limit,
		})
	}

	if err != nil {
		return nil, err
	}

	items := make([]FlowUser, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		items = append(items, newFlowUser(user))
	}

	return items, nil
}

func (p *Plugin) isUserInBoardScope(userID string, board Board) bool {
	if board.ChannelID != "" {
		_, appErr := p.API.GetChannelMember(board.ChannelID, userID)
		return appErr == nil
	}

	if board.TeamID != "" {
		_, appErr := p.API.GetTeamMember(board.TeamID, userID)
		return appErr == nil
	}

	return true
}

func newFlowUser(user *model.User) FlowUser {
	displayName := user.GetDisplayName(model.ShowFullName)
	if displayName == "" {
		displayName = user.Username
	}

	return FlowUser{
		ID:          user.Id,
		Username:    user.Username,
		DisplayName: displayName,
	}
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		candidate := strings.TrimSpace(part)
		if candidate == "" {
			continue
		}
		items = append(items, candidate)
	}

	return items
}

func parseLimit(value string, fallback, max int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > max {
		return max
	}
	return parsed
}

func (p *Plugin) postCardUpdate(board Board, card Card, actorID, action string) {
	if board.ChannelID == "" || !board.Settings.PostUpdates {
		return
	}

	actorName := "Someone"
	if user, appErr := p.API.GetUser(actorID); appErr == nil && user != nil {
		actorName = user.GetDisplayName(model.ShowFullName)
		if actorName == "" {
			actorName = user.Username
		}
	}

	message := fmt.Sprintf("[Flow] %s %s", actorName, action)
	if board.Settings.AllowMentions && len(card.AssigneeIDs) > 0 {
		mentions := make([]string, 0, len(card.AssigneeIDs))
		for _, assigneeID := range card.AssigneeIDs {
			mentions = append(mentions, fmt.Sprintf("<@%s>", assigneeID))
		}
		message += "\n" + strings.Join(mentions, " ")
	}

	props := p.buildFlowCardPostProps(board, card)
	cardLinkURL, _ := props["card_link_url"].(string)
	if cardLinkURL != "" {
		message += fmt.Sprintf("\n[Open card](%s)", cardLinkURL)
	}

	props["flow_type"] = "update"
	props["actor_name"] = actorName
	props["summary"] = fmt.Sprintf("%s %s", actorName, strings.ReplaceAll(action, "**", ""))
	props["link_url"] = cardLinkURL

	post := &model.Post{
		UserId:    actorID,
		ChannelId: board.ChannelID,
		Message:   message,
		Type:      FlowPostTypeUpdate,
		Props:     props,
	}

	if _, appErr := p.API.CreatePost(post); appErr != nil {
		p.API.LogError("failed to create flow post", "error", appErr.Error())
	}
}
