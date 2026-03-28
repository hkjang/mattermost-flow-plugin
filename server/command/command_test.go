package command

import (
	"reflect"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type env struct {
	client *pluginapi.Client
	api    *plugintest.API
}

type stubProvider struct {
	openURL       string
	openErr       error
	createMessage string
	createErr     error
	receivedTitle string
	receivedDue   string
}

func (p *stubProvider) OpenBoardURL(args *model.CommandArgs) (string, error) {
	return p.openURL, p.openErr
}

func (p *stubProvider) CreateCard(args *model.CommandArgs, title, dueDate string) (string, error) {
	p.receivedTitle = title
	p.receivedDue = dueDate
	return p.createMessage, p.createErr
}

func setupTest() *env {
	api := &plugintest.API{}
	driver := &plugintest.Driver{}
	client := pluginapi.NewClient(api, driver)

	return &env{
		client: client,
		api:    api,
	}
}

func expectFlowCommandRegistration(t *testing.T, api *plugintest.API) {
	t.Helper()

	api.On("RegisterCommand", mock.MatchedBy(func(command *model.Command) bool {
		return command.Trigger == flowCommandTrigger &&
			command.AutoComplete &&
			command.AutoCompleteDesc == "Open Mattermost Flow boards and create cards" &&
			command.AutoCompleteHint == "[open|new|help]" &&
			reflect.DeepEqual(command.AutocompleteData, buildAutocompleteData())
	})).Return(nil).Once()
}

func TestFlowOpenCommand(t *testing.T) {
	assert := assert.New(t)
	env := setupTest()
	provider := &stubProvider{openURL: "https://example.com/team/com.mattermost.flow-plugin/boards"}

	expectFlowCommandRegistration(t, env.api)
	cmdHandler := NewCommandHandler(env.client, provider)

	response, err := cmdHandler.Handle(&model.CommandArgs{Command: "/flow open"})
	assert.NoError(err)
	assert.Equal(model.CommandResponseTypeEphemeral, response.ResponseType)
	assert.Equal("Opening Mattermost Flow board.", response.Text)
	assert.Equal(provider.openURL, response.GotoLocation)
	env.api.AssertExpectations(t)
}

func TestFlowNewCommand(t *testing.T) {
	assert := assert.New(t)
	env := setupTest()
	provider := &stubProvider{createMessage: "Created **Ship release** in **Release / Todo**."}

	expectFlowCommandRegistration(t, env.api)
	cmdHandler := NewCommandHandler(env.client, provider)

	response, err := cmdHandler.Handle(&model.CommandArgs{Command: "/flow new Ship release --due 2026-04-02"})
	assert.NoError(err)
	assert.Equal(model.CommandResponseTypeEphemeral, response.ResponseType)
	assert.Equal(provider.createMessage, response.Text)
	assert.Equal("Ship release", provider.receivedTitle)
	assert.Equal("2026-04-02", provider.receivedDue)
	env.api.AssertExpectations(t)
}

func TestFlowHelpResponse(t *testing.T) {
	assert := assert.New(t)
	env := setupTest()
	provider := &stubProvider{}

	expectFlowCommandRegistration(t, env.api)
	cmdHandler := NewCommandHandler(env.client, provider)

	response, err := cmdHandler.Handle(&model.CommandArgs{Command: "/flow"})
	assert.NoError(err)
	assert.Contains(response.Text, "/flow open")
	assert.Contains(response.Text, "/flow new <title> [--due YYYY-MM-DD]")
	env.api.AssertExpectations(t)
}
