package service

type WebhookRepository interface {
	SaveWebhook(id string, body []byte) error
}

type WebhookService struct {
	repo WebhookRepository
}

func NewWebhookService(newRepo WebhookRepository) *WebhookService {
	return &WebhookService{
		repo: newRepo,
	}
}

func (s *WebhookService) ProcessWebhook(id string, body []byte) error {
	//TODO: Implement Business-logic (null-check, etc)
	return s.repo.SaveWebhook(id, body)
}
