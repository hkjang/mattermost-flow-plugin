import React, {useCallback, useEffect, useRef, useState} from 'react';

import {flowClient} from './client';
import {subscribeFlowSync} from './flow_sync';
import {
    DEFAULT_BOARD_SETTINGS,
    DEFAULT_COLUMNS,
    EMPTY_FILTERS,
    buildFlowUrl,
    buildGanttUnits,
    createBoardSettingsDraft,
    createCardDraft,
    createColumnInputs,
    createTemplateDraft,
    createTemplateInputs,
    findCardTitle,
    formatDateTime,
    getScheduledRange,
    getVisibleCards,
    groupCardsByColumn,
    nextColumnId,
    parseChecklist,
    parseLinks,
    previousColumnId,
    resolveBarPosition,
    selectBoardId,
    splitCommaValues,
    syncUrl,
    type BoardSettingsDraft,
    type CardEditorDraft,
} from './flow_helpers';
import type {
    Activity,
    BoardCalendarFeedInfo,
    BoardBundle,
    BoardColumn,
    BoardFilters,
    FlowBoardSummaryEvent,
    BoardSummary,
    Card,
    CardTemplate,
    CardMoveResult,
    CardMutationResult,
    CommentMutationResult,
    ContextSnapshot,
    Dependency,
    DependencyMutationResult,
    FlowUser,
    FlowStreamEvent,
} from './types';
import {UserPicker} from './user_picker';

type FlowPageProps = {context: ContextSnapshot};

type GanttDragState = {
    cardId: string;
    mode: 'move' | 'resize-start' | 'resize-end';
    startIndex: number;
    endIndex: number;
    startDate: string;
    dueDate: string;
};

type DashboardMetric = {
    label: string;
    value: string;
    tone?: 'neutral' | 'accent' | 'warning' | 'danger';
    hint?: string;
};

type DashboardBarItem = {
    label: string;
    value: number;
    detail: string;
    tone?: 'neutral' | 'accent' | 'warning' | 'danger';
};

type DashboardCardItem = {
    id: string;
    title: string;
    meta: string;
    detail: string;
    tone?: 'neutral' | 'accent' | 'warning' | 'danger';
};

type DashboardActivityItem = {
    id: string;
    title: string;
    detail: string;
    timestamp: string;
};

type DashboardModel = {
    metrics: DashboardMetric[];
    status: DashboardBarItem[];
    priorities: DashboardBarItem[];
    assignees: DashboardBarItem[];
    upcoming: DashboardCardItem[];
    milestones: DashboardCardItem[];
    activity: DashboardActivityItem[];
    filtered: boolean;
};

const GANTT_UNIT_WIDTH = 54;

export function FlowPage({context}: FlowPageProps) {
    const [boardList, setBoardList] = useState<BoardSummary[]>([]);
    const [boardData, setBoardData] = useState<BoardBundle | null>(null);
    const [selectedBoardId, setSelectedBoardId] = useState('');
    const [selectedCardId, setSelectedCardId] = useState('');
    const [viewType, setViewType] = useState<'board' | 'gantt' | 'dashboard'>('board');
    const [zoomLevel, setZoomLevel] = useState<'day' | 'week' | 'month'>('week');
    const [filters, setFilters] = useState(EMPTY_FILTERS);
    const [sortMode, setSortMode] = useState<'manual' | 'priority' | 'due' | 'created'>('manual');
    const [boardNameDraft, setBoardNameDraft] = useState('');
    const [boardDescriptionDraft, setBoardDescriptionDraft] = useState('');
    const [boardVisibilityDraft, setBoardVisibilityDraft] = useState<'channel' | 'team'>(context.channelId ? 'channel' : 'team');
    const [newCardTitle, setNewCardTitle] = useState('');
    const [newCardDueDate, setNewCardDueDate] = useState('');
    const [newCardAssigneeIds, setNewCardAssigneeIds] = useState<string[]>([]);
    const [newCardPriority, setNewCardPriority] = useState<Card['priority']>('normal');
    const [newCardMilestone, setNewCardMilestone] = useState(false);
    const [newCardTemplateId, setNewCardTemplateId] = useState('');
    const [boardSettingsOpen, setBoardSettingsOpen] = useState(false);
    const [boardSettingsDraft, setBoardSettingsDraft] = useState<BoardSettingsDraft | null>(null);
    const [cardDraft, setCardDraft] = useState<CardEditorDraft | null>(null);
    const [commentDraft, setCommentDraft] = useState('');
    const [dependencyTarget, setDependencyTarget] = useState('');
    const [error, setError] = useState('');
    const [notice, setNotice] = useState('');
    const [loadingBoards, setLoadingBoards] = useState(false);
    const [loadingBoard, setLoadingBoard] = useState(false);
    const [saving, setSaving] = useState(false);
    const [preferencesReady, setPreferencesReady] = useState(false);
    const [ganttDrag, setGanttDrag] = useState<GanttDragState | null>(null);
    const [usersById, setUsersById] = useState<Record<string, FlowUser>>({});
    const [calendarFeedInfo, setCalendarFeedInfo] = useState<BoardCalendarFeedInfo | null>(null);
    const liveRefreshTimerRef = useRef<number | null>(null);
    const pendingStreamEventsRef = useRef<Record<string, number>>({});

    const params = new URLSearchParams(window.location.search);
    const effectiveChannelId = params.get('channel_id') || context.channelId || '';
    const requestedBoardId = params.get('board_id') || '';
    const requestedCardId = params.get('card_id') || '';
    const requestedView = params.get('view') === 'gantt' ? 'gantt' : params.get('view') === 'dashboard' ? 'dashboard' : params.get('view') === 'board' ? 'board' : '';

    useEffect(() => {
        let active = true;
        const loadBoards = async () => {
            if (!context.teamId && !effectiveChannelId) {
                return;
            }
            setLoadingBoards(true);
            try {
                const response = await flowClient.listBoards(context.teamId, effectiveChannelId);
                if (!active) {
                    return;
                }
                setBoardList(response.items);
                const boardIds = response.items.map((item) => item.board.id);
                const defaultBoardId = response.items.find((item) => item.default_board)?.board.id || '';
                const nextBoardId = selectBoardId(boardIds, requestedBoardId, selectedBoardId, defaultBoardId);
                setSelectedBoardId(nextBoardId);
                if (nextBoardId) {
                    syncUrl({
                        boardId: nextBoardId,
                        channelId: effectiveChannelId,
                        cardId: requestedCardId,
                        view: requestedView || undefined,
                    });
                }
            } catch (fetchError) {
                if (active) {
                    setError(getErrorMessage(fetchError));
                }
            } finally {
                if (active) {
                    setLoadingBoards(false);
                }
            }
        };
        void loadBoards();
        return () => {
            active = false;
        };
    }, [context.teamId, effectiveChannelId]);

    useEffect(() => {
        let active = true;
        const loadBoard = async () => {
            if (!selectedBoardId) {
                setBoardData(null);
                return;
            }
            setLoadingBoard(true);
            setPreferencesReady(false);
            try {
                const response = await flowClient.getBoard(selectedBoardId);
                if (!active) {
                    return;
                }
                setBoardData(response);
                setBoardSettingsDraft(createBoardSettingsDraft(response));
                setViewType((requestedView as 'board' | 'gantt' | 'dashboard') || response.preference.view_type || response.board.settings.default_view || 'board');
                setZoomLevel(response.preference.zoom_level || 'week');
                setFilters(response.preference.filters || EMPTY_FILTERS);
                setPreferencesReady(true);
                if (requestedCardId && response.cards.some((card) => card.id === requestedCardId)) {
                    setSelectedCardId(requestedCardId);
                } else if (!response.cards.some((card) => card.id === selectedCardId)) {
                    setSelectedCardId('');
                }
            } catch (fetchError) {
                if (active) {
                    setError(getErrorMessage(fetchError));
                }
            } finally {
                if (active) {
                    setLoadingBoard(false);
                }
            }
        };
        void loadBoard();
        return () => {
            active = false;
        };
    }, [requestedCardId, requestedView, selectedBoardId]);

    useEffect(() => {
        const selectedCard = boardData?.cards.find((card) => card.id === selectedCardId);
        setCardDraft(selectedCard ? createCardDraft(selectedCard) : null);
        setDependencyTarget('');
        setCommentDraft('');
    }, [selectedCardId, boardData]);

    useEffect(() => {
        setUsersById({});
        setNewCardAssigneeIds([]);
        setNewCardTemplateId('');
        setCalendarFeedInfo(null);
    }, [selectedBoardId]);

    useEffect(() => {
        if (!preferencesReady || !selectedBoardId) {
            return;
        }
        const timeout = window.setTimeout(() => {
            void flowClient.savePreference(selectedBoardId, {
                view_type: viewType,
                zoom_level: zoomLevel,
                filters,
            }).catch(() => undefined);
        }, 300);
        return () => window.clearTimeout(timeout);
    }, [preferencesReady, selectedBoardId, viewType, zoomLevel, filters]);

    const selectedBoard = boardData?.board || null;
    const selectedCard = boardData?.cards.find((card) => card.id === selectedCardId) || null;
    const cards = boardData?.cards || [];
    const columns = boardData?.columns || [];
    const templates = boardData?.templates || [];
    const dependencies = boardData?.dependencies || [];
    const visibleCards = getVisibleCards(cards, columns, filters, sortMode);
    const groupedCards = groupCardsByColumn(visibleCards);
    const ganttCards = visibleCards.length > 0 ? visibleCards : cards;
    const ganttUnits = buildGanttUnits(ganttCards, zoomLevel);
    const boardAssignees = collectAssigneeUsers(cards, usersById);
    const selectedTemplate = templates.find((template) => template.id === newCardTemplateId) || null;
    const dashboardModel = buildDashboardModel(visibleCards, columns, dependencies, boardData?.activity || [], usersById, filters);

    useEffect(() => {
        if (newCardTemplateId && !templates.some((template) => template.id === newCardTemplateId)) {
            setNewCardTemplateId('');
        }
    }, [newCardTemplateId, templates]);
    const mergeUsers = useCallback((users: FlowUser[]) => {
        if (users.length === 0) {
            return;
        }

        setUsersById((current) => {
            const next = {...current};
            for (const user of users) {
                next[user.id] = user;
            }
            return next;
        });
    }, []);

    useEffect(() => {
        if (!boardData) {
            return;
        }

        setBoardList((current) => mergeBoardSummary(current, boardData));
    }, [boardData]);

    useEffect(() => {
        if (!selectedBoardId || !boardData) {
            return;
        }

        const userIds = collectBoardUserIds(boardData);
        if (userIds.length === 0) {
            return;
        }

        let active = true;
        void flowClient.searchBoardUsers(selectedBoardId, {
            ids: userIds,
            limit: Math.max(userIds.length, 8),
        }).then((response) => {
            if (active) {
                mergeUsers(response.items);
            }
        }).catch(() => undefined);

        return () => {
            active = false;
        };
    }, [boardData, mergeUsers, selectedBoardId]);

    useEffect(() => {
        if (!selectedBoard) {
            setCalendarFeedInfo(null);
            return;
        }

        setCalendarFeedInfo({
            enabled: selectedBoard.settings.calendar_feed_enabled,
            has_token: false,
            download_url: flowClient.getBoardCalendarDownloadUrl(selectedBoard.id),
        });
    }, [selectedBoard?.id, selectedBoard?.settings.calendar_feed_enabled]);

    useEffect(() => {
        if (!boardSettingsOpen || !selectedBoard) {
            return;
        }

        let active = true;
        void flowClient.getBoardCalendarFeed(selectedBoard.id).then((response) => {
            if (active) {
                setCalendarFeedInfo(response);
            }
        }).catch(() => {
            if (active) {
                setCalendarFeedInfo({
                    enabled: selectedBoard.settings.calendar_feed_enabled,
                    has_token: false,
                    download_url: flowClient.getBoardCalendarDownloadUrl(selectedBoard.id),
                });
            }
        });

        return () => {
            active = false;
        };
    }, [boardSettingsOpen, selectedBoard?.id, selectedBoard?.settings.calendar_feed_enabled]);

    useEffect(() => {
        if (!selectedBoardId || !boardData || boardData.board.id !== selectedBoardId) {
            return;
        }

        syncUrl({
            boardId: selectedBoardId,
            channelId: effectiveChannelId,
            cardId: selectedCardId || undefined,
            view: viewType,
        });
    }, [boardData, effectiveChannelId, selectedBoardId, selectedCardId, viewType]);

    const openBoardContext = useCallback((boardId: string, cardId?: string) => {
        setSelectedBoardId(boardId);
        setSelectedCardId(cardId || '');
        setViewType('board');
        syncUrl({
            boardId,
            channelId: effectiveChannelId,
            cardId,
            view: 'board',
        });
    }, [effectiveChannelId]);

    const openBoardActivity = useCallback((item: BoardSummary) => {
        openBoardContext(item.board.id, resolveActivityCardId(item.recent_activity));
    }, [openBoardContext]);

    const refreshBoards = useCallback(async (nextBoardId?: string) => {
        const response = await flowClient.listBoards(context.teamId, effectiveChannelId);
        setBoardList(response.items);
        const fallbackBoardId = nextBoardId || response.items[0]?.board.id || '';
        setSelectedBoardId(fallbackBoardId);
        setSelectedCardId('');
        if (fallbackBoardId) {
            syncUrl({
                boardId: fallbackBoardId,
                channelId: effectiveChannelId,
                view: viewType,
            });
        }
    }, [context.teamId, effectiveChannelId, viewType]);

    const refreshBoardList = useCallback(async () => {
        const response = await flowClient.listBoards(context.teamId, effectiveChannelId);
        setBoardList(response.items);
    }, [context.teamId, effectiveChannelId]);

    const refreshBoard = useCallback(async () => {
        if (!selectedBoardId) {
            return;
        }
        const response = await flowClient.getBoard(selectedBoardId);
        setBoardData(response);
        setBoardSettingsDraft(createBoardSettingsDraft(response));
        setPreferencesReady(true);
    }, [selectedBoardId]);

    function updateTemplateDraft(index: number, patch: Partial<BoardSettingsDraft['templates'][number]>) {
        if (!boardSettingsDraft) {
            return;
        }

        setBoardSettingsDraft({
            ...boardSettingsDraft,
            templates: boardSettingsDraft.templates.map((template, templateIndex) => (
                templateIndex === index ? {...template, ...patch} : template
            )),
        });
    }

    function addTemplateDraft() {
        if (!boardSettingsDraft) {
            return;
        }

        setBoardSettingsDraft({
            ...boardSettingsDraft,
            templates: [...boardSettingsDraft.templates, createTemplateDraft()],
        });
    }

    function removeTemplateDraft(index: number) {
        if (!boardSettingsDraft) {
            return;
        }

        setBoardSettingsDraft({
            ...boardSettingsDraft,
            templates: boardSettingsDraft.templates.filter((_, templateIndex) => templateIndex !== index),
        });
    }

    function applyTemplateSelection(templateId: string) {
        setNewCardTemplateId(templateId);
        if (!templateId) {
            return;
        }

        const template = templates.find((item) => item.id === templateId);
        if (!template) {
            return;
        }

        setNewCardTitle(template.title || '');
        setNewCardPriority(template.priority);
        setNewCardMilestone(template.milestone);
        setNewCardDueDate(resolveTemplateDate(template.due_offset_days));
    }

    const scheduleLiveRefresh = useCallback((message: {boardId: string; action: string; refreshBoard?: boolean; refreshBoardList?: boolean}) => {
        if (liveRefreshTimerRef.current !== null) {
            window.clearTimeout(liveRefreshTimerRef.current);
        }

        liveRefreshTimerRef.current = window.setTimeout(() => {
            liveRefreshTimerRef.current = null;
            void (async () => {
                if (message.action === 'board.deleted' && message.boardId === selectedBoardId) {
                    await refreshBoards();
                    return;
                }

                if (message.refreshBoardList) {
                    await refreshBoardList();
                }
                if (message.refreshBoard && message.boardId === selectedBoardId) {
                    await refreshBoard();
                }
            })();
        }, 120);
    }, [refreshBoard, refreshBoardList, refreshBoards, selectedBoardId]);

    useEffect(() => {
        return () => {
            if (liveRefreshTimerRef.current !== null) {
                window.clearTimeout(liveRefreshTimerRef.current);
            }
            pendingStreamEventsRef.current = {};
        };
    }, []);

    const rememberLocalEvent = useCallback((boardId: string, action: string, cardId?: string) => {
        pendingStreamEventsRef.current[`${boardId}:${action}:${cardId || ''}`] = Date.now();
    }, []);

    const consumePendingLocalEvent = useCallback((event: FlowStreamEvent) => {
        const key = `${event.board_id}:${event.action}:${event.card_id || ''}`;
        const timestamp = pendingStreamEventsRef.current[key];
        if (!timestamp) {
            return false;
        }

        delete pendingStreamEventsRef.current[key];
        return Date.now() - timestamp < 4000;
    }, []);

    useEffect(() => {
        return subscribeFlowSync((message) => {
            const isKnownBoard = boardList.some((item) => item.board.id === message.boardId);
            if (!isKnownBoard && message.boardId !== selectedBoardId) {
                return;
            }

            const patchable = Boolean(
                message.event &&
                message.boardId === selectedBoardId &&
                canPatchBoardEvent(message.event),
            );
            if (patchable && message.event) {
                setBoardData((current) => applyFlowStreamEvent(current, message.event!));
            }
            const summaryPatched = Boolean(message.summary && message.boardId !== selectedBoardId);
            if (message.summary && message.boardId !== selectedBoardId) {
                setBoardList((current) => mergeBoardSummarySnapshot(current, message.summary!));
            }

            scheduleLiveRefresh({
                boardId: message.boardId,
                action: message.reason,
                refreshBoard: message.boardId === selectedBoardId && !patchable,
                refreshBoardList: message.boardId !== selectedBoardId && !summaryPatched,
            });
        });
    }, [boardList, scheduleLiveRefresh, selectedBoardId]);

    useEffect(() => {
        if (!selectedBoardId) {
            return;
        }

        const stream = new EventSource(flowClient.getBoardStreamUrl(selectedBoardId));
        const handleBoardEvent = (event: MessageEvent<string>) => {
            try {
                const payload = JSON.parse(event.data) as FlowStreamEvent;
                const skipBoardRefresh = consumePendingLocalEvent(payload);
                const patchable = payload.board_id === selectedBoardId && canPatchBoardEvent(payload);
                if (patchable) {
                    setBoardData((current) => applyFlowStreamEvent(current, payload));
                }
                scheduleLiveRefresh({
                    boardId: payload.board_id,
                    action: payload.action,
                    refreshBoard: !patchable && !skipBoardRefresh,
                    refreshBoardList: false,
                });
            } catch {
                // Ignore malformed stream payloads and wait for the next event.
            }
        };

        stream.addEventListener('board_event', handleBoardEvent as EventListener);

        return () => {
            stream.removeEventListener('board_event', handleBoardEvent as EventListener);
            stream.close();
        };
    }, [consumePendingLocalEvent, scheduleLiveRefresh, selectedBoardId]);

    useEffect(() => {
        if (!context.teamId && !effectiveChannelId) {
            return;
        }

        const stream = new EventSource(flowClient.getBoardSummaryStreamUrl(context.teamId, effectiveChannelId));
        const handleSummaryEvent = (event: MessageEvent<string>) => {
            try {
                const payload = JSON.parse(event.data) as FlowBoardSummaryEvent;
                let summaryPatched = false;

                if (payload.action === 'board.deleted') {
                    summaryPatched = true;
                    setBoardList((current) => removeBoardSummary(current, payload.board_id));
                } else if (payload.summary) {
                    summaryPatched = true;
                    setBoardList((current) => mergeBoardSummarySnapshot(current, payload.summary!));
                }

                scheduleLiveRefresh({
                    boardId: payload.board_id,
                    action: payload.action,
                    refreshBoard: false,
                    refreshBoardList: !summaryPatched,
                });
            } catch {
                // Ignore malformed stream payloads and wait for the next event.
            }
        };

        stream.addEventListener('board_summary_event', handleSummaryEvent as EventListener);

        return () => {
            stream.removeEventListener('board_summary_event', handleSummaryEvent as EventListener);
            stream.close();
        };
    }, [context.teamId, effectiveChannelId, scheduleLiveRefresh]);

    async function createBoard(event: React.FormEvent<HTMLFormElement>) {
        event.preventDefault();
        if (!boardNameDraft.trim()) {
            setError('Board name is required.');
            return;
        }
        setSaving(true);
        try {
            const response = await flowClient.createBoard({
                team_id: context.teamId,
                channel_id: boardVisibilityDraft === 'channel' ? effectiveChannelId : undefined,
                name: boardNameDraft.trim(),
                description: boardDescriptionDraft.trim(),
                visibility: boardVisibilityDraft,
                admin_ids: [],
                columns: createColumnInputs(DEFAULT_COLUMNS.join('\n')),
                settings: DEFAULT_BOARD_SETTINGS,
                set_as_default: boardVisibilityDraft === 'channel',
            });
            setBoardNameDraft('');
            setBoardDescriptionDraft('');
            setNotice('Board created.');
            await refreshBoards(response.board.id);
        } catch (createError) {
            setError(getErrorMessage(createError));
        } finally {
            setSaving(false);
        }
    }

    async function updateBoardSettings(event: React.FormEvent<HTMLFormElement>) {
        event.preventDefault();
        if (!boardData || !boardSettingsDraft) {
            return;
        }
        setSaving(true);
        try {
            const response = await flowClient.updateBoard(boardData.board.id, {
                name: boardSettingsDraft.name,
                description: boardSettingsDraft.description,
                columns: createColumnInputs(boardSettingsDraft.columnsText, boardData.columns),
                templates: createTemplateInputs(boardSettingsDraft.templates),
                settings: {
                    post_updates: boardSettingsDraft.postUpdates,
                    post_due_soon: boardSettingsDraft.postDueSoon,
                    allow_mentions: boardSettingsDraft.allowMentions,
                    calendar_feed_enabled: boardSettingsDraft.calendarFeedEnabled,
                    default_view: boardSettingsDraft.defaultView,
                },
                version: boardSettingsDraft.version,
            });
            setBoardData(response);
            setBoardSettingsDraft(createBoardSettingsDraft(response));
            setPreferencesReady(true);
            rememberLocalEvent(response.board.id, 'board.updated');
            setBoardSettingsOpen(false);
            setNotice('Board settings saved.');
        } catch (updateError) {
            setError(getErrorMessage(updateError));
        } finally {
            setSaving(false);
        }
    }

    async function createCard(event: React.FormEvent<HTMLFormElement>) {
        event.preventDefault();
        if (!boardData || boardData.columns.length === 0) {
            return;
        }

        const template = boardData.templates.find((item) => item.id === newCardTemplateId) || null;
        const title = newCardTitle.trim() || template?.title || '';
        if (!title) {
            setError('Card title is required.');
            return;
        }

        setSaving(true);
        try {
            const response = await flowClient.createCard({
                board_id: boardData.board.id,
                column_id: boardData.columns[0].id,
                title,
                description: template?.description || '',
                assignee_ids: newCardAssigneeIds,
                labels: template?.labels || [],
                priority: newCardPriority,
                start_date: resolveTemplateDate(template?.start_offset_days),
                due_date: newCardDueDate || resolveTemplateDate(template?.due_offset_days),
                progress: 0,
                milestone: newCardMilestone,
                checklist: cloneChecklist(template?.checklist || []),
                attachment_links: cloneAttachmentLinks(template?.attachment_links || []),
            });
            setNewCardTitle('');
            setNewCardDueDate('');
            setNewCardAssigneeIds([]);
            setNewCardPriority('normal');
            setNewCardMilestone(false);
            setNewCardTemplateId('');
            setBoardData((current) => applyCardMutation(current, response));
            rememberLocalEvent(response.board.id, 'card.created', response.card.id);
            setNotice(template ? `Card created from template "${template.name}".` : 'Card created.');
        } catch (createError) {
            setError(getErrorMessage(createError));
        } finally {
            setSaving(false);
        }
    }

    async function moveCard(card: Card, targetColumnId: string, targetIndex: number) {
        setSaving(true);
        try {
            const response = await flowClient.moveCard(card.id, {
                target_column_id: targetColumnId,
                target_index: targetIndex,
                version: card.version,
            });
            setBoardData((current) => applyCardMoveMutation(current, response, targetColumnId, targetIndex));
            rememberLocalEvent(response.board.id, 'card.moved', response.card.id);
            setNotice('Card moved.');
        } catch (moveError) {
            setError(getErrorMessage(moveError));
        } finally {
            setSaving(false);
        }
    }

    async function updateCardSchedule(card: Card, startDate: string, dueDate: string) {
        if (!startDate || !dueDate) {
            return;
        }
        if (card.start_date === startDate && card.due_date === dueDate) {
            return;
        }

        setSaving(true);
        try {
            const response = await flowClient.updateCard(card.id, {
                start_date: startDate,
                due_date: dueDate,
                version: card.version,
            });
            setBoardData((current) => applyCardMutation(current, response));
            rememberLocalEvent(response.board.id, 'card.updated', response.card.id);
            setNotice('Schedule updated.');
        } catch (updateError) {
            setError(getErrorMessage(updateError));
        } finally {
            setSaving(false);
        }
    }

    function beginGanttDrag(event: React.DragEvent<HTMLElement>, card: Card, mode: GanttDragState['mode'], startIndex: number, endIndex: number, startDate: string, dueDate: string) {
        event.dataTransfer.effectAllowed = 'move';
        event.dataTransfer.setData('text/plain', `${card.id}:${mode}`);
        setGanttDrag({
            cardId: card.id,
            mode,
            startIndex,
            endIndex,
            startDate,
            dueDate,
        });
    }

    async function completeGanttDrop(card: Card, targetIndex: number) {
        if (!ganttDrag || ganttDrag.cardId !== card.id || targetIndex < 0 || targetIndex >= ganttUnits.length) {
            return;
        }

        const dragState = ganttDrag;
        setGanttDrag(null);

        let nextStartDate = dragState.startDate;
        let nextDueDate = dragState.dueDate;

        if (dragState.mode === 'move') {
            const deltaDays = diffInDays(ganttUnits[targetIndex].start, ganttUnits[dragState.startIndex].start);
            nextStartDate = shiftDayString(dragState.startDate, deltaDays);
            nextDueDate = shiftDayString(dragState.dueDate, deltaDays);
        } else if (dragState.mode === 'resize-start') {
            const nextStartIndex = Math.min(targetIndex, dragState.endIndex);
            nextStartDate = formatInputDate(ganttUnits[nextStartIndex].start);
        } else {
            const nextEndIndex = Math.max(targetIndex, dragState.startIndex);
            nextDueDate = formatInputDate(ganttUnits[nextEndIndex].end);
        }

        await updateCardSchedule(card, nextStartDate, nextDueDate);
    }

    async function saveCardDetails(event: React.FormEvent<HTMLFormElement>) {
        event.preventDefault();
        if (!selectedCard || !cardDraft) {
            return;
        }
        setSaving(true);
        try {
            const response = await flowClient.updateCard(selectedCard.id, {
                title: cardDraft.title.trim(),
                description: cardDraft.description.trim(),
                assignee_ids: cardDraft.assigneeIds,
                labels: splitCommaValues(cardDraft.labelsText),
                priority: cardDraft.priority,
                start_date: cardDraft.startDate || '',
                due_date: cardDraft.dueDate || '',
                progress: Number(cardDraft.progress),
                milestone: cardDraft.milestone,
                checklist: parseChecklist(cardDraft.checklistText, selectedCard.checklist),
                attachment_links: parseLinks(cardDraft.linksText, selectedCard.attachment_links),
                version: cardDraft.version,
            });
            setBoardData((current) => applyCardMutation(current, response));
            rememberLocalEvent(response.board.id, 'card.updated', response.card.id);
            setNotice('Card updated.');
        } catch (updateError) {
            setError(getErrorMessage(updateError));
        } finally {
            setSaving(false);
        }
    }

    async function addComment(event: React.FormEvent<HTMLFormElement>) {
        event.preventDefault();
        if (!selectedCard || !commentDraft.trim()) {
            return;
        }
        setSaving(true);
        try {
            const response = await flowClient.addComment(selectedCard.id, commentDraft.trim());
            setCommentDraft('');
            setBoardData((current) => applyCardMutation(current, response));
            rememberLocalEvent(response.board.id, 'comment.created', response.card.id);
            setNotice('Comment added.');
        } catch (commentError) {
            setError(getErrorMessage(commentError));
        } finally {
            setSaving(false);
        }
    }

    async function addDependency() {
        if (!selectedCard || !dependencyTarget) {
            return;
        }
        setSaving(true);
        try {
            const response = await flowClient.addDependency(selectedCard.id, dependencyTarget);
            setDependencyTarget('');
            setBoardData((current) => applyDependencyMutation(current, response));
            rememberLocalEvent(response.board.id, 'dependency.created', selectedCard.id);
            setNotice('Dependency added.');
        } catch (dependencyError) {
            setError(getErrorMessage(dependencyError));
        } finally {
            setSaving(false);
        }
    }

    async function deleteBoard() {
        if (!selectedBoard || !window.confirm(`Delete '${selectedBoard.name}'?`)) {
            return;
        }
        setSaving(true);
        try {
            await flowClient.deleteBoard(selectedBoard.id);
            await refreshBoards();
            setNotice('Board deleted.');
        } catch (deleteError) {
            setError(getErrorMessage(deleteError));
        } finally {
            setSaving(false);
        }
    }

    async function copyLink(url: string, message: string) {
        try {
            await navigator.clipboard.writeText(url);
            setNotice(message);
        } catch {
            window.prompt('Copy link', url);
        }
    }

    function openCalendarDownload() {
        if (!selectedBoard) {
            return;
        }

        window.open(flowClient.getBoardCalendarDownloadUrl(selectedBoard.id), '_blank', 'noopener,noreferrer');
    }

    function copyCalendarSubscriptionLink() {
        if (!calendarFeedInfo?.subscribe_url) {
            return;
        }

        void copyLink(calendarFeedInfo.subscribe_url, 'Calendar subscription link copied.');
    }

    async function rotateCalendarFeed() {
        if (!selectedBoard) {
            return;
        }

        setSaving(true);
        try {
            const response = await flowClient.rotateBoardCalendarFeed(selectedBoard.id);
            setCalendarFeedInfo(response);
            rememberLocalEvent(selectedBoard.id, 'board.calendar.rotated');
            setNotice('Calendar subscription token rotated.');
        } catch (rotateError) {
            setError(getErrorMessage(rotateError));
        } finally {
            setSaving(false);
        }
    }

    function copyBoardLink(nextView: 'board' | 'gantt' | 'dashboard') {
        if (!selectedBoardId) {
            return;
        }

        void copyLink(buildFlowUrl({
            boardId: selectedBoardId,
            channelId: effectiveChannelId,
            view: nextView,
        }), nextView === 'gantt' ? 'Gantt link copied.' : nextView === 'dashboard' ? 'Dashboard link copied.' : 'Board link copied.');
    }

    function copyCardLink() {
        if (!selectedBoardId || !selectedCardId) {
            return;
        }

        void copyLink(buildFlowUrl({
            boardId: selectedBoardId,
            channelId: effectiveChannelId,
            cardId: selectedCardId,
            view: viewType,
        }), 'Card link copied.');
    }

    return (
        <div className='flow-shell'>
            <aside className='flow-sidebar'>
                <div className='flow-sidebar__brand'>
                    <div className='flow-sidebar__eyebrow'>Mattermost Flow</div>
                    <h1>Kanban + Gantt</h1>
                    <p>{context.channelDisplayName || context.teamDisplayName || 'Collaborative planning'}</p>
                </div>

                <form className='flow-creation' onSubmit={createBoard}>
                    <label>
                        Board name
                        <input value={boardNameDraft} onChange={(event) => setBoardNameDraft(event.target.value)} placeholder='Release planning'/>
                    </label>
                    <label>
                        Description
                        <textarea rows={3} value={boardDescriptionDraft} onChange={(event) => setBoardDescriptionDraft(event.target.value)} placeholder='Keep channel context and task flow together.'/>
                    </label>
                    <label>
                        Scope
                        <select value={boardVisibilityDraft} onChange={(event) => setBoardVisibilityDraft(event.target.value as 'channel' | 'team')}>
                            {effectiveChannelId && <option value='channel'>Current channel</option>}
                            <option value='team'>Team board</option>
                        </select>
                    </label>
                    <button className='flow-button flow-button--primary' type='submit' disabled={saving}>Create board</button>
                </form>

                <div className='flow-boardlist'>
                    <div className='flow-boardlist__header'>
                        <span>Boards</span>
                        {loadingBoards && <span className='flow-status'>Loading</span>}
                    </div>
                    {boardList.map((item) => (
                        <div
                            key={item.board.id}
                            className={`flow-boardlist__item ${item.board.id === selectedBoardId ? 'is-active' : ''}`}
                        >
                            <button
                                className='flow-boardlist__select'
                                type='button'
                                onClick={() => openBoardContext(item.board.id)}
                            >
                                <div className='flow-boardlist__row'>
                                    <div className='flow-boardlist__title'>
                                        <strong>{item.board.name}</strong>
                                        {isRecentBoardActivity(item.recent_activity) && item.board.id !== selectedBoardId && <span className='flow-boardlist__signal' aria-hidden='true'/>}
                                    </div>
                                    {item.default_board && <span className='flow-chip flow-chip--accent'>Default</span>}
                                </div>
                                {item.recent_activity && (
                                    <div className='flow-boardlist__hint'>
                                        <span>{describeBoardActivity(item.recent_activity)}</span>
                                        <span>{formatRelativeTime(item.recent_activity.created_at)}</span>
                                    </div>
                                )}
                                <div className='flow-boardlist__meta'>
                                    <span>{item.card_count} cards</span>
                                    <span>{item.overdue_count} overdue</span>
                                </div>
                            </button>
                            {item.recent_activity && (
                                <button
                                    className='flow-boardlist__jump'
                                    type='button'
                                    onClick={() => openBoardActivity(item)}
                                >
                                    Open update
                                </button>
                            )}
                        </div>
                    ))}
                </div>
            </aside>

            <main className='flow-main'>
                <div className='flow-toolbar'>
                    <div>
                        <div className='flow-toolbar__eyebrow'>{selectedBoard?.visibility === 'channel' ? 'Channel board' : 'Team board'}</div>
                        <h2>{selectedBoard?.name || 'Select a board'}</h2>
                        <p>{selectedBoard?.description || 'Kanban board and gantt timeline in one plugin.'}</p>
                    </div>
                    <div className='flow-toolbar__actions'>
                        <button className={`flow-button ${viewType === 'board' ? 'flow-button--primary' : ''}`} onClick={() => setViewType('board')}>Board</button>
                        <button className={`flow-button ${viewType === 'dashboard' ? 'flow-button--primary' : ''}`} onClick={() => setViewType('dashboard')}>Dashboard</button>
                        <button className={`flow-button ${viewType === 'gantt' ? 'flow-button--primary' : ''}`} onClick={() => setViewType('gantt')}>Gantt</button>
                        <button className='flow-button' onClick={() => copyBoardLink('board')} disabled={!selectedBoard}>Copy board link</button>
                        <button className='flow-button' onClick={() => copyBoardLink('dashboard')} disabled={!selectedBoard}>Copy dashboard link</button>
                        <button className='flow-button' onClick={() => copyBoardLink('gantt')} disabled={!selectedBoard}>Copy gantt link</button>
                        <button className='flow-button' onClick={openCalendarDownload} disabled={!selectedBoard}>Open calendar .ics</button>
                        <button className='flow-button' onClick={() => setBoardSettingsOpen((value) => !value)} disabled={!selectedBoard}>Settings</button>
                        <button className='flow-button flow-button--danger' onClick={deleteBoard} disabled={!selectedBoard || saving}>Delete</button>
                    </div>
                </div>

                {error && <div className='flow-alert flow-alert--error'>{error}</div>}
                {notice && <div className='flow-alert flow-alert--success'>{notice}</div>}
                {loadingBoard && <div className='flow-alert'>Loading board data.</div>}

                {selectedBoard && boardSettingsOpen && boardSettingsDraft && (
                    <form className='flow-settings' onSubmit={updateBoardSettings}>
                        <div className='flow-settings__header'>
                            <strong>Board settings</strong>
                            <span>Columns, notifications, default view, and templates</span>
                        </div>
                        <div className='flow-settings__grid'>
                            <label>
                                Name
                                <input value={boardSettingsDraft.name} onChange={(event) => setBoardSettingsDraft({...boardSettingsDraft, name: event.target.value})}/>
                            </label>
                            <label>
                                Default view
                                <select value={boardSettingsDraft.defaultView} onChange={(event) => setBoardSettingsDraft({...boardSettingsDraft, defaultView: event.target.value as 'board' | 'gantt' | 'dashboard'})}>
                                    <option value='board'>Board</option>
                                    <option value='dashboard'>Dashboard</option>
                                    <option value='gantt'>Gantt</option>
                                </select>
                            </label>
                            <label className='flow-settings__wide'>
                                Description
                                <textarea rows={3} value={boardSettingsDraft.description} onChange={(event) => setBoardSettingsDraft({...boardSettingsDraft, description: event.target.value})}/>
                            </label>
                            <label className='flow-settings__wide'>
                                Columns
                                <textarea rows={4} value={boardSettingsDraft.columnsText} onChange={(event) => setBoardSettingsDraft({...boardSettingsDraft, columnsText: event.target.value})} placeholder='Todo|0&#10;In Progress|3&#10;Review|2&#10;Done|0'/>
                            </label>
                        </div>
                        <div className='flow-toggle-row'>
                            <label><input checked={boardSettingsDraft.postUpdates} onChange={(event) => setBoardSettingsDraft({...boardSettingsDraft, postUpdates: event.target.checked})} type='checkbox'/> Post channel updates</label>
                            <label><input checked={boardSettingsDraft.postDueSoon} onChange={(event) => setBoardSettingsDraft({...boardSettingsDraft, postDueSoon: event.target.checked})} type='checkbox'/> Post due soon alerts</label>
                            <label><input checked={boardSettingsDraft.allowMentions} onChange={(event) => setBoardSettingsDraft({...boardSettingsDraft, allowMentions: event.target.checked})} type='checkbox'/> Allow mentions</label>
                            <label><input checked={boardSettingsDraft.calendarFeedEnabled} onChange={(event) => setBoardSettingsDraft({...boardSettingsDraft, calendarFeedEnabled: event.target.checked})} type='checkbox'/> Enable calendar feed</label>
                        </div>
                        <div className='flow-calendar-panel'>
                            <div className='flow-template-editor__header'>
                                <div>
                                    <strong>Calendar integration</strong>
                                    <span>Publish scheduled cards as an iCalendar feed for Google Calendar, Apple Calendar, or Outlook.</span>
                                </div>
                                <div className='flow-calendar-panel__actions'>
                                    <button className='flow-button' type='button' onClick={openCalendarDownload}>Open .ics</button>
                                    <button className='flow-button' type='button' onClick={copyCalendarSubscriptionLink} disabled={!selectedBoard.settings.calendar_feed_enabled || !calendarFeedInfo?.subscribe_url}>Copy subscription link</button>
                                    <button className='flow-button' type='button' onClick={rotateCalendarFeed} disabled={!selectedBoard.settings.calendar_feed_enabled || saving}>Rotate token</button>
                                </div>
                            </div>
                            {!boardSettingsDraft.calendarFeedEnabled && (
                                <div className='flow-empty'>Enable the calendar feed and save board settings to generate a shareable subscription URL.</div>
                            )}
                            {boardSettingsDraft.calendarFeedEnabled && (
                                <div className='flow-calendar-panel__content'>
                                    <label className='flow-settings__wide'>
                                        Subscription URL
                                        <input readOnly value={calendarFeedInfo?.subscribe_url || ''} placeholder='Save settings to generate a calendar subscription URL'/>
                                    </label>
                                    <div className='flow-calendar-panel__meta'>
                                        <span>Use the `.ics` link for one-time downloads, or the subscription URL for live calendar sync.</span>
                                        {calendarFeedInfo?.updated_at ? <span>Last rotated {formatDateTime(calendarFeedInfo.updated_at)}</span> : <span>Save settings to mint the first token.</span>}
                                    </div>
                                </div>
                            )}
                        </div>
                        <div className='flow-template-editor'>
                            <div className='flow-template-editor__header'>
                                <div>
                                    <strong>Card templates</strong>
                                    <span>Save reusable card defaults for release, bugfix, handoff, and milestone work.</span>
                                </div>
                                <button className='flow-button' type='button' onClick={addTemplateDraft}>Add template</button>
                            </div>
                            {boardSettingsDraft.templates.length === 0 && <div className='flow-empty'>No templates yet.</div>}
                            {boardSettingsDraft.templates.map((template, index) => (
                                <div className='flow-template-card' key={template.id || `template-${index}`}>
                                    <div className='flow-template-card__header'>
                                        <strong>{template.name || `Template ${index + 1}`}</strong>
                                        <button className='flow-button' type='button' onClick={() => removeTemplateDraft(index)}>Remove</button>
                                    </div>
                                    <div className='flow-settings__grid'>
                                        <label>
                                            Template name
                                            <input value={template.name} onChange={(event) => updateTemplateDraft(index, {name: event.target.value})} placeholder='Release checklist'/>
                                        </label>
                                        <label>
                                            Default title
                                            <input value={template.title} onChange={(event) => updateTemplateDraft(index, {title: event.target.value})} placeholder='Ship release'/>
                                        </label>
                                        <label className='flow-settings__wide'>
                                            Description
                                            <textarea rows={3} value={template.description} onChange={(event) => updateTemplateDraft(index, {description: event.target.value})} placeholder='Default scope, exit criteria, or handoff notes'/>
                                        </label>
                                        <label>
                                            Priority
                                            <select value={template.priority} onChange={(event) => updateTemplateDraft(index, {priority: event.target.value as Card['priority']})}>
                                                <option value='urgent'>Urgent</option>
                                                <option value='high'>High</option>
                                                <option value='normal'>Normal</option>
                                                <option value='low'>Low</option>
                                            </select>
                                        </label>
                                        <label>
                                            Start offset (days)
                                            <input type='number' min='0' max='365' value={template.startOffsetDays} onChange={(event) => updateTemplateDraft(index, {startOffsetDays: event.target.value})} placeholder='0'/>
                                        </label>
                                        <label>
                                            Due offset (days)
                                            <input type='number' min='0' max='365' value={template.dueOffsetDays} onChange={(event) => updateTemplateDraft(index, {dueOffsetDays: event.target.value})} placeholder='3'/>
                                        </label>
                                        <label className='flow-settings__wide'>
                                            Labels
                                            <input value={template.labelsText} onChange={(event) => updateTemplateDraft(index, {labelsText: event.target.value})} placeholder='release, qa, docs'/>
                                        </label>
                                        <label className='flow-settings__wide flow-inline-checkbox'>
                                            <input checked={template.milestone} onChange={(event) => updateTemplateDraft(index, {milestone: event.target.checked})} type='checkbox'/>
                                            Mark cards from this template as milestones
                                        </label>
                                        <label className='flow-settings__wide'>
                                            Checklist
                                            <textarea rows={4} value={template.checklistText} onChange={(event) => updateTemplateDraft(index, {checklistText: event.target.value})} placeholder='[ ] Draft release notes&#10;[ ] Validate migration&#10;[ ] Announce in channel'/>
                                        </label>
                                        <label className='flow-settings__wide'>
                                            Links
                                            <textarea rows={3} value={template.linksText} onChange={(event) => updateTemplateDraft(index, {linksText: event.target.value})} placeholder='Runbook|https://example.com/runbook'/>
                                        </label>
                                    </div>
                                </div>
                            ))}
                        </div>
                        <div className='flow-settings__actions'>
                            <button className='flow-button flow-button--primary' type='submit' disabled={saving}>Save</button>
                            <button className='flow-button' type='button' onClick={() => setBoardSettingsOpen(false)}>Close</button>
                        </div>
                    </form>
                )}

                {selectedBoard && (
                    <>
                        <div className='flow-filterbar'>
                            <label>Search<input value={filters.query} onChange={(event) => setFilters({...filters, query: event.target.value})} placeholder='Title or description'/></label>
                            <label>Assignee<select value={filters.assignee_id} onChange={(event) => setFilters({...filters, assignee_id: event.target.value})}><option value=''>All</option>{boardAssignees.map((user) => <option key={user.id} value={user.id}>{formatUserName(user.id, usersById)}</option>)}</select></label>
                            <label>Label<input value={filters.label} onChange={(event) => setFilters({...filters, label: event.target.value})} placeholder='release'/></label>
                            <label>Status<select value={filters.status} onChange={(event) => setFilters({...filters, status: event.target.value})}><option value=''>All</option>{columns.map((column) => <option key={column.id} value={column.id}>{column.name}</option>)}</select></label>
                            <label>Sort<select value={sortMode} onChange={(event) => setSortMode(event.target.value as 'manual' | 'priority' | 'due' | 'created')}><option value='manual'>Manual</option><option value='priority'>Priority</option><option value='due'>Due date</option><option value='created'>Created</option></select></label>
                            <label>Zoom<select value={zoomLevel} onChange={(event) => setZoomLevel(event.target.value as 'day' | 'week' | 'month')}><option value='day'>Day</option><option value='week'>Week</option><option value='month'>Month</option></select></label>
                        </div>

                        <form className='flow-quick-create' onSubmit={createCard}>
                            <label>
                                Template
                                <select value={newCardTemplateId} onChange={(event) => applyTemplateSelection(event.target.value)} disabled={saving || templates.length === 0}>
                                    <option value=''>Blank card</option>
                                    {templates.map((template) => <option key={template.id} value={template.id}>{template.name}</option>)}
                                </select>
                            </label>
                            <input value={newCardTitle} onChange={(event) => setNewCardTitle(event.target.value)} placeholder='New card title'/>
                            <input type='date' value={newCardDueDate} onChange={(event) => setNewCardDueDate(event.target.value)}/>
                            <select value={newCardPriority} onChange={(event) => setNewCardPriority(event.target.value as Card['priority'])}><option value='normal'>Normal</option><option value='high'>High</option><option value='urgent'>Urgent</option><option value='low'>Low</option></select>
                            <label className='flow-inline-checkbox'><input checked={newCardMilestone} onChange={(event) => setNewCardMilestone(event.target.checked)} type='checkbox'/> milestone</label>
                            <button className='flow-button flow-button--primary' type='submit' disabled={saving}>Add card</button>
                            {selectedTemplate && (
                                <div className='flow-quick-create__wide flow-template-hint'>
                                    Template applies hidden defaults too: {describeTemplateCoverage(selectedTemplate)}.
                                </div>
                            )}
                            <div className='flow-form-field flow-quick-create__wide'>
                                <span>Assignees</span>
                                <UserPicker
                                    boardId={selectedBoard.id}
                                    selectedIds={newCardAssigneeIds}
                                    knownUsers={usersById}
                                    onUsersLoaded={mergeUsers}
                                    onChange={setNewCardAssigneeIds}
                                    placeholder='Assign teammates before creating the card'
                                    disabled={saving}
                                />
                            </div>
                        </form>
                    </>
                )}

                <div className='flow-layout'>
                    <section className='flow-board-view'>
                        {viewType === 'board' && (
                            <div className='flow-columns'>
                                {columns.map((column) => {
                                    const columnCards = groupedCards[column.id] || [];
                                    return (
                                        <div
                                            key={column.id}
                                            className='flow-column'
                                            onDragOver={(event) => event.preventDefault()}
                                            onDrop={(event) => {
                                                const cardId = event.dataTransfer.getData('text/plain');
                                                const card = cards.find((item) => item.id === cardId);
                                                if (card) {
                                                    void moveCard(card, column.id, columnCards.length);
                                                }
                                            }}
                                        >
                                            <div className='flow-column__header'>
                                                <div>
                                                    <strong>{column.name}</strong>
                                                    <span>{columnCards.length} cards</span>
                                                </div>
                                                {column.wip_limit > 0 && <span className='flow-chip'>WIP {column.wip_limit}</span>}
                                            </div>
                                            <div className='flow-column__body'>
                                                {columnCards.map((card, index) => (
                                                    <div
                                                        key={card.id}
                                                        className='flow-card'
                                                        draggable
                                                        onDragStart={(event) => event.dataTransfer.setData('text/plain', card.id)}
                                                        onClick={() => setSelectedCardId(card.id)}
                                                    >
                                                        <div className='flow-card__header'>
                                                            <strong>{card.title}</strong>
                                                            <span className={`flow-priority flow-priority--${card.priority}`}>{card.priority}</span>
                                                        </div>
                                                        <p>{card.description || 'Add details to capture the channel context.'}</p>
                                                        <div className='flow-card__meta'>
                                                            <span>{card.progress}%</span>
                                                            <span>{card.due_date || 'No due'}</span>
                                                        </div>
                                                        <div className='flow-card__chips'>
                                                            {card.milestone && <span className='flow-chip flow-chip--accent'>Milestone</span>}
                                                            {card.labels.map((label) => <span className='flow-chip' key={label}>{label}</span>)}
                                                        </div>
                                                        <div className='flow-card__actions'>
                                                            <button type='button' onClick={(event) => {
                                                                event.stopPropagation();
                                                                const previousColumn = previousColumnId(columns, card.column_id);
                                                                if (previousColumn) {
                                                                    void moveCard(card, previousColumn, Math.max(index - 1, 0));
                                                                }
                                                            }}>Prev</button>
                                                            <button type='button' onClick={(event) => {
                                                                event.stopPropagation();
                                                                const next = nextColumnId(columns, card.column_id);
                                                                if (next) {
                                                                    void moveCard(card, next, index);
                                                                }
                                                            }}>Next</button>
                                                        </div>
                                                    </div>
                                                ))}
                                                {columnCards.length === 0 && <div className='flow-empty'>No cards in this column.</div>}
                                            </div>
                                        </div>
                                    );
                                })}
                            </div>
                        )}

                        {viewType === 'dashboard' && (
                            <div className='flow-dashboard'>
                                {dashboardModel.filtered && <div className='flow-alert'>Dashboard metrics reflect the current filters.</div>}
                                <div className='flow-dashboard__metrics'>
                                    {dashboardModel.metrics.map((metric) => (
                                        <article className={`flow-dashboard__metric flow-dashboard__metric--${metric.tone || 'neutral'}`} key={metric.label}>
                                            <span>{metric.label}</span>
                                            <strong>{metric.value}</strong>
                                            {metric.hint && <small>{metric.hint}</small>}
                                        </article>
                                    ))}
                                </div>

                                <div className='flow-dashboard__grid'>
                                    <section className='flow-dashboard__panel'>
                                        <div className='flow-dashboard__panel-header'>
                                            <strong>Status distribution</strong>
                                            <span>Cards by column</span>
                                        </div>
                                        <div className='flow-dashboard__bars'>
                                            {dashboardModel.status.map((item) => (
                                                <div className='flow-dashboard__bar-row' key={item.label}>
                                                    <div className='flow-dashboard__bar-label'>
                                                        <strong>{item.label}</strong>
                                                        <span>{item.detail}</span>
                                                    </div>
                                                    <div className='flow-dashboard__bar-track'>
                                                        <span className={`flow-dashboard__bar-fill flow-dashboard__bar-fill--${item.tone || 'neutral'}`} style={{width: `${item.value}%`}}/>
                                                    </div>
                                                </div>
                                            ))}
                                        </div>
                                    </section>

                                    <section className='flow-dashboard__panel'>
                                        <div className='flow-dashboard__panel-header'>
                                            <strong>Priority mix</strong>
                                            <span>Current workload pressure</span>
                                        </div>
                                        <div className='flow-dashboard__bars'>
                                            {dashboardModel.priorities.map((item) => (
                                                <div className='flow-dashboard__bar-row' key={item.label}>
                                                    <div className='flow-dashboard__bar-label'>
                                                        <strong>{item.label}</strong>
                                                        <span>{item.detail}</span>
                                                    </div>
                                                    <div className='flow-dashboard__bar-track'>
                                                        <span className={`flow-dashboard__bar-fill flow-dashboard__bar-fill--${item.tone || 'neutral'}`} style={{width: `${item.value}%`}}/>
                                                    </div>
                                                </div>
                                            ))}
                                        </div>
                                    </section>

                                    <section className='flow-dashboard__panel'>
                                        <div className='flow-dashboard__panel-header'>
                                            <strong>Assignee load</strong>
                                            <span>Open cards per person</span>
                                        </div>
                                        <div className='flow-dashboard__bars'>
                                            {dashboardModel.assignees.length === 0 && <div className='flow-empty'>No active assignees yet.</div>}
                                            {dashboardModel.assignees.map((item) => (
                                                <div className='flow-dashboard__bar-row' key={item.label}>
                                                    <div className='flow-dashboard__bar-label'>
                                                        <strong>{item.label}</strong>
                                                        <span>{item.detail}</span>
                                                    </div>
                                                    <div className='flow-dashboard__bar-track'>
                                                        <span className={`flow-dashboard__bar-fill flow-dashboard__bar-fill--${item.tone || 'neutral'}`} style={{width: `${item.value}%`}}/>
                                                    </div>
                                                </div>
                                            ))}
                                        </div>
                                    </section>

                                    <section className='flow-dashboard__panel'>
                                        <div className='flow-dashboard__panel-header'>
                                            <strong>Upcoming due</strong>
                                            <span>Open the card detail from here</span>
                                        </div>
                                        <div className='flow-dashboard__list'>
                                            {dashboardModel.upcoming.length === 0 && <div className='flow-empty'>No upcoming due cards.</div>}
                                            {dashboardModel.upcoming.map((item) => (
                                                <button className={`flow-dashboard__card flow-dashboard__card--${item.tone || 'neutral'}`} key={item.id} onClick={() => setSelectedCardId(item.id)}>
                                                    <strong>{item.title}</strong>
                                                    <span>{item.meta}</span>
                                                    <small>{item.detail}</small>
                                                </button>
                                            ))}
                                        </div>
                                    </section>

                                    <section className='flow-dashboard__panel'>
                                        <div className='flow-dashboard__panel-header'>
                                            <strong>Milestones</strong>
                                            <span>Cards flagged as milestones</span>
                                        </div>
                                        <div className='flow-dashboard__list'>
                                            {dashboardModel.milestones.length === 0 && <div className='flow-empty'>No milestone cards yet.</div>}
                                            {dashboardModel.milestones.map((item) => (
                                                <button className={`flow-dashboard__card flow-dashboard__card--${item.tone || 'neutral'}`} key={item.id} onClick={() => setSelectedCardId(item.id)}>
                                                    <strong>{item.title}</strong>
                                                    <span>{item.meta}</span>
                                                    <small>{item.detail}</small>
                                                </button>
                                            ))}
                                        </div>
                                    </section>

                                    <section className='flow-dashboard__panel'>
                                        <div className='flow-dashboard__panel-header'>
                                            <strong>Recent activity</strong>
                                            <span>Latest movement on the board</span>
                                        </div>
                                        <div className='flow-dashboard__activity'>
                                            {dashboardModel.activity.length === 0 && <div className='flow-empty'>No activity yet.</div>}
                                            {dashboardModel.activity.map((item) => (
                                                <div className='flow-dashboard__activity-item' key={item.id}>
                                                    <strong>{item.title}</strong>
                                                    <span>{item.detail}</span>
                                                    <small>{item.timestamp}</small>
                                                </div>
                                            ))}
                                        </div>
                                    </section>
                                </div>
                            </div>
                        )}

                        {viewType === 'gantt' && (
                            <div className='flow-gantt'>
                                <div className='flow-gantt__hint'>Drag the center pill to move a task, or drag Start and End to resize the schedule.</div>
                                <div className='flow-gantt__header' style={{gridTemplateColumns: `240px repeat(${ganttUnits.length}, minmax(${GANTT_UNIT_WIDTH}px, 1fr))`}}>
                                    <div className='flow-gantt__cell flow-gantt__cell--label'>Task</div>
                                    {ganttUnits.map((unit) => <div className='flow-gantt__cell' key={unit.key}>{unit.label}</div>)}
                                </div>
                                {ganttCards.map((card) => {
                                    const range = getScheduledRange(card);
                                    const position = range ? resolveBarPosition(ganttUnits, range.startDate, range.dueDate) : null;
                                    return (
                                        <div className='flow-gantt__row' key={card.id}>
                                            <button className='flow-gantt__title' onClick={() => setSelectedCardId(card.id)}>
                                                <strong>{card.title}</strong>
                                                <span>{range ? `${range.startDate} - ${range.dueDate}` : 'No dates'}</span>
                                            </button>
                                            <div className='flow-gantt__timeline' style={{gridTemplateColumns: `repeat(${ganttUnits.length}, minmax(${GANTT_UNIT_WIDTH}px, 1fr))`}}>
                                                {ganttUnits.map((unit, index) => (
                                                    <div
                                                        className={`flow-gantt__grid ${ganttDrag?.cardId === card.id ? 'is-drop-target' : ''}`}
                                                        key={`${card.id}-${unit.key}`}
                                                        onDragOver={(event) => {
                                                            if (ganttDrag?.cardId === card.id) {
                                                                event.preventDefault();
                                                                event.dataTransfer.dropEffect = 'move';
                                                            }
                                                        }}
                                                        onDrop={(event) => {
                                                            event.preventDefault();
                                                            void completeGanttDrop(card, index);
                                                        }}
                                                    />
                                                ))}
                                                {position && range && (
                                                    <div className='flow-gantt__bar' style={{gridColumn: `${position.start + 1} / ${position.end + 2}`}}>
                                                        <span
                                                            className='flow-gantt__handle'
                                                            draggable
                                                            onDragEnd={() => setGanttDrag(null)}
                                                            onDragStart={(event) => beginGanttDrag(event, card, 'resize-start', position.start, position.end, range.startDate, range.dueDate)}
                                                        >
                                                            Start
                                                        </span>
                                                        <span
                                                            className='flow-gantt__bar-label'
                                                            draggable
                                                            onDragEnd={() => setGanttDrag(null)}
                                                            onDragStart={(event) => beginGanttDrag(event, card, 'move', position.start, position.end, range.startDate, range.dueDate)}
                                                        >
                                                            {card.progress}% move
                                                        </span>
                                                        <span
                                                            className='flow-gantt__handle'
                                                            draggable
                                                            onDragEnd={() => setGanttDrag(null)}
                                                            onDragStart={(event) => beginGanttDrag(event, card, 'resize-end', position.start, position.end, range.startDate, range.dueDate)}
                                                        >
                                                            End
                                                        </span>
                                                    </div>
                                                )}
                                            </div>
                                        </div>
                                    );
                                })}
                            </div>
                        )}
                    </section>

                    <aside className='flow-detail'>
                        {!selectedCard && <div className='flow-empty flow-empty--panel'>Select a card to edit details, comments, and dependencies.</div>}
                        {selectedCard && cardDraft && (
                            <>
                                <div className='flow-detail__header'>
                                    <div>
                                        <div className='flow-toolbar__eyebrow'>Card detail</div>
                                        <h3>{selectedCard.title}</h3>
                                    </div>
                                    <div className='flow-detail__header-actions'>
                                        <button className='flow-button' type='button' onClick={copyCardLink}>Copy card link</button>
                                        <span className={`flow-priority flow-priority--${selectedCard.priority}`}>{selectedCard.priority}</span>
                                    </div>
                                </div>
                                <form className='flow-detail__form' onSubmit={saveCardDetails}>
                                    <label>Title<input value={cardDraft.title} onChange={(event) => setCardDraft({...cardDraft, title: event.target.value})}/></label>
                                    <label>Description<textarea rows={4} value={cardDraft.description} onChange={(event) => setCardDraft({...cardDraft, description: event.target.value})}/></label>
                                    <div className='flow-form-field'>
                                        <span>Assignees</span>
                                        <UserPicker
                                            boardId={selectedBoardId}
                                            selectedIds={cardDraft.assigneeIds}
                                            knownUsers={usersById}
                                            onUsersLoaded={mergeUsers}
                                            onChange={(assigneeIds) => setCardDraft({...cardDraft, assigneeIds})}
                                            disabled={saving}
                                        />
                                    </div>
                                    <label>Labels<input value={cardDraft.labelsText} onChange={(event) => setCardDraft({...cardDraft, labelsText: event.target.value})} placeholder='release, qa'/></label>
                                    <div className='flow-detail__split'>
                                        <label>Priority<select value={cardDraft.priority} onChange={(event) => setCardDraft({...cardDraft, priority: event.target.value as Card['priority']})}><option value='urgent'>Urgent</option><option value='high'>High</option><option value='normal'>Normal</option><option value='low'>Low</option></select></label>
                                        <label>Progress<input type='number' min='0' max='100' value={cardDraft.progress} onChange={(event) => setCardDraft({...cardDraft, progress: Number(event.target.value)})}/></label>
                                    </div>
                                    <div className='flow-detail__split'>
                                        <label>Start<input type='date' value={cardDraft.startDate} onChange={(event) => setCardDraft({...cardDraft, startDate: event.target.value})}/></label>
                                        <label>Due<input type='date' value={cardDraft.dueDate} onChange={(event) => setCardDraft({...cardDraft, dueDate: event.target.value})}/></label>
                                    </div>
                                    <label className='flow-inline-checkbox'><input checked={cardDraft.milestone} onChange={(event) => setCardDraft({...cardDraft, milestone: event.target.checked})} type='checkbox'/> Mark as milestone</label>
                                    <label>Checklist<textarea rows={4} value={cardDraft.checklistText} onChange={(event) => setCardDraft({...cardDraft, checklistText: event.target.value})} placeholder='[ ] API spec&#10;[x] Column design'/></label>
                                    <label>Links<textarea rows={3} value={cardDraft.linksText} onChange={(event) => setCardDraft({...cardDraft, linksText: event.target.value})} placeholder='Spec|https://example.com/spec'/></label>
                                    <button className='flow-button flow-button--primary' type='submit' disabled={saving}>Save card</button>
                                </form>

                                <div className='flow-detail__section'>
                                    <div className='flow-detail__section-header'><strong>Dependencies</strong></div>
                                    <div className='flow-detail__inline'>
                                        <select value={dependencyTarget} onChange={(event) => setDependencyTarget(event.target.value)}>
                                            <option value=''>Select target card</option>
                                            {cards.filter((card) => card.id !== selectedCard.id).map((card) => <option key={card.id} value={card.id}>{card.title}</option>)}
                                        </select>
                                        <button className='flow-button' type='button' onClick={addDependency} disabled={!dependencyTarget || saving}>Add</button>
                                    </div>
                                    <div className='flow-chip-row'>{dependencies.filter((dependency) => dependency.source_card_id === selectedCard.id).map((dependency) => <span className='flow-chip' key={dependency.id}>{dependency.type}: {findCardTitle(cards, dependency.target_card_id)}</span>)}</div>
                                </div>

                                <div className='flow-detail__section'>
                                    <div className='flow-detail__section-header'><strong>Comments</strong></div>
                                    <div className='flow-comment-list'>
                                        {selectedCard.comments.map((comment) => <div className='flow-comment' key={comment.id}><strong>{formatUserName(comment.user_id, usersById)}</strong><span>{formatDateTime(comment.created_at)}</span><p>{comment.message}</p></div>)}
                                        {selectedCard.comments.length === 0 && <div className='flow-empty'>No comments yet.</div>}
                                    </div>
                                    <form className='flow-detail__inline' onSubmit={addComment}>
                                        <input value={commentDraft} onChange={(event) => setCommentDraft(event.target.value)} placeholder='Add a comment'/>
                                        <button className='flow-button' type='submit' disabled={saving}>Post</button>
                                    </form>
                                </div>

                                <div className='flow-detail__section'>
                                    <div className='flow-detail__section-header'><strong>Activity</strong></div>
                                    <div className='flow-comment-list'>
                                        {(boardData?.activity.filter((item) => item.entity_id === selectedCard.id).slice(0, 6) || []).map((item) => <div className='flow-comment' key={item.id}><strong>{item.action}</strong><span>{formatDateTime(item.created_at)}</span><p>{formatUserName(item.actor_id, usersById)}</p></div>)}
                                    </div>
                                </div>
                            </>
                        )}
                    </aside>
                </div>
            </main>
        </div>
    );
}

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : 'Unexpected error.';
}

function formatInputDate(date: Date) {
    return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

function shiftDayString(value: string, deltaDays: number) {
    const next = new Date(`${value}T00:00:00`);
    next.setDate(next.getDate() + deltaDays);
    return formatInputDate(next);
}

function diffInDays(target: Date, source: Date) {
    const targetDate = new Date(target);
    const sourceDate = new Date(source);
    targetDate.setHours(0, 0, 0, 0);
    sourceDate.setHours(0, 0, 0, 0);
    return Math.round((targetDate.getTime() - sourceDate.getTime()) / 86400000);
}

function resolveTemplateDate(offset?: number) {
    if (offset === undefined) {
        return '';
    }

    const next = new Date();
    next.setHours(0, 0, 0, 0);
    next.setDate(next.getDate() + offset);
    return formatInputDate(next);
}

function cloneChecklist(checklist: CardTemplate['checklist']) {
    return checklist.map((item) => ({
        id: '',
        text: item.text,
        completed: item.completed,
    }));
}

function cloneAttachmentLinks(links: CardTemplate['attachment_links']) {
    return links.map((link) => ({
        id: '',
        title: link.title,
        url: link.url,
    }));
}

function describeTemplateCoverage(template: CardTemplate) {
    const parts = ['milestone state'];

    if (template.description) {
        parts.unshift('description');
    }
    if (template.labels.length > 0) {
        parts.push('labels');
    }
    if (template.checklist.length > 0) {
        parts.push('checklist');
    }
    if (template.attachment_links.length > 0) {
        parts.push('links');
    }
    if (template.start_offset_days !== undefined) {
        parts.push('start date');
    }
    if (template.due_offset_days !== undefined) {
        parts.push('due date');
    }

    if (parts.length === 1) {
        return parts[0];
    }

    return `${parts.slice(0, -1).join(', ')}, and ${parts[parts.length - 1]}`;
}

function buildDashboardModel(cards: Card[], columns: BoardColumn[], dependencies: Dependency[], activity: Activity[], usersById: Record<string, FlowUser>, filters: BoardFilters): DashboardModel {
    const today = startOfUtcDay(new Date());
    const dueSoonLimit = new Date(today.getTime() + (72 * 60 * 60 * 1000));
    const visibleCardIds = new Set(cards.map((card) => card.id));
    const openCards: Card[] = [];
    let doneCount = 0;
    let overdueCount = 0;
    let dueSoonCount = 0;
    let milestoneCount = 0;
    let progressTotal = 0;
    let unscheduledCount = 0;

    for (const card of cards) {
        progressTotal += card.progress;
        if (card.progress >= 100) {
            doneCount++;
        } else {
            openCards.push(card);
        }
        if (card.milestone) {
            milestoneCount++;
        }
        if (!card.start_date && !card.due_date) {
            unscheduledCount++;
        }

        if (!card.due_date || card.progress >= 100) {
            continue;
        }

        const dueDate = parseUtcDay(card.due_date);
        if (!dueDate) {
            continue;
        }

        if (dueDate < today) {
            overdueCount++;
        } else if (dueDate < dueSoonLimit) {
            dueSoonCount++;
        }
    }

    const blockedCardIds = new Set<string>();
    for (const dependency of dependencies) {
        if (!visibleCardIds.has(dependency.source_card_id) && !visibleCardIds.has(dependency.target_card_id)) {
            continue;
        }
        if (visibleCardIds.has(dependency.source_card_id)) {
            blockedCardIds.add(dependency.source_card_id);
        }
        if (visibleCardIds.has(dependency.target_card_id)) {
            blockedCardIds.add(dependency.target_card_id);
        }
    }

    const totalCards = cards.length;
    const averageProgress = totalCards === 0 ? 0 : Math.round(progressTotal / totalCards);
    const filtered = hasActiveFilters(filters);
    const priorityOrder: Card['priority'][] = ['urgent', 'high', 'normal', 'low'];
    const assigneeCounts = new Map<string, number>();

    for (const card of openCards) {
        if (card.assignee_ids.length === 0) {
            assigneeCounts.set('unassigned', (assigneeCounts.get('unassigned') || 0) + 1);
            continue;
        }
        for (const assigneeId of card.assignee_ids) {
            assigneeCounts.set(assigneeId, (assigneeCounts.get(assigneeId) || 0) + 1);
        }
    }

    const assigneeMax = Math.max(...Array.from(assigneeCounts.values()), 0);

    return {
        metrics: [
            {label: 'Cards in view', value: String(totalCards), tone: 'neutral', hint: filtered ? 'Current filters applied' : 'Whole board'},
            {label: 'Completed', value: String(doneCount), tone: 'accent', hint: totalCards === 0 ? '0%' : `${Math.round((doneCount / totalCards) * 100)}% done`},
            {label: 'Overdue', value: String(overdueCount), tone: overdueCount > 0 ? 'danger' : 'neutral', hint: 'Open cards past due'},
            {label: 'Due soon', value: String(dueSoonCount), tone: dueSoonCount > 0 ? 'warning' : 'neutral', hint: 'Due within 72 hours'},
            {label: 'Milestones', value: String(milestoneCount), tone: milestoneCount > 0 ? 'accent' : 'neutral', hint: 'Flagged delivery anchors'},
            {label: 'Avg progress', value: `${averageProgress}%`, tone: averageProgress >= 70 ? 'accent' : averageProgress >= 40 ? 'warning' : 'neutral', hint: 'Across visible cards'},
            {label: 'Linked cards', value: String(blockedCardIds.size), tone: blockedCardIds.size > 0 ? 'warning' : 'neutral', hint: 'Cards in dependencies'},
            {label: 'No dates', value: String(unscheduledCount), tone: unscheduledCount > 0 ? 'warning' : 'neutral', hint: 'Missing start and due dates'},
        ],
        status: columns.map((column) => {
            const count = cards.filter((card) => card.column_id === column.id).length;
            const ratio = totalCards === 0 ? 0 : Math.round((count / totalCards) * 100);
            const overLimit = column.wip_limit > 0 && count > column.wip_limit;
            return {
                label: column.name,
                value: ratio,
                detail: `${count} cards${column.wip_limit > 0 ? ` · WIP ${column.wip_limit}` : ''}`,
                tone: overLimit ? 'danger' : count > 0 ? 'accent' : 'neutral',
            };
        }),
        priorities: priorityOrder.map((priority) => {
            const count = cards.filter((card) => card.priority === priority).length;
            const ratio = totalCards === 0 ? 0 : Math.round((count / totalCards) * 100);
            return {
                label: priority[0].toUpperCase() + priority.slice(1),
                value: ratio,
                detail: `${count} cards`,
                tone: priority === 'urgent' ? 'danger' : priority === 'high' ? 'warning' : priority === 'normal' ? 'accent' : 'neutral',
            };
        }),
        assignees: Array.from(assigneeCounts.entries()).sort((left, right) => right[1] - left[1]).slice(0, 6).map(([assigneeId, count]) => ({
            label: assigneeId === 'unassigned' ? 'Unassigned' : formatUserName(assigneeId, usersById),
            value: assigneeMax === 0 ? 0 : Math.round((count / assigneeMax) * 100),
            detail: `${count} open cards`,
            tone: count >= 5 ? 'danger' : count >= 3 ? 'warning' : 'accent',
        })),
        upcoming: openCards
            .filter((card) => card.due_date)
            .sort((left, right) => (left.due_date || '').localeCompare(right.due_date || ''))
            .slice(0, 5)
            .map((card) => {
                const dueDate = card.due_date ? parseUtcDay(card.due_date) : null;
                return {
                    id: card.id,
                    title: card.title,
                    meta: `${card.due_date || 'No due'} · ${columnName(columns, card.column_id)}`,
                    detail: summarizeCardPeople(card.assignee_ids, usersById),
                    tone: dueDate && dueDate < today ? 'danger' : dueDate && dueDate < dueSoonLimit ? 'warning' : 'neutral',
                };
            }),
        milestones: cards
            .filter((card) => card.milestone)
            .sort((left, right) => {
                if (!left.due_date && !right.due_date) {
                    return right.updated_at - left.updated_at;
                }
                return (left.due_date || '9999-12-31').localeCompare(right.due_date || '9999-12-31');
            })
            .slice(0, 5)
            .map((card) => {
                const dueDate = card.due_date ? parseUtcDay(card.due_date) : null;
                return {
                    id: card.id,
                    title: card.title,
                    meta: `${card.due_date || 'No due'} · ${card.progress}%`,
                    detail: summarizeCardPeople(card.assignee_ids, usersById),
                    tone: card.progress >= 100 ? 'accent' : dueDate && dueDate < dueSoonLimit ? 'warning' : 'neutral',
                };
            }),
        activity: activity.slice(0, 6).map((item) => ({
            id: item.id,
            title: describeBoardActivity(item),
            detail: formatUserName(item.actor_id, usersById),
            timestamp: formatRelativeTime(item.created_at),
        })),
        filtered,
    };
}

function hasActiveFilters(filters: BoardFilters) {
    return Boolean(filters.query || filters.assignee_id || filters.label || filters.status || filters.date_from || filters.date_to);
}

function summarizeCardPeople(assigneeIds: string[], usersById: Record<string, FlowUser>) {
    if (assigneeIds.length === 0) {
        return 'No assignees';
    }

    return assigneeIds.slice(0, 2).map((assigneeId) => formatUserName(assigneeId, usersById)).join(', ');
}

function columnName(columns: BoardColumn[], columnId: string) {
    return columns.find((column) => column.id === columnId)?.name || 'Column';
}

function canPatchBoardEvent(event: FlowStreamEvent) {
    if (event.entity_type === 'board') {
        return false;
    }
    if (event.action === 'card.moved' || event.action === 'card.completed') {
        return Boolean(event.card && event.column_card_ids);
    }
    return Boolean(event.card || event.dependency || event.activity);
}

function applyFlowStreamEvent(bundle: BoardBundle | null, event: FlowStreamEvent) {
    if (!bundle || bundle.board.id !== event.board_id) {
        return bundle;
    }

    const baseCards = event.card ? upsertById(bundle.cards, event.card) : bundle.cards;
    const nextCards = event.column_card_ids ? applyColumnCardOrder(baseCards, event.column_card_ids) : baseCards;

    return withUpdatedBundle(bundle, {
        ...bundle,
        board: event.board || bundle.board,
        cards: nextCards,
        dependencies: event.dependency ? upsertById(bundle.dependencies, event.dependency) : bundle.dependencies,
        activity: event.activity ? prependActivity(bundle.activity, event.activity) : bundle.activity,
    });
}

function applyCardMutation(bundle: BoardBundle | null, result: CardMutationResult | CardMoveResult | CommentMutationResult) {
    if (!bundle || bundle.board.id !== result.board.id) {
        return bundle;
    }

    return withUpdatedBundle(bundle, {
        ...bundle,
        board: result.board,
        cards: upsertById(bundle.cards, result.card),
    });
}

function applyDependencyMutation(bundle: BoardBundle | null, result: DependencyMutationResult) {
    if (!bundle || bundle.board.id !== result.board.id) {
        return bundle;
    }

    return withUpdatedBundle(bundle, {
        ...bundle,
        board: result.board,
        dependencies: upsertById(bundle.dependencies, result.dependency),
    });
}

function applyCardMoveMutation(bundle: BoardBundle | null, result: CardMoveResult, targetColumnId: string, targetIndex: number) {
    if (!bundle || bundle.board.id !== result.board.id) {
        return bundle;
    }

    return withUpdatedBundle(bundle, {
        ...bundle,
        board: result.board,
        cards: moveCardsLocally(bundle.cards, result.card, targetColumnId, targetIndex),
    });
}

function withUpdatedBundle(previous: BoardBundle, next: BoardBundle) {
    return {
        ...next,
        summary: buildBoardSummarySnapshot(next, previous.summary.default_board),
    };
}

function mergeBoardSummary(items: BoardSummary[], bundle: BoardBundle) {
    const index = items.findIndex((item) => item.board.id === bundle.board.id);
    const isDefault = index >= 0 ? items[index].default_board : bundle.summary.default_board;
    const nextSummary = buildBoardSummarySnapshot(bundle, isDefault);
    if (index === -1) {
        return sortBoardSummaries([nextSummary, ...items]);
    }

    const nextItems = [...items];
    nextItems[index] = nextSummary;
    return sortBoardSummaries(nextItems);
}

function mergeBoardSummarySnapshot(items: BoardSummary[], summary: BoardSummary) {
    const index = items.findIndex((item) => item.board.id === summary.board.id);
    if (index === -1) {
        return sortBoardSummaries([summary, ...items]);
    }

    const nextItems = [...items];
    nextItems[index] = summary;
    return sortBoardSummaries(nextItems);
}

function removeBoardSummary(items: BoardSummary[], boardId: string) {
    return items.filter((item) => item.board.id !== boardId);
}

function sortBoardSummaries(items: BoardSummary[]) {
    return [...items].sort((left, right) => {
        if (left.board.updated_at === right.board.updated_at) {
            return left.board.name.localeCompare(right.board.name);
        }
        return right.board.updated_at - left.board.updated_at;
    });
}

function buildBoardSummarySnapshot(bundle: BoardBundle, isDefault: boolean): BoardSummary {
    const assigneeIds = new Set<string>();
    let overdueCount = 0;
    let dueSoonCount = 0;
    const today = startOfUtcDay(new Date());
    const dueSoonLimit = new Date(today.getTime() + (72 * 60 * 60 * 1000));

    for (const card of bundle.cards) {
        for (const assigneeId of card.assignee_ids) {
            assigneeIds.add(assigneeId);
        }

        if (!card.due_date || card.progress >= 100) {
            continue;
        }

        const dueDate = parseUtcDay(card.due_date);
        if (!dueDate) {
            continue;
        }

        if (dueDate < today) {
            overdueCount++;
        }
        if (dueDate >= today && dueDate < dueSoonLimit) {
            dueSoonCount++;
        }
    }

    return {
        board: bundle.board,
        card_count: bundle.cards.length,
        overdue_count: overdueCount,
        due_soon_count: dueSoonCount,
        default_board: isDefault,
        columns: bundle.columns.length,
        assignees: Array.from(assigneeIds).sort(),
        recent_activity: bundle.activity[0],
    };
}

function upsertById<T extends {id: string}>(items: T[], nextItem: T) {
    const nextItems = items.filter((item) => item.id !== nextItem.id);
    return [nextItem, ...nextItems];
}

function prependActivity(items: Activity[], nextItem: Activity) {
    const nextItems = items.filter((item) => item.id !== nextItem.id);
    return [nextItem, ...nextItems];
}

function applyColumnCardOrder(cards: Card[], columnCardIds: Record<string, string[]>) {
    const cardsById = new Map(cards.map((card) => [card.id, {...card}]));

    Object.entries(columnCardIds).forEach(([columnId, orderedCardIds]) => {
        orderedCardIds.forEach((cardId, index) => {
            const current = cardsById.get(cardId);
            if (!current) {
                return;
            }

            current.column_id = columnId;
            current.position = index;
        });
    });

    return Array.from(cardsById.values());
}

function moveCardsLocally(cards: Card[], movedCard: Card, targetColumnId: string, targetIndex: number) {
    const remaining: Card[] = [];
    const targetCards: Card[] = [];

    for (const card of cards) {
        if (card.id === movedCard.id) {
            continue;
        }
        if (card.column_id === targetColumnId) {
            targetCards.push({...card});
        } else {
            remaining.push({...card});
        }
    }

    targetCards.sort((left, right) => left.position - right.position);
    const safeIndex = Math.max(0, Math.min(targetIndex, targetCards.length));
    const inserted = [
        ...targetCards.slice(0, safeIndex),
        {...movedCard, column_id: targetColumnId},
        ...targetCards.slice(safeIndex),
    ];

    return reindexCardPositions([...remaining, ...inserted]);
}

function reindexCardPositions(cards: Card[]) {
    const next = cards.map((card) => ({...card}));
    const groupedByColumn = next.reduce<Record<string, Card[]>>((accumulator, card) => {
        accumulator[card.column_id] = accumulator[card.column_id] || [];
        accumulator[card.column_id].push(card);
        return accumulator;
    }, {});

    for (const columnCards of Object.values(groupedByColumn)) {
        columnCards.sort((left, right) => left.position - right.position);
        columnCards.forEach((card, index) => {
            card.position = index;
        });
    }

    return next;
}

function parseUtcDay(value: string) {
    const date = new Date(`${value}T00:00:00Z`);
    return Number.isNaN(date.getTime()) ? null : date;
}

function startOfUtcDay(date: Date) {
    return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()));
}

function collectBoardUserIds(bundle: BoardBundle) {
    const userIds = new Set<string>();

    userIds.add(bundle.board.created_by);

    for (const card of bundle.cards) {
        userIds.add(card.created_by);
        userIds.add(card.updated_by);
        for (const assigneeId of card.assignee_ids) {
            userIds.add(assigneeId);
        }
        for (const comment of card.comments) {
            userIds.add(comment.user_id);
        }
    }

    for (const item of bundle.activity) {
        userIds.add(item.actor_id);
    }

    return Array.from(userIds).filter(Boolean);
}

function collectAssigneeUsers(cards: Card[], usersById: Record<string, FlowUser>) {
    const seen = new Set<string>();
    const items: FlowUser[] = [];

    for (const card of cards) {
        for (const assigneeId of card.assignee_ids) {
            if (!assigneeId || seen.has(assigneeId)) {
                continue;
            }
            seen.add(assigneeId);
            items.push(usersById[assigneeId] || {
                id: assigneeId,
                username: assigneeId,
                display_name: assigneeId,
            });
        }
    }

    return items.sort((left, right) => formatUserName(left.id, usersById).localeCompare(formatUserName(right.id, usersById)));
}

function formatUserName(userId: string, usersById: Record<string, FlowUser>) {
    const user = usersById[userId];
    if (!user) {
        return userId;
    }
    if (user.display_name && user.display_name !== user.username) {
        return `${user.display_name} (@${user.username})`;
    }
    return `@${user.username}`;
}

function describeBoardActivity(activity: Activity) {
    switch (activity.action) {
    case 'board.created':
        return 'Board created';
    case 'board.updated':
        return 'Board settings updated';
    case 'card.created':
        return 'New card added';
    case 'card.updated':
        return 'Card details updated';
    case 'card.moved':
        return 'Card moved';
    case 'card.assignee_added':
        return 'Assignee updated';
    case 'card.due_date_updated':
        return 'Due date changed';
    case 'card.checklist_item_completed':
        return 'Checklist progressed';
    case 'card.completed':
        return 'Card marked done';
    case 'dependency.created':
        return 'Dependency linked';
    case 'comment.created':
        return 'New comment';
    default:
        return activity.action.replace(/\./g, ' ');
    }
}

function formatRelativeTime(timestamp: number) {
    const diff = Date.now() - timestamp;
    if (diff < 60000) {
        return 'just now';
    }
    if (diff < 3600000) {
        return `${Math.max(1, Math.round(diff / 60000))}m ago`;
    }
    if (diff < 86400000) {
        return `${Math.max(1, Math.round(diff / 3600000))}h ago`;
    }
    return `${Math.max(1, Math.round(diff / 86400000))}d ago`;
}

function isRecentBoardActivity(activity?: Activity) {
    if (!activity) {
        return false;
    }
    return Date.now() - activity.created_at < 10 * 60 * 1000;
}

function resolveActivityCardId(activity?: Activity) {
    if (!activity) {
        return '';
    }

    if (activity.entity_type === 'card') {
        return activity.entity_id;
    }

    if (activity.entity_type === 'comment') {
        return readActivityStringField(activity, 'card_id');
    }

    if (activity.entity_type === 'dependency') {
        return readActivityStringField(activity, 'source_card_id') || readActivityStringField(activity, 'target_card_id');
    }

    return '';
}

function readActivityStringField(activity: Activity, field: string) {
    const after = readRecord(activity.after);
    if (typeof after?.[field] === 'string') {
        return after[field] as string;
    }

    const before = readRecord(activity.before);
    if (typeof before?.[field] === 'string') {
        return before[field] as string;
    }

    return '';
}

function readRecord(value: unknown) {
    if (!value || typeof value !== 'object') {
        return null;
    }

    return value as Record<string, unknown>;
}
