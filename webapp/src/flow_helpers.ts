import type {
    AttachmentLink,
    BoardBundle,
    BoardColumn,
    BoardColumnInput,
    BoardFilters,
    Card,
    ChecklistItem,
} from './types';

export type BoardSettingsDraft = {
    name: string;
    description: string;
    postUpdates: boolean;
    postDueSoon: boolean;
    allowMentions: boolean;
    defaultView: 'board' | 'gantt';
    columnsText: string;
    version: number;
};

export type CardEditorDraft = {
    title: string;
    description: string;
    assigneeIds: string[];
    labelsText: string;
    priority: Card['priority'];
    startDate: string;
    dueDate: string;
    progress: number;
    milestone: boolean;
    checklistText: string;
    linksText: string;
    version: number;
};

export type FlowUrlState = {
    boardId: string;
    channelId?: string;
    cardId?: string;
    view?: 'board' | 'gantt';
};

export const EMPTY_FILTERS: BoardFilters = {
    query: '',
    assignee_id: '',
    label: '',
    status: '',
    date_from: '',
    date_to: '',
};

export const DEFAULT_BOARD_SETTINGS = {
    post_updates: true,
    post_due_soon: false,
    allow_mentions: true,
    default_view: 'board' as const,
};

export const DEFAULT_COLUMNS = ['Todo|0', 'In Progress|0', 'Review|0', 'Done|0'];

const PRIORITY_ORDER: Record<Card['priority'], number> = {
    urgent: 0,
    high: 1,
    normal: 2,
    low: 3,
};

export function createBoardSettingsDraft(bundle: BoardBundle): BoardSettingsDraft {
    return {
        name: bundle.board.name,
        description: bundle.board.description,
        postUpdates: bundle.board.settings.post_updates,
        postDueSoon: bundle.board.settings.post_due_soon,
        allowMentions: bundle.board.settings.allow_mentions,
        defaultView: bundle.board.settings.default_view,
        columnsText: bundle.columns.map((column) => `${column.name}|${column.wip_limit || 0}`).join('\n'),
        version: bundle.board.version,
    };
}

export function createCardDraft(card: Card): CardEditorDraft {
    return {
        title: card.title,
        description: card.description,
        assigneeIds: card.assignee_ids,
        labelsText: card.labels.join(', '),
        priority: card.priority,
        startDate: card.start_date || '',
        dueDate: card.due_date || '',
        progress: card.progress,
        milestone: card.milestone,
        checklistText: card.checklist.map((item) => `${item.completed ? '[x]' : '[ ]'} ${item.text}`).join('\n'),
        linksText: card.attachment_links.map((link) => `${link.title || 'Link'}|${link.url}`).join('\n'),
        version: card.version,
    };
}

export function createColumnInputs(columnsText: string, existingColumns: BoardColumn[] = []): BoardColumnInput[] {
    return columnsText.split('\n').map((line, index) => {
        const [namePart, wipPart] = line.split('|');
        return {
            id: existingColumns[index]?.id,
            name: (namePart || '').trim(),
            sort_order: index,
            wip_limit: Number((wipPart || '0').trim()) || 0,
        };
    }).filter((column) => column.name);
}

export function splitCommaValues(value: string) {
    return value.split(',').map((item) => item.trim()).filter(Boolean);
}

export function parseChecklist(text: string, existingItems: ChecklistItem[]) {
    const lines = text.split('\n').map((line) => line.trim()).filter(Boolean);
    return lines.map((line, index) => {
        const completed = line.startsWith('[x]');
        const cleanText = line.replace(/^\[(x| )\]\s*/i, '').trim();
        return {
            id: existingItems[index]?.id || `draft-${index}`,
            text: cleanText,
            completed,
        };
    });
}

export function parseLinks(text: string, existingLinks: AttachmentLink[]) {
    const lines = text.split('\n').map((line) => line.trim()).filter(Boolean);
    return lines.map((line, index) => {
        const [title, url] = line.split('|');
        return {
            id: existingLinks[index]?.id || `link-${index}`,
            title: (title || 'Link').trim(),
            url: (url || title || '').trim(),
        };
    }).filter((item) => item.url);
}

export function selectBoardId(boardIds: string[], requestedBoardId: string, currentBoardId: string, defaultBoardId: string) {
    if (requestedBoardId && boardIds.includes(requestedBoardId)) {
        return requestedBoardId;
    }
    if (currentBoardId && boardIds.includes(currentBoardId)) {
        return currentBoardId;
    }
    return defaultBoardId || boardIds[0] || '';
}

export function syncUrl(state: FlowUrlState) {
    const search = new URLSearchParams(window.location.search);
    if (state.boardId) {
        search.set('board_id', state.boardId);
    } else {
        search.delete('board_id');
    }
    if (state.channelId) {
        search.set('channel_id', state.channelId);
    } else {
        search.delete('channel_id');
    }
    if (state.cardId) {
        search.set('card_id', state.cardId);
    } else {
        search.delete('card_id');
    }
    if (state.view) {
        search.set('view', state.view);
    } else {
        search.delete('view');
    }
    window.history.replaceState({}, '', `${window.location.pathname}?${search.toString()}`);
}

export function buildFlowUrl(state: FlowUrlState) {
    const search = new URLSearchParams();
    if (state.boardId) {
        search.set('board_id', state.boardId);
    }
    if (state.channelId) {
        search.set('channel_id', state.channelId);
    }
    if (state.cardId) {
        search.set('card_id', state.cardId);
    }
    if (state.view) {
        search.set('view', state.view);
    }

    const path = `${window.location.pathname}?${search.toString()}`;
    return new URL(path, window.location.origin).toString();
}

export function getVisibleCards(cards: Card[], columns: BoardColumn[], filters: BoardFilters, sortMode: 'manual' | 'priority' | 'due' | 'created') {
    const next = cards.filter((card) => {
        const matchesQuery = !filters.query || `${card.title} ${card.description}`.toLowerCase().includes(filters.query.toLowerCase());
        const matchesAssignee = !filters.assignee_id || card.assignee_ids.includes(filters.assignee_id.trim());
        const matchesLabel = !filters.label || card.labels.some((label) => label.toLowerCase().includes(filters.label.toLowerCase()));
        const matchesStatus = !filters.status || card.column_id === filters.status;
        const matchesDateFrom = !filters.date_from || !card.due_date || card.due_date >= filters.date_from;
        const matchesDateTo = !filters.date_to || !card.start_date || card.start_date <= filters.date_to;
        return matchesQuery && matchesAssignee && matchesLabel && matchesStatus && matchesDateFrom && matchesDateTo;
    });

    if (sortMode === 'priority') {
        next.sort((left, right) => PRIORITY_ORDER[left.priority] - PRIORITY_ORDER[right.priority]);
    } else if (sortMode === 'due') {
        next.sort((left, right) => (left.due_date || '9999-12-31').localeCompare(right.due_date || '9999-12-31'));
    } else if (sortMode === 'created') {
        next.sort((left, right) => right.created_at - left.created_at);
    } else {
        const columnOrder = columns.reduce<Record<string, number>>((accumulator, column, index) => {
            accumulator[column.id] = index;
            return accumulator;
        }, {});
        next.sort((left, right) => {
            if (left.column_id !== right.column_id) {
                return (columnOrder[left.column_id] || 0) - (columnOrder[right.column_id] || 0);
            }
            return left.position - right.position;
        });
    }

    return next;
}

export function groupCardsByColumn(cards: Card[]) {
    return cards.reduce<Record<string, Card[]>>((accumulator, card) => {
        accumulator[card.column_id] = accumulator[card.column_id] || [];
        accumulator[card.column_id].push(card);
        return accumulator;
    }, {});
}

export function previousColumnId(columns: BoardColumn[], currentColumnId: string) {
    const index = columns.findIndex((column) => column.id === currentColumnId);
    if (index <= 0) {
        return '';
    }
    return columns[index - 1].id;
}

export function nextColumnId(columns: BoardColumn[], currentColumnId: string) {
    const index = columns.findIndex((column) => column.id === currentColumnId);
    if (index < 0 || index >= columns.length - 1) {
        return '';
    }
    return columns[index + 1].id;
}

export function buildGanttUnits(cards: Card[], zoomLevel: 'day' | 'week' | 'month') {
    const range = resolveTimelineRange(cards);
    const units: Array<{key: string; label: string; start: Date; end: Date}> = [];
    let cursor = new Date(range.start);

    while (cursor <= range.end) {
        if (zoomLevel === 'month') {
            const monthStart = new Date(cursor.getFullYear(), cursor.getMonth(), 1);
            units.push({
                key: monthStart.toISOString(),
                label: `${monthStart.getFullYear()}-${String(monthStart.getMonth() + 1).padStart(2, '0')}`,
                start: monthStart,
                end: new Date(cursor.getFullYear(), cursor.getMonth() + 1, 0),
            });
            cursor = new Date(cursor.getFullYear(), cursor.getMonth() + 1, 1);
            continue;
        }

        if (zoomLevel === 'week') {
            const weekStart = startOfWeek(cursor);
            units.push({
                key: weekStart.toISOString(),
                label: `${weekStart.getMonth() + 1}/${weekStart.getDate()}`,
                start: weekStart,
                end: addDays(weekStart, 6),
            });
            cursor = addDays(weekStart, 7);
            continue;
        }

        units.push({
            key: cursor.toISOString(),
            label: `${cursor.getMonth() + 1}/${cursor.getDate()}`,
            start: new Date(cursor),
            end: new Date(cursor),
        });
        cursor = addDays(cursor, 1);
    }

    return units;
}

export function getScheduledRange(card: Card) {
    if (!card.start_date && !card.due_date) {
        return null;
    }
    return {
        startDate: card.start_date || card.due_date || '',
        dueDate: card.due_date || card.start_date || '',
    };
}

export function resolveBarPosition(units: Array<{start: Date; end: Date}>, startDate: string, dueDate: string) {
    const start = stripTime(new Date(`${startDate}T00:00:00`));
    const end = stripTime(new Date(`${dueDate}T00:00:00`));
    let startIndex = 0;
    let endIndex = units.length - 1;

    units.forEach((unit, index) => {
        if (start >= stripTime(unit.start) && start <= stripTime(unit.end)) {
            startIndex = index;
        }
        if (end >= stripTime(unit.start) && end <= stripTime(unit.end)) {
            endIndex = index;
        }
    });

    if (endIndex < startIndex) {
        endIndex = startIndex;
    }

    return {start: startIndex, end: endIndex};
}

export function formatDateTime(timestamp: number) {
    const date = new Date(timestamp);
    return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
}

export function findCardTitle(cards: Card[], cardId: string) {
    return cards.find((card) => card.id === cardId)?.title || cardId;
}

function resolveTimelineRange(cards: Card[]) {
    const scheduledCards = cards.map((card) => getScheduledRange(card)).filter(Boolean) as Array<{startDate: string; dueDate: string}>;
    if (scheduledCards.length === 0) {
        const today = stripTime(new Date());
        return {
            start: addDays(today, -3),
            end: addDays(today, 10),
        };
    }

    return scheduledCards.reduce((accumulator, range) => {
        const start = stripTime(new Date(`${range.startDate}T00:00:00`));
        const end = stripTime(new Date(`${range.dueDate}T00:00:00`));
        return {
            start: start < accumulator.start ? start : accumulator.start,
            end: end > accumulator.end ? end : accumulator.end,
        };
    }, {
        start: addDays(stripTime(new Date(`${scheduledCards[0].startDate}T00:00:00`)), -2),
        end: addDays(stripTime(new Date(`${scheduledCards[0].dueDate}T00:00:00`)), 3),
    });
}

function startOfWeek(date: Date) {
    const next = stripTime(date);
    const weekday = next.getDay() === 0 ? 7 : next.getDay();
    next.setDate(next.getDate() - weekday + 1);
    return next;
}

function addDays(date: Date, amount: number) {
    const next = new Date(date);
    next.setDate(next.getDate() + amount);
    return next;
}

function stripTime(date: Date) {
    const next = new Date(date);
    next.setHours(0, 0, 0, 0);
    return next;
}
