import React, {useEffect, useState} from 'react';

import type {Post} from '@mattermost/types/posts';

import {flowClient} from './client';
import {publishFlowSync} from './flow_sync';
import type {FlowStreamEvent} from './types';

type FlowPostProps = {
    post: Post;
    currentUserId?: string;
};

function readString(post: Post, key: string) {
    const value = post.props?.[key];
    return typeof value === 'string' ? value : '';
}

function readBoolean(post: Post, key: string) {
    return post.props?.[key] === true;
}

function readNumber(post: Post, key: string) {
    const value = post.props?.[key];
    return typeof value === 'number' ? value : 0;
}

function readStringArray(post: Post, key: string) {
    const value = post.props?.[key];
    if (!Array.isArray(value)) {
        return [];
    }

    return value.filter((item): item is string => typeof item === 'string');
}

function getSummary(post: Post) {
    const summary = readString(post, 'summary');
    if (summary) {
        return summary;
    }

    const firstLine = post.message.split('\n').map((line) => line.trim()).find(Boolean) || '';
    return firstLine.replace(/^\[Flow\]\s*/i, '');
}

function openLink(event: React.MouseEvent<HTMLButtonElement>, url: string) {
    event.preventDefault();
    event.stopPropagation();
    if (!url) {
        return;
    }

    window.location.assign(url);
}

function getErrorMessage(error: unknown) {
    if (error instanceof Error) {
        return error.message;
    }
    return 'Action failed';
}

function describeChecklist(checklist: Array<{completed: boolean; text: string}>) {
    const totalCount = checklist.length;
    let completedCount = 0;
    let nextItemText = '';

    for (const item of checklist) {
        if (item.completed) {
            completedCount++;
            continue;
        }
        if (!nextItemText) {
            nextItemText = item.text;
        }
    }

    return {
        completedCount,
        totalCount,
        nextItemText,
    };
}

function toFlowSyncEvent(response: Awaited<ReturnType<typeof flowClient.runCardAction>>): FlowStreamEvent {
    return {
        type: 'board_event',
        board_id: response.board_id,
        entity_type: 'card',
        action: response.event_action,
        card_id: response.card.id,
        occurred_at: Date.now(),
        board: response.board,
        card: response.card,
        column_card_ids: response.column_card_ids,
    };
}

export function FlowPost({post, currentUserId}: FlowPostProps) {
    const flowType = readString(post, 'flow_type') === 'due_soon' ? 'due_soon' : 'update';
    const badgeText = flowType === 'due_soon' ? 'Due soon' : 'Flow update';
    const cardId = readString(post, 'card_id');
    const cardTitle = readString(post, 'card_title') || 'Flow card';
    const boardName = readString(post, 'board_name');
    const dueDate = readString(post, 'due_date');
    const cardLinkURL = readString(post, 'card_link_url') || readString(post, 'link_url');
    const ganttLinkURL = readString(post, 'gantt_link_url');
    const summary = getSummary(post);

    const [busyAction, setBusyAction] = useState('');
    const [feedback, setFeedback] = useState('');
    const [assignedToCurrentUser, setAssignedToCurrentUser] = useState(false);
    const [hasNextColumn, setHasNextColumn] = useState(readBoolean(post, 'has_next_column'));
    const [currentColumnName, setCurrentColumnName] = useState(readString(post, 'current_column_name'));
    const [nextColumnName, setNextColumnName] = useState(readString(post, 'next_column_name'));
    const [currentDueDate, setCurrentDueDate] = useState(dueDate);
    const [currentProgress, setCurrentProgress] = useState(readNumber(post, 'progress'));
    const [checklistCompletedCount, setChecklistCompletedCount] = useState(readNumber(post, 'checklist_completed_count'));
    const [checklistTotalCount, setChecklistTotalCount] = useState(readNumber(post, 'checklist_total_count'));
    const [nextChecklistItem, setNextChecklistItem] = useState(readString(post, 'next_checklist_item'));
    const [hasDoneColumn, setHasDoneColumn] = useState(readBoolean(post, 'has_done_column'));
    const [inDoneColumn, setInDoneColumn] = useState(readBoolean(post, 'in_done_column'));
    const [doneColumnName, setDoneColumnName] = useState(readString(post, 'done_column_name'));

    useEffect(() => {
        const assigneeIDs = readStringArray(post, 'assignee_ids');
        setAssignedToCurrentUser(Boolean(currentUserId && assigneeIDs.includes(currentUserId)));
        setHasNextColumn(readBoolean(post, 'has_next_column'));
        setCurrentColumnName(readString(post, 'current_column_name'));
        setNextColumnName(readString(post, 'next_column_name'));
        setCurrentDueDate(readString(post, 'due_date'));
        setCurrentProgress(readNumber(post, 'progress'));
        setChecklistCompletedCount(readNumber(post, 'checklist_completed_count'));
        setChecklistTotalCount(readNumber(post, 'checklist_total_count'));
        setNextChecklistItem(readString(post, 'next_checklist_item'));
        setHasDoneColumn(readBoolean(post, 'has_done_column'));
        setInDoneColumn(readBoolean(post, 'in_done_column'));
        setDoneColumnName(readString(post, 'done_column_name'));
        setBusyAction('');
        setFeedback('');
    }, [currentUserId, post.id, post.update_at]);

    async function runAction(event: React.MouseEvent<HTMLButtonElement>, action: 'assign-self' | 'move-next' | 'push-1d' | 'push-7d' | 'complete-next-checklist' | 'complete-card') {
        event.preventDefault();
        event.stopPropagation();

        if (!cardId) {
            return;
        }

        setBusyAction(action);
        setFeedback('');

        try {
            const response = await flowClient.runCardAction(cardId, action);
            setFeedback(response.message);
            setCurrentColumnName(response.current_column_name);
            setNextColumnName(response.next_column_name || '');
            setHasNextColumn(response.has_next_column);
            setDoneColumnName(response.done_column_name || '');
            setHasDoneColumn(response.has_done_column);
            setInDoneColumn(response.in_done_column);
            setCurrentDueDate(response.card.due_date || '');
            setCurrentProgress(response.card.progress);
            const checklistState = describeChecklist(response.card.checklist);
            setChecklistCompletedCount(checklistState.completedCount);
            setChecklistTotalCount(checklistState.totalCount);
            setNextChecklistItem(checklistState.nextItemText);
            if (currentUserId) {
                setAssignedToCurrentUser(response.card.assignee_ids.includes(currentUserId));
            }
            if (response.status === 'applied') {
                publishFlowSync({
                    boardId: response.board_id,
                    cardId: response.card.id,
                    reason: response.event_action,
                    event: toFlowSyncEvent(response),
                    summary: response.summary,
                });
            }
        } catch (error) {
            setFeedback(getErrorMessage(error));
        } finally {
            setBusyAction('');
        }
    }

    const openActions = flowType === 'due_soon' ? [
        {label: 'Open gantt', url: ganttLinkURL, primary: true},
        {label: 'Open card', url: cardLinkURL, primary: false},
    ] : [
        {label: 'Open card', url: cardLinkURL, primary: true},
        {label: 'Open gantt', url: ganttLinkURL, primary: false},
    ];
    const checklistReadyForDone = checklistTotalCount === 0 || checklistCompletedCount === checklistTotalCount;
    const cardAlreadyDone = currentProgress >= 100 && (!hasDoneColumn || inDoneColumn);

    return (
        <div className={`flow-post flow-post--${flowType.replace('_', '-')}`}>
            <div className='flow-post__eyebrow'>{badgeText}</div>
            <div className='flow-post__title'>{cardTitle}</div>
            {summary && <div className='flow-post__summary'>{summary}</div>}
            {(boardName || currentDueDate || currentColumnName) && (
                <div className='flow-post__meta'>
                    {boardName && <span>{boardName}</span>}
                    {currentColumnName && <span>{currentColumnName}</span>}
                    {currentDueDate && <span>Due {currentDueDate}</span>}
                    <span>{currentProgress}% progress</span>
                    {checklistTotalCount > 0 && <span>{checklistCompletedCount}/{checklistTotalCount} checklist</span>}
                </div>
            )}
            {nextChecklistItem && (
                <div className='flow-post__summary'>Next checklist item: {nextChecklistItem}</div>
            )}
            <div className='flow-post__actions'>
                <button
                    type='button'
                    className='flow-button'
                    onClick={(event) => void runAction(event, 'assign-self')}
                    disabled={!cardId || !currentUserId || assignedToCurrentUser || busyAction !== ''}
                >
                    {busyAction === 'assign-self' ? 'Assigning...' : assignedToCurrentUser ? 'Assigned to you' : 'Assign to me'}
                </button>
                <button
                    type='button'
                    className='flow-button'
                    onClick={(event) => void runAction(event, 'move-next')}
                    disabled={!cardId || !hasNextColumn || busyAction !== ''}
                >
                    {busyAction === 'move-next' ? 'Moving...' : nextColumnName ? `Move to ${nextColumnName}` : 'Move to next'}
                </button>
                <button
                    type='button'
                    className='flow-button'
                    onClick={(event) => void runAction(event, 'complete-next-checklist')}
                    disabled={!cardId || !nextChecklistItem || busyAction !== ''}
                >
                    {busyAction === 'complete-next-checklist' ? 'Updating...' : nextChecklistItem ? 'Complete next item' : 'Checklist complete'}
                </button>
                <button
                    type='button'
                    className='flow-button'
                    onClick={(event) => void runAction(event, 'complete-card')}
                    disabled={!cardId || busyAction !== '' || !checklistReadyForDone || cardAlreadyDone}
                >
                    {busyAction === 'complete-card' ? 'Finishing...' : !checklistReadyForDone ? 'Complete checklist first' : cardAlreadyDone ? 'Done' : hasDoneColumn && !inDoneColumn ? `Move to ${doneColumnName || 'Done'}` : 'Mark done'}
                </button>
            </div>
            {flowType === 'due_soon' && (
                <div className='flow-post__actions'>
                    <button
                        type='button'
                        className='flow-button'
                        onClick={(event) => void runAction(event, 'push-1d')}
                        disabled={!cardId || busyAction !== ''}
                    >
                        {busyAction === 'push-1d' ? 'Updating...' : 'Push +1 day'}
                    </button>
                    <button
                        type='button'
                        className='flow-button'
                        onClick={(event) => void runAction(event, 'push-7d')}
                        disabled={!cardId || busyAction !== ''}
                    >
                        {busyAction === 'push-7d' ? 'Updating...' : 'Push +1 week'}
                    </button>
                </div>
            )}
            <div className='flow-post__actions'>
                {openActions.filter((action) => action.url).map((action) => (
                    <button
                        key={action.label}
                        type='button'
                        className={`flow-button${action.primary ? ' flow-button--primary' : ''}`}
                        onClick={(event) => openLink(event, action.url)}
                    >
                        {action.label}
                    </button>
                ))}
            </div>
            {feedback && <div className='flow-post__feedback'>{feedback}</div>}
        </div>
    );
}
