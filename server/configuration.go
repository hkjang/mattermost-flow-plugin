package main

import (
	"reflect"

	"github.com/pkg/errors"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.
type configuration struct {
	// MaxBoardsPerChannel is the maximum number of boards that can be created per channel.
	// A value of 0 means unlimited.
	MaxBoardsPerChannel int `json:"MaxBoardsPerChannel"`

	// MaxCardsPerBoard is the maximum number of cards allowed per board.
	// A value of 0 means unlimited.
	MaxCardsPerBoard int `json:"MaxCardsPerBoard"`

	// DueSoonHours is how many hours before a due date to send "due soon" notifications.
	DueSoonHours int `json:"DueSoonHours"`

	// EnableCalendarFeed enables or disables calendar feed (iCal) functionality.
	EnableCalendarFeed bool `json:"EnableCalendarFeed"`

	// EnableBoardExportImport enables or disables board export/import functionality.
	EnableBoardExportImport bool `json:"EnableBoardExportImport"`

	// DefaultBoardView is the default view when opening a board (board, gantt, dashboard).
	DefaultBoardView string `json:"DefaultBoardView"`

	// BackgroundJobIntervalMinutes is the interval for running background tasks.
	BackgroundJobIntervalMinutes int `json:"BackgroundJobIntervalMinutes"`
}

// setDefaults fills zero-value fields with sensible defaults that match plugin.json.
func (c *configuration) setDefaults() {
	if c.DueSoonHours <= 0 {
		c.DueSoonHours = 48
	}
	if c.DefaultBoardView == "" {
		c.DefaultBoardView = "board"
	}
	if c.BackgroundJobIntervalMinutes < 5 {
		c.BackgroundJobIntervalMinutes = 60
	}
}

// Clone shallow copies the configuration. Your implementation may require a deep copy if
// your configuration has reference types.
func (c *configuration) Clone() *configuration {
	clone := *c
	return &clone
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		cfg := &configuration{}
		cfg.setDefaults()
		return cfg
	}

	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		// Ignore assignment if the configuration struct is empty. Go will optimize the
		// allocation for same to point at the same memory address, breaking the check
		// above.
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	configuration := new(configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	configuration.setDefaults()
	p.setConfiguration(configuration)

	return nil
}
