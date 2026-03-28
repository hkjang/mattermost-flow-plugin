import React, {useDeferredValue, useEffect, useMemo, useState} from 'react';

import {flowClient} from './client';
import type {FlowUser} from './types';

type UserPickerProps = {
    boardId: string;
    selectedIds: string[];
    knownUsers: Record<string, FlowUser>;
    onUsersLoaded: (users: FlowUser[]) => void;
    onChange: (userIds: string[]) => void;
    placeholder?: string;
    emptyText?: string;
    disabled?: boolean;
};

export function UserPicker({
    boardId,
    selectedIds,
    knownUsers,
    onUsersLoaded,
    onChange,
    placeholder = 'Search teammates',
    emptyText = 'No assignees selected.',
    disabled,
}: UserPickerProps) {
    const [query, setQuery] = useState('');
    const [results, setResults] = useState<FlowUser[]>([]);
    const [loading, setLoading] = useState(false);
    const deferredQuery = useDeferredValue(query);
    const selectedKey = selectedIds.join(',');

    useEffect(() => {
        if (!boardId || selectedIds.length === 0) {
            return;
        }

        let active = true;
        void flowClient.searchBoardUsers(boardId, {
            ids: selectedIds,
            limit: Math.max(selectedIds.length, 8),
        }).then((response) => {
            if (active) {
                onUsersLoaded(response.items);
            }
        }).catch(() => undefined);

        return () => {
            active = false;
        };
    }, [boardId, selectedKey, onUsersLoaded]);

    useEffect(() => {
        if (!boardId) {
            setResults([]);
            return;
        }

        let active = true;
        setLoading(true);

        void flowClient.searchBoardUsers(boardId, {
            term: deferredQuery.trim(),
            limit: 8,
        }).then((response) => {
            if (!active) {
                return;
            }
            setResults(response.items);
            onUsersLoaded(response.items);
        }).catch(() => {
            if (active) {
                setResults([]);
            }
        }).finally(() => {
            if (active) {
                setLoading(false);
            }
        });

        return () => {
            active = false;
        };
    }, [boardId, deferredQuery, onUsersLoaded]);

    const selectedUsers = useMemo(() => {
        return selectedIds.map((id) => knownUsers[id] || {
            id,
            username: id,
            display_name: id,
        });
    }, [knownUsers, selectedKey]);

    const selectableResults = useMemo(() => {
        return results.filter((user) => !selectedIds.includes(user.id));
    }, [results, selectedKey]);

    function addUser(userId: string) {
        if (selectedIds.includes(userId)) {
            return;
        }
        onChange([...selectedIds, userId]);
        setQuery('');
    }

    function removeUser(userId: string) {
        onChange(selectedIds.filter((id) => id !== userId));
    }

    return (
        <div className='flow-user-picker'>
            <div className='flow-user-picker__selected'>
                {selectedUsers.length > 0 ? selectedUsers.map((user) => (
                    <span className='flow-user-picker__chip' key={user.id}>
                        <span>{formatUserLabel(user)}</span>
                        <button type='button' onClick={() => removeUser(user.id)} disabled={disabled}>Remove</button>
                    </span>
                )) : <div className='flow-user-picker__empty'>{emptyText}</div>}
            </div>

            <input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder={placeholder}
                disabled={disabled}
            />

            <div className='flow-user-picker__results'>
                {loading && <div className='flow-user-picker__empty'>Loading people...</div>}
                {!loading && selectableResults.map((user) => (
                    <button
                        key={user.id}
                        className='flow-user-picker__result'
                        type='button'
                        onClick={() => addUser(user.id)}
                        disabled={disabled}
                    >
                        <strong>{user.display_name}</strong>
                        <span>@{user.username}</span>
                    </button>
                ))}
                {!loading && selectableResults.length === 0 && (
                    <div className='flow-user-picker__empty'>
                        {deferredQuery.trim() ? 'No matching teammates.' : 'Start typing or pick from the current board scope.'}
                    </div>
                )}
            </div>
        </div>
    );
}

function formatUserLabel(user: FlowUser) {
    if (user.display_name && user.display_name !== user.username) {
        return `${user.display_name} (@${user.username})`;
    }
    return `@${user.username}`;
}
