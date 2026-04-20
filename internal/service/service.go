package service

import "hookrelay/internal/storage"

type WebhookService struct {
	repo *storage.Storage
}

func NewWebhookService(newRepo *storage.Storage) *WebhookService {
	return &WebhookService{
		repo: newRepo,
	}
}
