# User Guide

This guide is for Mattermost users who want to manage work inside Flow boards and gantt views.

## Opening Flow

You can open the plugin in any of these ways:

- Click the channel header button `Open Flow board`
- Run `/flow open`
- Open a shared deep link to a board, gantt view, or card

When Flow opens from a channel, it uses the current team and channel context automatically.

## Creating and Selecting Boards

Flow supports both team-scoped and channel-scoped boards.

- Team board: shared across the whole team
- Channel board: tied to one channel and ideal for project or delivery streams

If a channel has a default board, `/flow open` and quick-create flows will use it first.

## Board View

The board view is the kanban-style working surface.

Typical actions:

- Create a new board with default columns
- Add cards into a target column
- Drag cards between columns
- Reorder cards inside the same column
- Filter by assignee, label, status, and date range
- Copy a shareable board link

Cards can include:

- Title and description
- Assignees from Mattermost users in the board scope
- Labels and priority
- Start date and due date
- Progress and milestone state
- Checklist items
- Attachment links
- Comments

## Card Templates

Board admins can save reusable card templates from board settings.

Templates can prefill:

- Card title and description
- Labels and priority
- Relative start and due dates
- Milestone state
- Checklist items
- Attachment links

In the quick-create bar, pick a template first and then adjust the title, due date, assignees, or priority before creating the card.

## Gantt View

The gantt view uses the same cards and dates as the board view.

What you can do:

- Switch between board and gantt without leaving the board
- View scheduled work across a timeline
- Drag the center of a task bar to move a schedule
- Drag the `Start` and `End` handles to resize the task
- Open a card directly from the gantt row
- Copy a shareable gantt link

## Card Detail

Selecting a card opens its detail panel.

From the card detail you can:

- Update metadata and dates
- Assign or remove users
- Edit checklist items
- Add comments
- Add dependency links between cards
- Review activity history
- Copy a deep link to the card

## Collaboration Inside Mattermost

Flow posts updates directly into the connected channel when board settings allow it.

Examples:

- Card created
- Card moved
- Due-soon reminder

Flow posts can also expose quick actions such as:

- `Assign to me`
- `Move to next`
- `Push +1 day`
- `Push +1 week`
- `Complete next item`
- `Mark done`
- `Open card`
- `Open gantt`

These actions update the board and related open Flow views in real time.

## Slash Commands

Use the built-in slash command for quick access:

```text
/flow open
/flow new Ship release --due 2026-04-10
/flow help
```

`/flow new` creates a card in the current board scope. If the channel has a default board, that board is used first.

## Tips

- Use a channel-scoped board when work is closely tied to one channel conversation stream.
- Use a team-scoped board when multiple channels need the same planning surface.
- Use shared board and card links in posts when you want others to jump into the exact context.
- Use due dates on cards so the gantt view and due-soon notifications stay meaningful.

## Troubleshooting

- If `/flow open` says no board exists, create a board in that scope first.
- If you cannot find an assignee, confirm the user is a member of the relevant team or channel.
- If a post action updates the card but your screen looks stale, reopen the board link once. Flow normally syncs automatically, but a browser refresh is a quick fallback.
