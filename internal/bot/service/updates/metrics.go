package updates

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var promNotificationsProcessed = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "ads_mrkt_bot_notifications_processed_total",
		Help: "Total number of telegram notifications successfully sent",
	},
)

var promNotificationsFailed = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "ads_mrkt_bot_notifications_failed_total",
		Help: "Total number of telegram notifications that failed to send",
	},
)

var promNotificationAcksFailed = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "ads_mrkt_bot_notification_acks_failed_total",
		Help: "Total number of failed ACKs for telegram notification messages",
	},
)

var promUpdatesProcessed = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "ads_mrkt_bot_updates_processed_total",
		Help: "Total number of telegram update events successfully processed",
	},
)

var promUpdatesFailed = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "ads_mrkt_bot_updates_failed_total",
		Help: "Total number of telegram update events that failed processing",
	},
)
