package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type commandOptionType int

const (
	optTypeSubCommand      commandOptionType = 1
	optTypeSubCommandGroup commandOptionType = 2
	optTypeString          commandOptionType = 3
	optTypeInteger         commandOptionType = 4
	optTypeBoolean         commandOptionType = 5
	optTypeUser            commandOptionType = 6
	optTypeChannel         commandOptionType = 7
	optTypeRole            commandOptionType = 8
)

type slashChoices struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type slashOptions struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        commandOptionType `json:"type"`
	Required    bool              `json:"required"`
	Choices     []slashChoices    `json:"choices,omitempty"`
}

type slashCommandDefinition struct {
	ID            string         `json:"id,omitempty"`
	ApplicationID string         `json:"application_id,omitempty"`
	Version       string         `json:"version,omitempty"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Options       []slashOptions `json:"options,omitempty"`
}

func botRegisterSlashCommands(appID int64, authToken string) error {
	var helpChoices []slashChoices
	for name := range cmdMap {
		helpChoices = append(helpChoices, slashChoices{
			Name:  name,
			Value: name,
		})
	}
	slashCommands := []slashCommandDefinition{
		{
			Name:        "help",
			Description: "Get help for a command",
			Options: []slashOptions{
				{Name: "command", Description: "Which command to lookup help for", Type: optTypeString, Required: false, Choices: helpChoices},
			},
		},
	}

	url := fmt.Sprintf("https://discord.com/api/v8/applications/%d/commands", appID)
	client := &http.Client{
		Timeout: time.Second * 10,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}
	for _, cmd := range slashCommands {
		b, err := json.Marshal(&cmd)
		if err != nil {
			return errors.Wrapf(err, "Failed to encode command: %s", cmd.Name)
		}
		req, err2 := http.NewRequest("POST", url, bytes.NewReader(b))
		if err2 != nil {
			return errors.Wrapf(err2, "Failed to create request command: %s", cmd.Name)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bot %s", authToken))
		req.Header.Set("Content-Type", "application/json")
		resp, err3 := client.Do(req)
		if err3 != nil {
			return errors.Wrapf(err3, "Failed to post request: %s", cmd.Name)
		}
		if resp.StatusCode == http.StatusCreated {
			log.Infof("Registered command successfully: %s", cmd.Name)
		} else if resp.StatusCode == http.StatusOK {
			log.Debugf("Registered duplicate command: %s", cmd.Name)
		} else {
			return errors.Wrapf(err3, "Failed to post request, invalid response code: %s (%d)", cmd.Name, resp.StatusCode)
		}
	}
	return nil
}
