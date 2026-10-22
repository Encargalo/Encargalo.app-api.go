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

const maxEmbedsPerDiscordMessage = 10

type Discord struct {
	cfg config.Discord
	env string
}

func NewDiscord(cfg config.NotificationsConfig) *Discord {
	return &Discord{
		cfg: cfg.Discord,
		env: cfg.Environment.Current,
	}
}

func (d *Discord) Name() string {
	return "discord"
}

func (d *Discord) Send(ctx context.Context, msgs []models.NotificationMessage) error {
	if len(msgs) == 0 {
		return nil
	}

	groups := make(map[string][]models.NotificationMessage)
	for _, msg := range msgs {
		channelID, err := d.webhookID(msg)
		if err != nil {
			return err
		}
		groups[channelID] = append(groups[channelID], msg)
	}

	for channelID, group := range groups {
		for start := 0; start < len(group); start += maxEmbedsPerDiscordMessage {
			end := min(start+maxEmbedsPerDiscordMessage, len(group))
			if err := d.post(ctx, channelID, group[start:end]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *Discord) webhookID(msg models.NotificationMessage) (string, error) {
	if !models.IsProductionDeployment(d.env) {
		return d.developChannel()
	}

	switch msg.Kind {
	case models.KindGrafanaFiring, models.KindGrafanaResolved:
		return d.grafanaAlertChannel(msg.AlertEnvironment)
	}

	var channelID string
	switch msg.Kind {
	case models.KindOrderCreated, models.KindOrderCompleted:
		channelID = d.cfg.OrdersStatusID
	case models.KindOrderRejected:
		channelID = d.cfg.OrdersRejectedID
	case models.KindOrderPendingUnassigned:
		channelID = d.cfg.OrdersPendingUnassignedID
	case models.KindOrderAutoRejected:
		channelID = d.cfg.OrdersAutoRejectedID
		if channelID == "" {
			channelID = d.cfg.OrdersPendingUnassignedID
		}
	case models.KindPaymentProofPending:
		channelID = d.cfg.PaymentProofsID
		if channelID == "" {
			channelID = d.cfg.OrdersStatusID
		}
	case models.KindBalanceRefundPending:
		channelID = d.cfg.BalanceRefundsID
	}

	if channelID == "" {
		return "", fmt.Errorf("discord channel id not found for kind %s", msg.Kind)
	}
	return channelID, nil
}

func (d *Discord) developChannel() (string, error) {
	if d.cfg.DevelopChannelID == "" {
		return "", errors.New("develop channel id not found")
	}
	return d.cfg.DevelopChannelID, nil
}

func (d *Discord) grafanaAlertChannel(deploymentEnv string) (string, error) {
	if models.IsProductionDeployment(deploymentEnv) {
		if d.cfg.ErrorsID == "" {
			return "", errors.New("discord errors channel id not found")
		}
		return d.cfg.ErrorsID, nil
	}
	return d.developChannel()
}

func (d *Discord) post(ctx context.Context, channelID string, msgs []models.NotificationMessage) error {
	embeds := make([]models.DiscordEmbed, 0, len(msgs))
	for _, msg := range msgs {
		embeds = append(embeds, toDiscordEmbed(msg))
	}

	tree := ravenTree.NewRavensTree()
	options := &ravenTree.Options{
		Host:       d.cfg.Url,
		Path:       d.cfg.Path + channelID,
		Method:     http.MethodPost,
		Body:       models.DiscordPayload{Embeds: embeds},
		Timeout:    5 * time.Second,
		RetryCount: 3,
	}

	resp, err := tree.SendRaven(ctx, options)
	if err != nil {
		return err
	}
	slog.Debug("discord webhook response", "body", resp.ParseBodyToString())
	return nil
}

func toDiscordEmbed(msg models.NotificationMessage) models.DiscordEmbed {
	embed := models.DiscordEmbed{
		Title:  msg.Title,
		URL:    msg.URL,
		Color:  msg.Color,
		Fields: make([]models.EmbedField, len(msg.Fields)),
	}
	for i, field := range msg.Fields {
		embed.Fields[i] = models.EmbedField(field)
	}
	if msg.Footer != "" {
		embed.Footer = &models.EmbedFooter{Text: msg.Footer}
	}
	return embed
}
