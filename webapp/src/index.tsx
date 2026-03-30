import manifest from './manifest';
import React from 'react';
import type {Store} from 'redux';

import type {Post} from '@mattermost/types/posts';
import type {GlobalState} from '@mattermost/types/store';

import {FlowErrorBoundary} from './error_boundary';
import {FlowPage} from './flow_page';
import {FlowPost} from './flow_post';
import './styles.css';

import type {PluginRegistry} from 'types/mattermost-webapp';

function readContext(store: Store<GlobalState>) {
    const state: any = store.getState();
    const currentTeamId = state?.entities?.teams?.currentTeamId;
    const currentChannelId = state?.entities?.channels?.currentChannelId;
    const team = currentTeamId ? state?.entities?.teams?.teams?.[currentTeamId] : null;
    const channel = currentChannelId ? state?.entities?.channels?.channels?.[currentChannelId] : null;

    return {
        teamId: currentTeamId,
        teamName: team?.name,
        teamDisplayName: team?.display_name,
        channelId: currentChannelId,
        channelDisplayName: channel?.display_name,
    };
}

const FlowIcon = () => (
    <svg width='18' height='18' viewBox='0 0 18 18' fill='none'>
        <rect x='1.5' y='2' width='4.2' height='4.2' rx='1.1' fill='currentColor'/>
        <rect x='6.9' y='6.9' width='4.2' height='4.2' rx='1.1' fill='currentColor' opacity='0.85'/>
        <rect x='12.3' y='11.8' width='4.2' height='4.2' rx='1.1' fill='currentColor' opacity='0.7'/>
        <path d='M4.7 6.2L8.2 9.4L13.6 12.2' stroke='currentColor' strokeWidth='1.5' strokeLinecap='round'/>
    </svg>
);

export default class Plugin {
    public async initialize(registry: PluginRegistry, store: Store<GlobalState>) {
        const RouteComponent = () => <FlowErrorBoundary><FlowPage context={readContext(store)}/></FlowErrorBoundary>;
        const PostComponent = (props: {post: Post}) => {
            const state: any = store.getState();
            const currentUserId = state?.entities?.users?.currentUserId;
            return <FlowPost {...props} currentUserId={currentUserId}/>;
        };

        registry.registerNeedsTeamRoute('boards', RouteComponent);
        registry.registerPostTypeComponent('custom_mattermost_flow_update', PostComponent);
        registry.registerPostTypeComponent('custom_mattermost_flow_due_soon', PostComponent);
        registry.registerChannelHeaderButtonAction(
            <FlowIcon/>,
            () => {
                const context = readContext(store);
                if (!context.teamName) {
                    return;
                }
                const params = new URLSearchParams();
                if (context.channelId) {
                    params.set('channel_id', context.channelId);
                }
                const query = params.toString();
                const suffix = query ? `?${query}` : '';
                window.location.href = `/${context.teamName}/${manifest.id}/boards${suffix}`;
            },
            'Open Flow board',
            'Open Flow board',
        );
    }
}

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: Plugin): void;
    }
}

window.registerPlugin(manifest.id, new Plugin());
