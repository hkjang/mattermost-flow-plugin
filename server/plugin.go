package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
	"github.com/pkg/errors"

	"github.com/hkjang/mattermost-flow-plugin/server/command"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// client is the Mattermost server API client.
	client *pluginapi.Client

	botUserID string

	// commandClient is the client used to register and execute slash commands.
	commandClient command.Command

	// router is the HTTP router for handling API requests.
	router *mux.Router

	// service contains the domain logic for boards, cards, gantt data, activity and preferences.
	service     *FlowService
	eventBroker *boardEventBroker

	backgroundJob *cluster.Job

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

// OnActivate is invoked when the plugin is activated. If an error is returned, the plugin will be deactivated.
func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)
	p.service = NewFlowService(newKVStore(p.API))
	p.eventBroker = newBoardEventBroker()

	botUserID, err := p.client.Bot.EnsureBot(&model.Bot{
		Username:    FlowBotUsername,
		DisplayName: "Mattermost Flow",
		Description: "Posts workflow notifications for Mattermost Flow boards.",
	})
	if err != nil {
		return errors.Wrap(err, "failed to ensure flow bot")
	}
	p.botUserID = botUserID

	p.commandClient = command.NewCommandHandler(p.client, &commandProvider{plugin: p})

	p.router = p.initRouter()

	job, err := cluster.Schedule(
		p.API,
		"BackgroundJob",
		cluster.MakeWaitForRoundedInterval(1*time.Hour),
		p.runJob,
	)
	if err != nil {
		return errors.Wrap(err, "failed to schedule background job")
	}

	p.backgroundJob = job

	return nil
}

// OnDeactivate is invoked when the plugin is deactivated.
func (p *Plugin) OnDeactivate() error {
	if p.backgroundJob != nil {
		if err := p.backgroundJob.Close(); err != nil {
			p.API.LogError("Failed to close background job", "err", err)
		}
	}
	if p.eventBroker != nil {
		p.eventBroker.Close()
	}
	return nil
}

// This will execute the commands that were registered in the NewCommandHandler function.
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	response, err := p.commandClient.Handle(args)
	if err != nil {
		return nil, model.NewAppError("ExecuteCommand", "plugin.command.execute_command.app_error", nil, err.Error(), http.StatusInternalServerError)
	}
	return response, nil
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
