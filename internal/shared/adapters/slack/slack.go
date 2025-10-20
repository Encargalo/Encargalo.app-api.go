package slack

import (
	"Encargalo.app-api.go/internal/shared/config"
	"github.com/slack-go/slack"
)

func NewConnectionSlack(config config.Config) *slack.Client {

	api := slack.New(config.Slack.Token)

	return api
}
