# Admin Guide

This guide is for Mattermost administrators and board owners who install, enable, and operate the Flow plugin.

## Requirements

- Mattermost server version `6.2.1` or later, based on the plugin manifest minimum version
- Plugin uploads enabled on the Mattermost server
- Permission to install and enable plugins

## Install the Plugin

### Option 1: Upload a Release Bundle

1. Download `com.mattermost.flow-plugin-<version>.tar.gz` from [GitHub Releases](https://github.com/hkjang/mattermost-flow-plugin/releases).
2. Open `System Console -> Plugin Management`.
3. Upload the `.tar.gz` bundle.
4. Enable the plugin after upload.

### Option 2: Install with `mmctl`

```bash
mmctl plugin add dist/com.mattermost.flow-plugin-<version>.tar.gz --local
mmctl plugin enable com.mattermost.flow-plugin
```

## Enable Plugin Uploads

If plugin uploads are disabled, enable them in Mattermost:

```json
{
  "PluginSettings": {
    "EnableUploads": true
  }
}
```

Apply the configuration change and restart Mattermost if your environment requires it.

## Permissions Model

Flow uses Mattermost membership and admin checks for scope access.

- Board viewers must belong to the board scope
- Board admins can modify board settings and board structure
- Team admins can administer boards in their team
- System admins can administer all boards

Board-level admins are stored as part of board metadata.

## Operational Behavior

### Storage

Flow stores operational data in the Mattermost plugin KV store:

- Board metadata and columns
- Cards and dependencies
- Activity history
- User preferences
- Channel default board mappings
- Due-soon notification state

No external database is required for the current plugin design.

### Notifications

Flow can post updates into the connected channel when board settings allow it.

Available board-level settings include:

- `post_updates`
- `post_due_soon`
- `allow_mentions`
- `default_view`

Due-soon scanning runs as a background cluster job on an hourly interval.

### Executable Permissions in Release Bundles

Release archives are packaged so that files under `server/dist/` are stored with executable mode `0755`. This avoids the common issue where Mattermost extracts a plugin bundle but the server binary is not runnable on Linux or macOS hosts.

## Recommended Rollout Pattern

1. Install the plugin in a staging Mattermost instance.
2. Create one team board and one channel board.
3. Verify board view, gantt view, slash commands, and channel posts.
4. Enable the plugin in production.
5. Share user guidance and preferred board conventions with team owners.

## Upgrade and Rollback

- Upgrade: upload a newer `.tar.gz` bundle or install the newer release through `mmctl`
- Rollback: re-upload a previous release bundle and re-enable it if needed

Because Flow stores data in KV, keep plugin versions reasonably close when rolling backward.

## Troubleshooting

### Plugin uploads fail

- Confirm `PluginSettings.EnableUploads` is enabled
- Confirm the uploaded file is the generated `.tar.gz`, not the repo source archive

### Users receive `Not authorized`

- Confirm the user is logged into Mattermost
- Confirm the user belongs to the board team or channel
- Confirm reverse proxies are not stripping Mattermost auth headers for plugin routes

### No due-soon posts appear

- Confirm the board is channel-scoped
- Confirm `post_due_soon` is enabled in board settings
- Confirm cards have a due date and are not already complete

### Quick actions do not mention assignees

- Confirm `allow_mentions` is enabled in board settings

## Related Documents

- [README](../README.md)
- [User Guide](./USER_GUIDE.md)
- [Development Guide](./DEVELOPMENT_GUIDE.md)
- [Release Guide](./RELEASE_GUIDE.md)
