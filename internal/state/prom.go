package state

import "github.com/prometheus/client_golang/prometheus"

const namespace = "gbans"

var messagesReceived = prometheus.NewDesc(
	prometheus.BuildFQName(namespace, "", "bans_total"),
	"How many messages have been received (per channel).",
	[]string{"channel"}, nil,
)