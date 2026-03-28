import type {BoardSummary, FlowStreamEvent} from './types';

export type FlowSyncMessage = {
    type: 'board-updated';
    boardId: string;
    cardId?: string;
    reason: string;
    event?: FlowStreamEvent;
    summary?: BoardSummary;
    occurredAt: number;
};

const FLOW_SYNC_EVENT = 'mattermost-flow:sync';
const FLOW_SYNC_STORAGE_KEY = 'mattermost-flow:sync';
const FLOW_SYNC_CHANNEL = 'mattermost-flow-sync';

function createBroadcastChannel() {
    if (typeof window === 'undefined' || typeof window.BroadcastChannel === 'undefined') {
        return null;
    }

    return new window.BroadcastChannel(FLOW_SYNC_CHANNEL);
}

const broadcastChannel = createBroadcastChannel();

export function publishFlowSync(message: Omit<FlowSyncMessage, 'type' | 'occurredAt'>) {
    const payload: FlowSyncMessage = {
        type: 'board-updated',
        occurredAt: Date.now(),
        ...message,
    };

    window.dispatchEvent(new CustomEvent<FlowSyncMessage>(FLOW_SYNC_EVENT, {
        detail: payload,
    }));

    if (broadcastChannel) {
        broadcastChannel.postMessage(payload);
        return;
    }

    try {
        window.localStorage.setItem(FLOW_SYNC_STORAGE_KEY, JSON.stringify(payload));
        window.localStorage.removeItem(FLOW_SYNC_STORAGE_KEY);
    } catch {
        // Ignore sync persistence failures and keep local dispatch only.
    }
}

export function subscribeFlowSync(listener: (message: FlowSyncMessage) => void) {
    const handleWindowEvent = (event: Event) => {
        const customEvent = event as CustomEvent<FlowSyncMessage>;
        if (customEvent.detail) {
            listener(customEvent.detail);
        }
    };

    const handleStorageEvent = (event: StorageEvent) => {
        if (event.key !== FLOW_SYNC_STORAGE_KEY || !event.newValue) {
            return;
        }

        try {
            listener(JSON.parse(event.newValue) as FlowSyncMessage);
        } catch {
            // Ignore malformed payloads.
        }
    };

    const handleBroadcastMessage = (event: MessageEvent<FlowSyncMessage>) => {
        if (event.data) {
            listener(event.data);
        }
    };

    window.addEventListener(FLOW_SYNC_EVENT, handleWindowEvent as EventListener);
    window.addEventListener('storage', handleStorageEvent);
    broadcastChannel?.addEventListener('message', handleBroadcastMessage);

    return () => {
        window.removeEventListener(FLOW_SYNC_EVENT, handleWindowEvent as EventListener);
        window.removeEventListener('storage', handleStorageEvent);
        broadcastChannel?.removeEventListener('message', handleBroadcastMessage);
    };
}
