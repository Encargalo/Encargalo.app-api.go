package models

import "strings"

type NotificationKind string

const (
	KindOrderCreated           NotificationKind = "order_created"
	KindOrderCompleted         NotificationKind = "order_completed"
	KindOrderRejected          NotificationKind = "order_rejected"
	KindOrderPendingUnassigned NotificationKind = "order_pending_unassigned"
	KindOrderAutoRejected      NotificationKind = "order_auto_rejected"
	KindPaymentProofPending    NotificationKind = "payment_proof_pending"
	KindBalanceRefundPending   NotificationKind = "balance_refund_pending"
	KindWithdrawalRequested    NotificationKind = "withdrawal_requested"
	KindOrderNoRider           NotificationKind = "order_no_rider"
	KindTripCreated            NotificationKind = "trip_created"
	KindTripNoDriver           NotificationKind = "trip_no_driver"
	KindGrafanaFiring          NotificationKind = "grafana_firing"
	KindGrafanaResolved        NotificationKind = "grafana_resolved"
)

type NotificationField struct {
	Name   string
	Value  string
	Inline bool
}

// NotificationMessage is the channel-agnostic payload that destinations map
// to Discord embeds, Slack attachments, etc.
type NotificationMessage struct {
	Kind   NotificationKind
	Title  string
	URL    string
	Color  int
	Fields []NotificationField
	Footer string
	// AlertEnvironment es el deployment_environment_name de alertas Grafana.
	// Define si van al canal de errores de prod o al de dev.
	AlertEnvironment string
}

func IsProductionDeployment(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "production", "produccion", "prod":
		return true
	default:
		return false
	}
}
