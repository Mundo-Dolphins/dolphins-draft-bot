package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/domain"
)

func TestJSONRepositoryPersistAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	repo := NewJSONEventRepository(path)
	e := domain.PublishedEvent{Key: "selection:11", Type: domain.EventSelection, Overall: 11, CreatedAt: time.Unix(0, 0)}
	if err := repo.Save(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	reloaded := NewJSONEventRepository(path)
	has, err := reloaded.Has(context.Background(), "selection:11")
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("expected persisted key")
	}
}
