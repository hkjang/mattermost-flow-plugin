# Development Guide

This guide is for contributors working on the Flow plugin codebase.

## Stack

- Server plugin: Go
- Web app plugin: React + TypeScript
- Router/API: `gorilla/mux`
- Persistence: Mattermost plugin KV store
- Real-time updates: server-sent events plus local client sync bridge

## Repository Layout

```text
assets/                  Plugin assets bundled into releases
build/                   Manifest, packaging, and plugin control helpers
docs/                    User, admin, development, and release documentation
server/                  Go plugin server code
server/command/          Slash command handler
server/store/            KV store wrappers
webapp/src/              React UI, Mattermost integrations, styles, tests
plugin.json              Mattermost plugin manifest
Makefile                 Local build, test, deploy, and release helpers
```

## Key Runtime Entry Points

### Server

- [server/plugin.go](../server/plugin.go): plugin activation, bot setup, slash command registration, background job scheduling
- [server/api.go](../server/api.go): custom plugin API under `/api/v1`
- [server/service.go](../server/service.go): business logic for boards, cards, activity, gantt, and summaries
- [server/store.go](../server/store.go): KV store contract and key handling
- [server/event_broker.go](../server/event_broker.go): SSE subscription broker

### Web App

- [webapp/src/index.tsx](../webapp/src/index.tsx): Mattermost plugin registration
- [webapp/src/flow_page.tsx](../webapp/src/flow_page.tsx): main board, dashboard, and gantt experience
- [webapp/src/flow_post.tsx](../webapp/src/flow_post.tsx): custom post rendering and quick actions
- [webapp/src/client.ts](../webapp/src/client.ts): plugin API client
- [webapp/src/flow_sync.ts](../webapp/src/flow_sync.ts): same-tab and cross-tab sync bridge

## Local Setup

Prerequisites:

- Go `1.25+`
- Node.js `24.13.1`
- A Mattermost instance with plugin uploads enabled

Install and build:

```bash
make
```

Useful targets:

```bash
make test
make dist
make deploy
make watch
make logs
```

## API Surface

The plugin exposes its server API under:

```text
/plugins/com.mattermost.flow-plugin/api/v1
```

Notable endpoints:

- `GET /boards`
- `POST /boards`
- `GET /boards/{id}`
- `PATCH /boards/{id}`
- `DELETE /boards/{id}`
- `GET /boards/{id}/calendar-feed`
- `POST /boards/{id}/calendar-feed/rotate`
- `GET /boards/{id}/diagnostics`
- `POST /boards/{id}/diagnostics/repair`
- `GET /boards/{id}/calendar.ics`
- `GET /boards/{id}/stream`
- `GET /boards/summary/stream`
- `GET /boards/{id}/cards`
- `GET /boards/{id}/gantt`
- `GET /boards/{id}/activity`
- `PUT /boards/{id}/preferences`
- `GET /boards/{id}/users`
- `POST /cards`
- `PATCH /cards/{id}`
- `POST /cards/{id}/move`
- `POST /cards/{id}/actions/{action}`
- `POST /cards/{id}/dependencies`
- `POST /cards/{id}/comments`

Requests rely on Mattermost authentication headers and board scope authorization checks.

In addition, the plugin exposes a tokenized public calendar route for external subscribers:

```text
/plugins/com.mattermost.flow-plugin/calendar/{boardId}.ics?token=...
```

## Data Model

Core entities:

- `Board`
- `BoardColumn`
- `CardTemplate`
- `Card`
- `Dependency`
- `Activity`
- `Preference`
- `DueSoonNotification`
- `BoardCalendarFeed`

Data is stored in the Mattermost plugin KV store. Templates are stored per board alongside columns and cards, and calendar feed tokens are stored per board for external `.ics` subscriptions. The code keeps board summaries, activity, and live updates close to the write path so sidebar and board views can patch themselves quickly without full refreshes.

## Collaboration Features to Keep in Mind

- Flow uses Mattermost posts for board updates and due-soon alerts
- Post quick actions mutate cards directly through the plugin API
- SSE updates keep open boards and board lists synchronized across users
- A background cluster job scans for due-soon cards every hour

When changing mutation flows, update both the server event publishing path and the client patching logic.

## Testing

Recommended checks before pushing:

```bash
go test ./server/... ./server/command ./server/store/...
cd webapp && npm run check-types
cd webapp && npm run test
```

For a full distributable verification:

```bash
make dist
```

## Documentation Expectations

If you change user-visible behavior, update:

- [README](../README.md)
- [User Guide](./USER_GUIDE.md)
- [Admin Guide](./ADMIN_GUIDE.md)
- [Release Guide](./RELEASE_GUIDE.md) when packaging or release behavior changes
