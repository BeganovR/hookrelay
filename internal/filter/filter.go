package filter

import "hookrelay/internal/domain"

func Match(cfg *domain.FilterConfig, eventType string) bool {
	if cfg == nil || len(cfg.EventTypes) == 0 {
		return true
	}
	for _, t := range cfg.EventTypes {
		if t == eventType || t == "*" {
			return true
		}
	}
	return false
}
