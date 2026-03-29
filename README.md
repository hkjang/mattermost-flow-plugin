# Mattermost Flow Plugin

[![Build Status](https://github.com/hkjang/mattermost-flow-plugin/actions/workflows/ci.yml/badge.svg)](https://github.com/hkjang/mattermost-flow-plugin/actions/workflows/ci.yml)

Mattermost Flow Plugin brings kanban boards and gantt timelines into Mattermost so teams can plan, track, and finish work without leaving channel context.

Live site: [hkjang.github.io/mattermost-flow-plugin](https://hkjang.github.io/mattermost-flow-plugin/)

## Highlights

- Team-scoped and channel-scoped boards with default columns: `Todo`, `In Progress`, `Review`, `Done`
- Board and gantt views in one plugin, including drag-and-drop card movement and gantt date resizing
- Rich cards with assignees, labels, priority, dates, progress, milestone flag, checklist, comments, links, and dependencies
- Board-scoped card templates for reusable release, bugfix, handoff, and milestone workflows
- Mattermost-native collaboration with slash commands, channel header entry point, channel posts, mentions, deep links, and quick post actions
- Plugin API, KV store persistence, and server-sent events for live board and sidebar updates

## Quick Start

### Install from a GitHub Release

1. Download the latest `com.mattermost.flow-plugin-<version>.tar.gz` from [Releases](https://github.com/hkjang/mattermost-flow-plugin/releases).
2. In Mattermost System Console, upload the plugin bundle from `System Console -> Plugin Management`.
3. Enable the plugin.
4. Open a channel and use the channel header button labeled `Open Flow board`, or run `/flow open`.

### Build Locally

Prerequisites:

- Go `1.25+`
- Node.js `24.13.1` from [.nvmrc](./.nvmrc)

Build the distributable plugin bundle:

```bash
make dist
```

The generated bundle will be written to:

```text
dist/com.mattermost.flow-plugin-<version>.tar.gz
```

## How Users Access Flow

- Channel header button: `Open Flow board`
- Slash command: `/flow open`
- Direct route: `/{team-name}/com.mattermost.flow-plugin/boards`
- Shared board, gantt, and card deep links from the UI or Flow posts

## Slash Commands

```text
/flow open
/flow new <title> [--due YYYY-MM-DD]
/flow help
```

## Documentation

- [User Guide](./docs/USER_GUIDE.md)
- [Admin Guide](./docs/ADMIN_GUIDE.md)
- [Development Guide](./docs/DEVELOPMENT_GUIDE.md)
- [Release Guide](./docs/RELEASE_GUIDE.md)
- [Promo Site](https://hkjang.github.io/mattermost-flow-plugin/)

## Architecture Summary

- Web app: React + TypeScript plugin UI for board, gantt, filters, post UI, and real-time updates
- Server plugin: Go service layer, custom API under `/plugins/com.mattermost.flow-plugin/api/v1`, authorization, KV storage, jobs, and Mattermost integration
- Storage: Mattermost plugin KV store for boards, columns, cards, dependencies, activity logs, preferences, and due-soon notifications
- Reuse: board-scoped card templates saved with default labels, checklist, links, milestone state, and relative dates
- Collaboration: Mattermost posts, mentions, post quick actions, slash commands, deep links, and SSE streams

## Development

Common local workflows:

```bash
make
make test
make dist
make deploy
make watch
```

For local deployment, the plugin can be uploaded through Mattermost or deployed with `make deploy` once local mode or admin credentials are configured. See the [Admin Guide](./docs/ADMIN_GUIDE.md) and [Development Guide](./docs/DEVELOPMENT_GUIDE.md) for details.

## Release Notes

Release bundles are created automatically when a `v*` tag is pushed. The release workflow builds the plugin, publishes `SHA256SUMS.txt`, and uploads both assets to GitHub Releases.

The bundle packager also normalizes server binary permissions to `0755` inside the generated `.tar.gz`, so uploaded plugin archives are safe to run on Mattermost hosts without a post-extract chmod step.

## License

See [LICENSE](./LICENSE).
