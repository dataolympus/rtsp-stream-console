package stream

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidName = errors.New("stream name is required")
	ErrInvalidURL  = errors.New("valid RTSP URL is required")
)

type Service struct {
	registry Registry
}

func NewService(registry Registry) *Service {
	return &Service{
		registry: registry,
	}
}

func (s *Service) Create(name, rawURL string) (Stream, error) {
	name = strings.TrimSpace(name)
	rawURL = strings.TrimSpace(rawURL)

	if name == "" {
		return Stream{}, ErrInvalidName
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil ||
		parsedURL.Scheme != "rtsp" ||
		parsedURL.Host == "" {
		return Stream{}, ErrInvalidURL
	}

	created := Stream{
		ID:        uuid.NewString(),
		Name:      name,
		URL:       rawURL,
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.registry.Create(created); err != nil {
		return Stream{}, err
	}

	return created, nil
}

func (s *Service) Get(id string) (Stream, error) {
	return s.registry.Get(id)
}

func (s *Service) List() []Stream {
	return s.registry.List()
}

func (s *Service) Delete(id string) error {
	return s.registry.Delete(id)
}
