package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	EventsIngested = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "hookrelay_events_ingested_total",
		Help: "Total number of events ingested, by source and type",
	}, []string{"source_uid", "event_type"})

	DeliveriesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "hookrelay_deliveries_total",
		Help: "Delivery outcomes by status (success|discarded|retry)",
	}, []string{"status"})

	DeliveryDurationSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "hookrelay_delivery_duration_seconds",
		Help:    "End-to-end HTTP delivery latency",
		Buckets: prometheus.DefBuckets,
	})

	WorkerClaimCycles = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hookrelay_worker_claim_cycles_total",
		Help: "Number of times the worker polled the DB for pending deliveries",
	})

	WorkerClaimedDeliveries = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hookrelay_worker_claimed_deliveries_total",
		Help: "Total deliveries claimed and dispatched to goroutines",
	})

	RecoveredDeliveries = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hookrelay_recovered_deliveries_total",
		Help: "Stuck processing deliveries reset to pending",
	})
)
