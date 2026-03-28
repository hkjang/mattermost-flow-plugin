package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

const flowCommandTrigger = "flow"

type Provider interface {
	OpenBoardURL(args *model.CommandArgs) (string, error)
	CreateCard(args *model.CommandArgs, title, dueDate string) (string, error)
}

type Handler struct {
	client   *pluginapi.Client
	provider Provider
}

type Command interface {
	Handle(args *model.CommandArgs) (*model.CommandResponse, error)
}

func NewCommandHandler(client *pluginapi.Client, provider Provider) Command {
	command := &model.Command{
		Trigger:          flowCommandTrigger,
		AutoComplete:     true,
		AutoCompleteDesc: "Open Mattermost Flow boards and create cards",
		AutoCompleteHint: "[open|new|help]",
		AutocompleteData: buildAutocompleteData(),
	}

	if err := client.SlashCommand.Register(command); err != nil {
		client.Log.Error("Failed to register flow command", "error", err)
	}

	return &Handler{
		client:   client,
		provider: provider,
	}
}

func (c *Handler) Handle(args *model.CommandArgs) (*model.CommandResponse, error) {
	fields := strings.Fields(args.Command)
	if len(fields) == 0 {
		return ephemeralResponse("Empty command."), nil
	}

	trigger := strings.TrimPrefix(fields[0], "/")
	if trigger != flowCommandTrigger {
		return ephemeralResponse(fmt.Sprintf("Unknown command: %s", args.Command)), nil
	}

	if len(fields) == 1 {
		return c.helpResponse(), nil
	}

	subcommand := strings.ToLower(strings.TrimSpace(fields[1]))
	switch subcommand {
	case "open":
		return c.executeOpenCommand(args)
	case "new":
		return c.executeNewCommand(args)
	case "help":
		return c.helpResponse(), nil
	default:
		return ephemeralResponse("Supported commands: /flow open, /flow new <title> [--due YYYY-MM-DD], /flow help"), nil
	}
}

func (c *Handler) executeOpenCommand(args *model.CommandArgs) (*model.CommandResponse, error) {
	url, err := c.provider.OpenBoardURL(args)
	if err != nil {
		return ephemeralResponse(err.Error()), nil
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		GotoLocation: url,
		Text:         "Opening Mattermost Flow board.",
	}, nil
}

func (c *Handler) executeNewCommand(args *model.CommandArgs) (*model.CommandResponse, error) {
	fields := strings.Fields(args.Command)
	if len(fields) < 3 {
		return ephemeralResponse("Usage: /flow new <title> [--due YYYY-MM-DD]"), nil
	}

	titleParts := make([]string, 0, len(fields)-2)
	dueDate := ""
	for index := 2; index < len(fields); index++ {
		if fields[index] == "--due" {
			if index+1 >= len(fields) {
				return ephemeralResponse("Usage: /flow new <title> [--due YYYY-MM-DD]"), nil
			}
			dueDate = strings.TrimSpace(fields[index+1])
			if _, err := time.Parse("2006-01-02", dueDate); err != nil {
				return ephemeralResponse("The due date must use YYYY-MM-DD format."), nil
			}
			index++
			continue
		}
		titleParts = append(titleParts, fields[index])
	}

	title := strings.TrimSpace(strings.Join(titleParts, " "))
	if title == "" {
		return ephemeralResponse("Card title is required."), nil
	}

	message, err := c.provider.CreateCard(args, title, dueDate)
	if err != nil {
		return ephemeralResponse(err.Error()), nil
	}

	return ephemeralResponse(message), nil
}

func (c *Handler) helpResponse() *model.CommandResponse {
	return ephemeralResponse(strings.Join([]string{
		"Mattermost Flow commands:",
		"/flow open - open the board page for the current team or channel",
		"/flow new <title> [--due YYYY-MM-DD] - create a card in the current board scope",
		"/flow help - show this help",
	}, "\n"))
}

func ephemeralResponse(text string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         text,
	}
}

func buildAutocompleteData() *model.AutocompleteData {
	top := model.NewAutocompleteData(flowCommandTrigger, "[command]", "Open boards and create cards")

	open := model.NewAutocompleteData("open", "", "Open Mattermost Flow in the current team or channel")
	top.AddCommand(open)

	create := model.NewAutocompleteData("new", "[title]", "Create a new card in the current board scope")
	create.AddTextArgument("Card title", "[title]", "")
	create.AddNamedTextArgument("--due", "Optional due date", "[YYYY-MM-DD]", "", false)
	top.AddCommand(create)

	help := model.NewAutocompleteData("help", "", "Show Mattermost Flow command help")
	top.AddCommand(help)

	return top
}
