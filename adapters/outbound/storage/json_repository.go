package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/domain"
)

type JSONEventRepository struct {
	path string
	mu   sync.Mutex
}

type stateFile struct {
	Events []domain.PublishedEvent `json:"events"`
}

func NewJSONEventRepository(path string) *JSONEventRepository {
	return &JSONEventRepository{path: path}
}

func (r *JSONEventRepository) Has(_ context.Context, key domain.EventKey) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	events, err := r.read()
	if err != nil {
		return false, err
	}
	for _, ev := range events {
		if ev.Key == key {
			return true, nil
		}
	}
	return false, nil
}

func (r *JSONEventRepository) Save(_ context.Context, event domain.PublishedEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	events, err := r.read()
	if err != nil {
		return err
	}
	for _, ev := range events {
		if ev.Key == event.Key {
			return nil
		}
	}
	events = append(events, event)
	return r.write(events)
}

func (r *JSONEventRepository) List(_ context.Context) ([]domain.PublishedEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.read()
}

func (r *JSONEventRepository) read() ([]domain.PublishedEvent, error) {
	if _, err := os.Stat(r.path); os.IsNotExist(err) {
		return []domain.PublishedEvent{}, nil
	}
	raw, err := os.ReadFile(r.path)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return []domain.PublishedEvent{}, nil
	}
	var st stateFile
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, err
	}
	return st.Events, nil
}

func (r *JSONEventRepository) write(events []domain.PublishedEvent) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(stateFile{Events: events}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, b, 0o600)
}
