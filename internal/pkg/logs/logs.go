package logs

import (
	"fmt"

	"github.com/slack-go/slack"
)

type Logs interface {
	Slack(err error)
}

type logs struct {
	slack *slack.Client
}

func NewLogs(slack *slack.Client) Logs {
	return &logs{
		slack,
	}
}

func (l *logs) Slack(err error) {

	_, _, error := l.slack.PostMessage("C09ME9F6KTQ", slack.MsgOptionText(err.Error(), false))
	if error != nil {
		fmt.Println(error)
	}

}
