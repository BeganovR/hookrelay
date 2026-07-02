package filter_test

import (
	"testing"

	"hookrelay/internal/domain"
	"hookrelay/internal/filter"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *domain.FilterConfig
		eventType string
		want      bool
	}{
		{"nil filter matches anything", nil, "order.created", true},
		{"empty filter matches anything", &domain.FilterConfig{}, "order.created", true},
		{"exact match", &domain.FilterConfig{EventTypes: []string{"order.created"}}, "order.created", true},
		{"no match", &domain.FilterConfig{EventTypes: []string{"order.updated"}}, "order.created", false},
		{"wildcard matches anything", &domain.FilterConfig{EventTypes: []string{"*"}}, "anything", true},
		{"multiple types, first matches", &domain.FilterConfig{EventTypes: []string{"order.created", "order.updated"}}, "order.created", true},
		{"multiple types, second matches", &domain.FilterConfig{EventTypes: []string{"order.created", "order.updated"}}, "order.updated", true},
		{"multiple types, none match", &domain.FilterConfig{EventTypes: []string{"order.created", "order.updated"}}, "order.deleted", false},
		{"empty event type with no filter", nil, "", true},
		{"filter with empty event types slice", &domain.FilterConfig{EventTypes: []string{}}, "order.created", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := filter.Match(tc.cfg, tc.eventType)
			if got != tc.want {
				t.Errorf("Match(%v, %q) = %v, want %v", tc.cfg, tc.eventType, got, tc.want)
			}
		})
	}
}
