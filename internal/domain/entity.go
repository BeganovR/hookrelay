package domain

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrConflict     = errors.New("conflict")
	ErrInvalidInput = errors.New("invalid input")
)

const (
	RetryExponential = "exponential"
	RetryLinear      = "linear"
	RetryConstant    = "constant"
)

const (
	VerifierNoop = "noop"
	VerifierHMAC = "hmac"
)

const (
	AuthNone   = "none"
	AuthBearer = "bearer"
	AuthBasic  = "basic"
	AuthAPIKey = "api_key"
)

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type APIKey struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"project_id"`
	Name      string     `json:"name"`
	KeyHash   string     `json:"-"`
	Prefix    string     `json:"prefix"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	RawKey    string     `json:"key,omitempty"`
}

type Source struct {
	ID           string          `json:"id"`
	ProjectID    string          `json:"project_id"`
	Name         string          `json:"name"`
	UID          string          `json:"uid"`
	VerifierType string          `json:"verifier_type"`
	VerifierCfg  json.RawMessage `json:"verifier_cfg,omitempty"`
	IsActive     bool            `json:"is_active"`
	IngestURL    string          `json:"ingest_url,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type Endpoint struct {
	ID          string          `json:"id"`
	ProjectID   string          `json:"project_id"`
	Name        string          `json:"name"`
	URL         string          `json:"url"`
	Description string          `json:"description"`
	HTTPTimeout int             `json:"http_timeout"`
	AuthType    string          `json:"auth_type"`
	AuthCfg     json.RawMessage `json:"auth_cfg,omitempty"`
	IsActive    bool            `json:"is_active"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type Subscription struct {
	ID            string          `json:"id"`
	ProjectID     string          `json:"project_id"`
	Name          string          `json:"name"`
	SourceID      *string         `json:"source_id,omitempty"`
	EndpointID    string          `json:"endpoint_id"`
	MaxRetries    int             `json:"max_retries"`
	RetryInterval int             `json:"retry_interval"`
	RetryType     string          `json:"retry_type"`
	FilterCfg     json.RawMessage `json:"filter_cfg,omitempty"`
	IsActive      bool            `json:"is_active"`
	SigningSecret string          `json:"signing_secret"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type Event struct {
	ID             string          `json:"id"`
	ProjectID      string          `json:"project_id"`
	SourceID       *string         `json:"source_id,omitempty"`
	EventType      string          `json:"event_type"`
	Headers        json.RawMessage `json:"headers,omitempty"`
	Payload        json.RawMessage `json:"payload"`
	SenderIP       string          `json:"sender_ip"`
	IdempotencyKey *string         `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	IsNew          bool            `json:"-"`
}

type EventDelivery struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"project_id"`
	EventID        string    `json:"event_id"`
	EndpointID     string    `json:"endpoint_id"`
	SubscriptionID *string   `json:"subscription_id,omitempty"`
	Status         string    `json:"status"`
	RetryCount     int       `json:"retry_count"`
	MaxRetries     int       `json:"max_retries"`
	RetryInterval  int       `json:"retry_interval"`
	RetryType      string    `json:"retry_type"`
	ScheduledAt    time.Time `json:"scheduled_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type DeliveryAttempt struct {
	ID           string    `json:"id"`
	DeliveryID   string    `json:"delivery_id"`
	StatusCode   *int      `json:"status_code,omitempty"`
	ResponseBody *string   `json:"response_body,omitempty"`
	Error        *string   `json:"error,omitempty"`
	DurationMs   *int      `json:"duration_ms,omitempty"`
	AttemptedAt  time.Time `json:"attempted_at"`
}

type PendingDelivery struct {
	EventDelivery
	EventPayload     json.RawMessage
	EventHeaders     json.RawMessage
	EndpointURL      string
	EndpointTimeout  int
	EndpointAuthType string
	EndpointAuthCfg  json.RawMessage
	SigningSecret    string
}

type HMACVerifierConfig struct {
	Header   string `json:"header"`
	Prefix   string `json:"prefix"`
	Secret   string `json:"secret"`
	Encoding string `json:"encoding"`
}

type FilterConfig struct {
	EventTypes []string `json:"event_types"`
}

type BearerAuthConfig struct {
	Token string `json:"token"`
}

type BasicAuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type APIKeyAuthConfig struct {
	Header string `json:"header"`
	Value  string `json:"value"`
}
