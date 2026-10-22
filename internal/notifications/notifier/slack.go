package notifier

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"Encargalo.app-api.go/internal/notifications/domain/models"
	"Encargalo.app-api.go/internal/shared/config"
	"github.com/AndresXLP/ravenTree"
)

const maxAttachmentsPerSlackMessage = 10

type Slack struct {
	cfg config.Slack
	env string
}

type slackPayload struct {
	Text        string            `json:"text,omitempty"`
	Attachments []slackAttachment `json:"attachments"`
}

type slackAttachment struct {
	Color     string       `json:"color,omitempty"`
	Title     string       `json:"title,omitempty"`
	TitleLink string       `json:"title_link,omitempty"`
	Fields    []slackField `json:"fields,omitempty"`
	Footer    string       `json:"footer,omitempty"`
}

type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func NewSlack(cfg config.NotificationsConfig) *Slack {
	return &Slack{
		cfg: cfg.Slack,
		env: cfg.Environment.Current,
	}
}

func (s *Slack) Name() string {
	return "slack"
}

func (s *Slack) Send(ctx context.Context, msgs []models.NotificationMessage) error {
	if len(msgs) == 0 {
		return nil
	}

	groups := make(map[string][]models.NotificationMessage)
	for _, msg := range msgs {
		channelID, err := s.webhookID(msg)
		if err != nil {
			return err
		}
		groups[channelID] = append(groups[channelID], msg)
	}

	for channelID, group := range groups {
		for start := 0; start < len(group); start += maxAttachmentsPerSlackMessage {
			end := min(start+maxAttachmentsPerSlackMessage, len(group))
			if err := s.post(ctx, channelID, group[start:end]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Slack) webhookID(msg models.NotificationMessage) (string, error) {
	if !models.IsProductionDeployment(s.env) {
		return s.developChannel()
	}

	switch msg.Kind {
	case models.KindGrafanaFiring, models.KindGrafanaResolved:
		return s.grafanaAlertChannel(msg.AlertEnvironment)
	}

	var channelID string
	switch msg.Kind {
	case models.KindOrderCreated, models.KindOrderCompleted, models.KindTripCreated:
		channelID = s.cfg.OrdersStatusID
	case models.KindOrderRejected:
		channelID = s.cfg.OrdersRejectedID
	case models.KindOrderPendingUnassigned, models.KindOrderNoRider:
		channelID = s.cfg.OrdersPendingUnassignedID
	case models.KindOrderAutoRejected, models.KindTripNoDriver:
		channelID = s.cfg.OrdersAutoRejectedID
		if channelID == "" {
			channelID = s.cfg.OrdersPendingUnassignedID
		}
	case models.KindPaymentProofPending:
		channelID = s.cfg.PaymentProofsID
		if channelID == "" {
			channelID = s.cfg.OrdersStatusID
		}
	case models.KindBalanceRefundPending:
		channelID = s.cfg.BalanceRefundsID
	}

	if channelID == "" {
		return "", fmt.Errorf("slack channel id not found for kind %s", msg.Kind)
	}
	return channelID, nil
}

func (s *Slack) developChannel() (string, error) {
	if s.cfg.DevelopChannelID == "" {
		return "", errors.New("slack develop channel id not found")
	}
	return s.cfg.DevelopChannelID, nil
}

func (s *Slack) grafanaAlertChannel(deploymentEnv string) (string, error) {
	if models.IsProductionDeployment(deploymentEnv) {
		if s.cfg.ErrorsID == "" {
			return "", errors.New("slack errors channel id not found")
		}
		return s.cfg.ErrorsID, nil
	}
	return s.developChannel()
}

func (s *Slack) post(ctx context.Context, channelID string, msgs []models.NotificationMessage) error {
	attachments := make([]slackAttachment, 0, len(msgs))
	for _, msg := range msgs {
		attachments = append(attachments, toSlackAttachment(msg))
	}

	payload := slackPayload{
		Attachments: attachments,
	}

	tree := ravenTree.NewRavensTree()
	options := &ravenTree.Options{
		Host:       s.cfg.Url,
		Path:       s.cfg.Path + channelID,
		Method:     http.MethodPost,
		Body:       payload,
		Timeout:    5 * time.Second,
		RetryCount: 3,
	}

	resp, err := tree.SendRaven(ctx, options)
	if err != nil {
		return err
	}
	slog.Debug("slack webhook response", "body", resp.ParseBodyToString())
	return nil
}

func toSlackAttachment(msg models.NotificationMessage) slackAttachment {
	fields := make([]slackField, len(msg.Fields))
	for i, field := range msg.Fields {
		fields[i] = slackField{
			Title: field.Name,
			Value: field.Value,
			Short: field.Inline,
		}
	}
	return slackAttachment{
		Color:     fmt.Sprintf("#%06x", msg.Color),
		Title:     msg.Title,
		TitleLink: msg.URL,
		Fields:    fields,
		Footer:    msg.Footer,
	}
}
