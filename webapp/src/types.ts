export type BoardSettings = {
    post_updates: boolean;
    post_due_soon: boolean;
    allow_mentions: boolean;
    calendar_feed_enabled: boolean;
    default_view: 'board' | 'gantt' | 'dashboard';
};

export type Board = {
    id: string;
    team_id?: string;
    channel_id?: string;
    name: string;
    description: string;
    visibility: 'team' | 'channel';
    admin_ids: string[];
    settings: BoardSettings;
    created_by: string;
    created_at: number;
    updated_at: number;
    version: number;
};

export type BoardColumn = {
    id: string;
    board_id: string;
    name: string;
    sort_order: number;
    wip_limit: number;
};

export type ChecklistItem = {
    id: string;
    text: string;
    completed: boolean;
};

export type AttachmentLink = {
    id: string;
    title: string;
    url: string;
};

export type CardComment = {
    id: string;
    card_id: string;
    user_id: string;
    message: string;
    created_at: number;
};

export type CardTemplate = {
    id: string;
    board_id: string;
    name: string;
    title: string;
    description: string;
    labels: string[];
    priority: Card['priority'];
    start_offset_days?: number;
    due_offset_days?: number;
    milestone: boolean;
    checklist: ChecklistItem[];
    attachment_links: AttachmentLink[];
    created_by: string;
    created_at: number;
    updated_at: number;
};

export type Card = {
    id: string;
    board_id: string;
    column_id: string;
    title: string;
    description: string;
    assignee_ids: string[];
    labels: string[];
    priority: 'low' | 'normal' | 'high' | 'urgent';
    start_date?: string;
    due_date?: string;
    progress: number;
    milestone: boolean;
    checklist: ChecklistItem[];
    attachment_links: AttachmentLink[];
    comments: CardComment[];
    position: number;
    created_by: string;
    updated_by: string;
    created_at: number;
    updated_at: number;
    version: number;
};

export type Dependency = {
    id: string;
    board_id: string;
    source_card_id: string;
    target_card_id: string;
    type: string;
    created_by: string;
    created_at: number;
};

export type Activity = {
    id: string;
    board_id: string;
    entity_type: string;
    entity_id: string;
    action: string;
    actor_id: string;
    before?: unknown;
    after?: unknown;
    created_at: number;
};

export type BoardFilters = {
    query: string;
    assignee_id: string;
    label: string;
    status: string;
    date_from: string;
    date_to: string;
};

export type Preference = {
    user_id: string;
    board_id: string;
    view_type: 'board' | 'gantt' | 'dashboard';
    filters: BoardFilters;
    zoom_level: 'day' | 'week' | 'month';
    updated_at: number;
};

export type FlowUser = {
    id: string;
    username: string;
    display_name: string;
};

export type BoardSummary = {
    board: Board;
    card_count: number;
    overdue_count: number;
    due_soon_count: number;
    default_board: boolean;
    columns: number;
    assignees: string[];
    recent_activity?: Activity;
};

export type BoardCalendarFeedInfo = {
    enabled: boolean;
    has_token: boolean;
    download_url: string;
    subscribe_url?: string;
    updated_at?: number;
};

export type BoardDiagnosticsSummary = {
    columns: number;
    cards: number;
    templates: number;
    dependencies: number;
    activities: number;
    comments: number;
    milestones: number;
    scheduled_cards: number;
    overdue_cards: number;
    invalid_dates: number;
};

export type BoardDiagnosticsIssue = {
    code: string;
    severity: 'warning' | 'error';
    title: string;
    detail: string;
    entity_ids?: string[];
    entity_type?: string;
    count: number;
};

export type BoardDiagnosticsReport = {
    board_id: string;
    generated_at: number;
    summary: BoardDiagnosticsSummary;
    issues: BoardDiagnosticsIssue[];
    healthy: boolean;
    repair_available: boolean;
};

export type BoardBundle = {
    board: Board;
    columns: BoardColumn[];
    templates: CardTemplate[];
    cards: Card[];
    dependencies: Dependency[];
    activity: Activity[];
    preference: Preference;
    summary: BoardSummary;
};

export type GanttViewData = {
    board: Board;
    columns: BoardColumn[];
    tasks: Card[];
    dependencies: Dependency[];
};

export type CardMutationResult = {
    board: Board;
    card: Card;
    column_name: string;
};

export type CardMoveResult = {
    board: Board;
    card: Card;
    from_column_name: string;
    to_column_name: string;
};

export type DependencyMutationResult = {
    board: Board;
    dependency: Dependency;
};

export type CommentMutationResult = {
    board: Board;
    card: Card;
    comment: CardComment;
};

export type BoardColumnInput = {
    id?: string;
    name: string;
    sort_order: number;
    wip_limit: number;
};

export type CardTemplateInput = {
    id?: string;
    name: string;
    title: string;
    description: string;
    labels: string[];
    priority: Card['priority'];
    start_offset_days?: number;
    due_offset_days?: number;
    milestone: boolean;
    checklist: ChecklistItem[];
    attachment_links: AttachmentLink[];
};

export type CreateBoardRequest = {
    team_id?: string;
    channel_id?: string;
    name: string;
    description: string;
    visibility: 'team' | 'channel';
    admin_ids: string[];
    columns: BoardColumnInput[];
    templates?: CardTemplateInput[];
    settings: BoardSettings;
    set_as_default: boolean;
};

export type UpdateBoardRequest = Partial<{
    name: string;
    description: string;
    admin_ids: string[];
    columns: BoardColumnInput[];
    templates: CardTemplateInput[];
    settings: BoardSettings;
    version: number;
}>;

export type CreateCardRequest = {
    board_id: string;
    column_id: string;
    title: string;
    description: string;
    assignee_ids: string[];
    labels: string[];
    priority: Card['priority'];
    start_date?: string;
    due_date?: string;
    progress: number;
    milestone: boolean;
    checklist: ChecklistItem[];
    attachment_links: AttachmentLink[];
};

export type UpdateCardRequest = Partial<{
    title: string;
    description: string;
    assignee_ids: string[];
    labels: string[];
    priority: Card['priority'];
    start_date: string;
    due_date: string;
    progress: number;
    milestone: boolean;
    checklist: ChecklistItem[];
    attachment_links: AttachmentLink[];
    version: number;
}>;

export type MoveCardRequest = {
    target_column_id: string;
    target_index: number;
    version: number;
};

export type CardActionResponse = {
    action: string;
    event_action: string;
    status: 'applied' | 'noop';
    message: string;
    board_id: string;
    board: Board;
    summary: BoardSummary;
    card: Card;
    column_card_ids?: Record<string, string[]>;
    current_column_name: string;
    next_column_name?: string;
    has_next_column: boolean;
    done_column_name?: string;
    has_done_column: boolean;
    in_done_column: boolean;
};

export type FlowStreamEvent = {
    type: string;
    board_id: string;
    entity_type: string;
    action: string;
    actor_id?: string;
    card_id?: string;
    occurred_at: number;
    column_card_ids?: Record<string, string[]>;
    board?: Board;
    card?: Card;
    dependency?: Dependency;
    comment?: CardComment;
    activity?: Activity;
};

export type FlowBoardSummaryEvent = {
    type: string;
    board_id: string;
    action: string;
    occurred_at: number;
    summary?: BoardSummary;
};

export type ContextSnapshot = {
    teamId?: string;
    teamName?: string;
    teamDisplayName?: string;
    channelId?: string;
    channelDisplayName?: string;
};
