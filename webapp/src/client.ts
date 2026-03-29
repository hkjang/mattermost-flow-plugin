import manifest from './manifest';

import type {
    BoardCalendarFeedInfo,
    BoardDiagnosticsReport,
    BoardBundle,
    BoardSummary,
    CardMoveResult,
    CardMutationResult,
    CommentMutationResult,
    CreateBoardRequest,
    CreateCardRequest,
    CardActionResponse,
    DependencyMutationResult,
    Dependency,
    FlowUser,
    MoveCardRequest,
    Preference,
    UpdateBoardRequest,
    UpdateCardRequest,
} from './types';

const API_ROOT = `/plugins/${manifest.id}/api/v1`;

async function request<T>(path: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${API_ROOT}${path}`, {
        credentials: 'same-origin',
        headers: {
            'Content-Type': 'application/json',
        },
        ...options,
    });

    if (!response.ok) {
        let message = `${response.status} ${response.statusText}`;
        try {
            const errorResponse = await response.json();
            message = errorResponse?.error?.message || message;
        } catch {
            // keep the default error message
        }
        throw new Error(message);
    }

    return response.json() as Promise<T>;
}

export const flowClient = {
    getBoardStreamUrl(boardId: string) {
        return `${API_ROOT}/boards/${boardId}/stream`;
    },
    getBoardSummaryStreamUrl(teamId?: string, channelId?: string) {
        const query = new URLSearchParams();
        if (teamId) {
            query.set('team_id', teamId);
        }
        if (channelId) {
            query.set('channel_id', channelId);
        }
        return `${API_ROOT}/boards/summary/stream?${query.toString()}`;
    },
    listBoards(teamId?: string, channelId?: string) {
        const query = new URLSearchParams();
        if (teamId) {
            query.set('team_id', teamId);
        }
        if (channelId) {
            query.set('channel_id', channelId);
        }
        return request<{items: BoardSummary[]}>(`/boards?${query.toString()}`);
    },
    createBoard(payload: CreateBoardRequest) {
        return request<BoardBundle>('/boards', {
            method: 'POST',
            body: JSON.stringify(payload),
        });
    },
    getBoard(boardId: string) {
        return request<BoardBundle>(`/boards/${boardId}`);
    },
    searchBoardUsers(boardId: string, options?: {term?: string; ids?: string[]; limit?: number}) {
        const query = new URLSearchParams();
        if (options?.term) {
            query.set('term', options.term);
        }
        if (options?.ids?.length) {
            query.set('ids', options.ids.join(','));
        }
        if (options?.limit) {
            query.set('limit', String(options.limit));
        }
        return request<{items: FlowUser[]}>(`/boards/${boardId}/users?${query.toString()}`);
    },
    updateBoard(boardId: string, payload: UpdateBoardRequest) {
        return request<BoardBundle>(`/boards/${boardId}`, {
            method: 'PATCH',
            body: JSON.stringify(payload),
        });
    },
    getBoardCalendarFeed(boardId: string) {
        return request<BoardCalendarFeedInfo>(`/boards/${boardId}/calendar-feed`);
    },
    rotateBoardCalendarFeed(boardId: string) {
        return request<BoardCalendarFeedInfo>(`/boards/${boardId}/calendar-feed/rotate`, {
            method: 'POST',
        });
    },
    getBoardDiagnostics(boardId: string) {
        return request<BoardDiagnosticsReport>(`/boards/${boardId}/diagnostics`);
    },
    repairBoardDiagnostics(boardId: string) {
        return request<BoardDiagnosticsReport>(`/boards/${boardId}/diagnostics/repair`, {
            method: 'POST',
        });
    },
    getBoardCalendarDownloadUrl(boardId: string) {
        return `${API_ROOT}/boards/${boardId}/calendar.ics`;
    },
    deleteBoard(boardId: string) {
        return request<{status: string}>(`/boards/${boardId}`, {
            method: 'DELETE',
        });
    },
    createCard(payload: CreateCardRequest) {
        return request<CardMutationResult>(`/cards`, {
            method: 'POST',
            body: JSON.stringify(payload),
        });
    },
    updateCard(cardId: string, payload: UpdateCardRequest) {
        return request<CardMutationResult>(`/cards/${cardId}`, {
            method: 'PATCH',
            body: JSON.stringify(payload),
        });
    },
    moveCard(cardId: string, payload: MoveCardRequest) {
        return request<CardMoveResult>(`/cards/${cardId}/move`, {
            method: 'POST',
            body: JSON.stringify(payload),
        });
    },
    runCardAction(cardId: string, action: 'assign-self' | 'move-next' | 'push-1d' | 'push-7d' | 'complete-next-checklist' | 'complete-card') {
        return request<CardActionResponse>(`/cards/${cardId}/actions/${action}`, {
            method: 'POST',
        });
    },
    addDependency(cardId: string, targetCardId: string, type = 'finish_to_start') {
        return request<DependencyMutationResult>(`/cards/${cardId}/dependencies`, {
            method: 'POST',
            body: JSON.stringify({target_card_id: targetCardId, type}),
        });
    },
    addComment(cardId: string, message: string) {
        return request<CommentMutationResult>(`/cards/${cardId}/comments`, {
            method: 'POST',
            body: JSON.stringify({message}),
        });
    },
    savePreference(boardId: string, payload: Pick<Preference, 'view_type' | 'filters' | 'zoom_level'>) {
        return request<Preference>(`/boards/${boardId}/preferences`, {
            method: 'PUT',
            body: JSON.stringify(payload),
        });
    },
};
