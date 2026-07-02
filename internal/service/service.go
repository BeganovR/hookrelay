package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"hookrelay/internal/config"
	"hookrelay/internal/domain"
	"hookrelay/internal/metrics"
)

type workerStore interface {
	ClaimPendingDeliveries(ctx context.Context, batchSize int) ([]domain.PendingDelivery, error)
	MarkDeliverySuccess(ctx context.Context, id string) error
	MarkDeliveryDiscarded(ctx context.Context, id string) error
	ScheduleDeliveryRetry(ctx context.Context, id string, scheduledAt time.Time) error
	RecoverStuckDeliveries(ctx context.Context, stuckBefore time.Time) (int64, error)
	CreateDeliveryAttempt(ctx context.Context, deliveryID string, statusCode *int, durationMs *int, responseBody *string, errMsg *string) error
}

type Worker struct {
	store  workerStore
	cfg    config.Worker
	client *http.Client
	wg     sync.WaitGroup
}

func NewWorker(store workerStore, cfg config.Worker) *Worker {
	return &Worker{
		store: store,
		cfg:   cfg,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (w *Worker) Run(ctx context.Context) {
	pollTicker := time.NewTicker(w.cfg.PollInterval)
	recoveryTicker := time.NewTicker(30 * time.Second)
	defer pollTicker.Stop()
	defer recoveryTicker.Stop()

	slog.Info("worker started",
		"concurrency", w.cfg.Concurrency,
		"poll_interval", w.cfg.PollInterval,
		"batch_size", w.cfg.BatchSize,
	)

	sem := make(chan struct{}, w.cfg.Concurrency)

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker draining in-flight deliveries")
			w.wg.Wait()
			slog.Info("worker stopped")
			return

		case <-recoveryTicker.C:
			stuckBefore := time.Now().Add(-w.cfg.StuckTimeout)
			if n, err := w.store.RecoverStuckDeliveries(context.Background(), stuckBefore); err != nil {
				slog.Error("recovery failed", "error", err)
			} else if n > 0 {
				slog.Info("recovered stuck deliveries", "count", n)
				metrics.RecoveredDeliveries.Add(float64(n))
			}

		case <-pollTicker.C:
			metrics.WorkerClaimCycles.Inc()
			deliveries, err := w.store.ClaimPendingDeliveries(ctx, w.cfg.BatchSize)
			if err != nil {
				slog.Error("claim failed", "error", err)
				continue
			}
			metrics.WorkerClaimedDeliveries.Add(float64(len(deliveries)))
			for _, pd := range deliveries {
				pd := pd
				sem <- struct{}{}
				w.wg.Add(1)
				go func() {
					defer w.wg.Done()
					defer func() { <-sem }()
					defer func() {
						if r := recover(); r != nil {
							slog.Error("delivery panicked", "id", pd.ID, "panic", r)
						}
					}()
					w.deliver(context.Background(), pd)
				}()
			}
		}
	}
}

func (w *Worker) deliver(ctx context.Context, pd domain.PendingDelivery) {
	start := time.Now()

	timeout := time.Duration(pd.EndpointTimeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, pd.EndpointURL, bytes.NewReader(pd.EventPayload))
	if err != nil {
		slog.Error("build request failed", "id", pd.ID, "error", err)
		w.recordFailure(ctx, pd, nil, nil, err.Error())
		return
	}

	ts := fmt.Sprintf("%d", time.Now().Unix())

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-HookRelay-Delivery", pd.ID)
	req.Header.Set("X-HookRelay-Timestamp", ts)

	if pd.SigningSecret != "" {
		toSign := pd.ID + "." + ts + "." + string(pd.EventPayload)
		mac := hmac.New(sha256.New, []byte(pd.SigningSecret))
		mac.Write([]byte(toSign))
		req.Header.Set("X-HookRelay-Signature", "v1,"+base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	}

	w.applyAuth(req, pd.EndpointAuthType, pd.EndpointAuthCfg)

	resp, err := w.client.Do(req)
	durationMs := int(time.Since(start).Milliseconds())

	metrics.DeliveryDurationSeconds.Observe(time.Since(start).Seconds())

	if err != nil {
		slog.Warn("delivery http error", "id", pd.ID, "attempt", pd.RetryCount+1, "error", err)
		w.recordFailure(ctx, pd, nil, &durationMs, err.Error())
		return
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	bodyStr := string(body)
	code := resp.StatusCode

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		slog.Info("delivery success", "id", pd.ID, "status", resp.StatusCode)
		metrics.DeliveriesTotal.WithLabelValues("success").Inc()
		if err := w.store.MarkDeliverySuccess(ctx, pd.ID); err != nil {
			slog.Error("mark success failed", "id", pd.ID, "error", err)
		}
		if err := w.store.CreateDeliveryAttempt(ctx, pd.ID, &code, &durationMs, &bodyStr, nil); err != nil {
			slog.Error("create delivery attempt failed", "id", pd.ID, "error", err)
		}
		return
	}

	slog.Warn("delivery non-2xx", "id", pd.ID, "status", resp.StatusCode, "attempt", pd.RetryCount+1)
	msg := fmt.Sprintf("non-2xx status: %d", resp.StatusCode)
	w.recordFailure(ctx, pd, &code, &durationMs, msg)
}

func (w *Worker) recordFailure(ctx context.Context, pd domain.PendingDelivery, statusCode *int, durationMs *int, errMsg string) {
	if err := w.store.CreateDeliveryAttempt(ctx, pd.ID, statusCode, durationMs, nil, &errMsg); err != nil {
		slog.Error("create delivery attempt failed", "id", pd.ID, "error", err)
	}

	if pd.RetryCount+1 >= pd.MaxRetries {
		slog.Warn("delivery permanently failed", "id", pd.ID, "attempts", pd.RetryCount+1)
		metrics.DeliveriesTotal.WithLabelValues("discarded").Inc()
		if err := w.store.MarkDeliveryDiscarded(ctx, pd.ID); err != nil {
			slog.Error("mark discarded failed", "id", pd.ID, "error", err)
		}
		return
	}

	metrics.DeliveriesTotal.WithLabelValues("retry").Inc()
	delay := CalcBackoff(pd.RetryType, pd.RetryInterval, pd.RetryCount)
	next := time.Now().Add(delay)
	slog.Info("scheduling retry", "id", pd.ID, "attempt", pd.RetryCount+1, "next", next)
	if err := w.store.ScheduleDeliveryRetry(ctx, pd.ID, next); err != nil {
		slog.Error("schedule retry failed", "id", pd.ID, "error", err)
	}
}

func (w *Worker) applyAuth(req *http.Request, authType string, authCfgJSON []byte) {
	switch authType {
	case domain.AuthBearer:
		var cfg domain.BearerAuthConfig
		if json.Unmarshal(authCfgJSON, &cfg) == nil && cfg.Token != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.Token)
		}
	case domain.AuthBasic:
		var cfg domain.BasicAuthConfig
		if json.Unmarshal(authCfgJSON, &cfg) == nil {
			req.SetBasicAuth(cfg.Username, cfg.Password)
		}
	case domain.AuthAPIKey:
		var cfg domain.APIKeyAuthConfig
		if json.Unmarshal(authCfgJSON, &cfg) == nil && cfg.Header != "" {
			req.Header.Set(cfg.Header, cfg.Value)
		}
	}
}

func CalcBackoff(retryType string, baseInterval, attempt int) time.Duration {
	base := float64(baseInterval)
	switch retryType {
	case domain.RetryLinear:
		return time.Duration(base*float64(attempt+1)) * time.Second
	case domain.RetryConstant:
		return time.Duration(base) * time.Second
	default:
		exp := math.Pow(2, float64(attempt)) * base
		jitter := (rand.Float64()*0.2 - 0.1) * exp
		seconds := exp + jitter
		if seconds > 3600 {
			seconds = 3600
		}
		return time.Duration(seconds) * time.Second
	}
}
