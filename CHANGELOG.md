# Changelog

## v0.2.0

### Design Overhaul
- Replaced all custom CSS variables with Mattermost native CSS variables (`--center-channel-bg`, `--center-channel-color`, `--button-bg`, `--button-color`, `--error-text`, `--away-indicator`, `--online-indicator`, etc.)
- Changed border-radius from 18px to 4px (Mattermost standard)
- Removed glassmorphism effects, gradient backgrounds, and custom fonts
- Adopted Mattermost-style flat panel design with subtle box shadows
- Dashboard and post cards now use left-border color indicators instead of gradient backgrounds
- Kanban cards display priority-based left-border colors (urgent=red, high=orange, normal=blue, low=gray)

### Admin Settings (System Console)
- Added 7 configurable admin settings accessible from the Mattermost System Console:
  - **Max Boards per Channel** — limit how many boards can be created per channel (default: 10, 0=unlimited)
  - **Max Cards per Board** — limit cards per board (default: 500, 0=unlimited)
  - **Due Soon Notification Hours** — configurable notification horizon (default: 48h)
  - **Enable Calendar Feed** — toggle iCal feed functionality on/off
  - **Enable Board Export/Import** — toggle export/import feature on/off
  - **Default Board View** — set system-wide default view (Board/Gantt/Dashboard)
  - **Background Job Interval** — configurable background task interval (default: 60 min)
- Server enforces all limits at the API level
- Webapp hides disabled features (calendar, export/import) based on server config

### Card Deletion
- Added card deletion feature with `DELETE /api/v1/cards/{id}` endpoint
- Automatically cleans up related dependencies when a card is deleted
- Records deletion in the activity log
- Delete button added to card detail panel header

### Confirmation Dialog
- Replaced browser-native `window.confirm()` with a Mattermost-styled modal dialog
- Used for board deletion and card deletion
- Supports ESC key, backdrop click to cancel
- Danger mode with red confirmation button for destructive actions

### Keyboard Shortcuts
- `Escape` — close dialog, settings panel, or card detail (in priority order)
- `N` — focus the quick-create card input
- `1` / `2` / `3` — switch to Board / Gantt / Dashboard view
- `S` — toggle board settings panel

### Slash Command Expansion
- `/flow list` — show all boards in the current scope with card/overdue/due-soon counts
- `/flow status` — display a markdown status table for the default board (columns, cards, completion %, overdue, due soon)
- `/flow assign <keyword> @user` — find a card by title keyword and assign a user to it
- Updated autocomplete data for all new commands

### Error Boundary
- Added React Error Boundary component wrapping the main FlowPage
- Prevents full application crash on unexpected rendering errors
- Displays a recovery UI with error message and "Reload plugin" button
- Logs error details to browser console for debugging

### Mobile Responsive
- Sidebar hidden on screens <= 1200px with a hamburger toggle button
- Sidebar slides in as an overlay with backdrop shadow
- Added 768px breakpoint: single-column kanban, full-width filters/settings/dashboard metrics
- Toolbar wraps gracefully on narrow screens

### Bug Fixes & Null Safety
- Fixed null pointer risk in `handlePublicBoardCalendarICS` — added `p.service == nil` check (this endpoint is outside the auth middleware)
- Due-soon notification hours now read from admin config instead of hardcoded 48h
- Background job interval now configurable instead of hardcoded 1 hour

## v0.1.3
- Board export and import tools
- Board diagnostics and repair tools
- Dashboard and calendar integration
- Board card templates
