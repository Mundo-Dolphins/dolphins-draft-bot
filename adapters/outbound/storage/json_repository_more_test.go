package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/domain"
)

func TestJSONRepositoryListAndInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	repo := NewJSONEventRepository(path)
	list, err := repo.List(context.Background())
	if err != nil || len(list) != 0 {
		t.Fatalf("expected empty list, err=%v", err)
	}
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.List(context.Background()); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestJSONRepositoryWritesPrivateFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	path := filepath.Join(dir, "state.json")
	repo := NewJSONEventRepository(path)
	if err := repo.Save(context.Background(), domain.PublishedEvent{Key: "selection:1", Type: domain.EventSelection, Overall: 1}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("state file mode = %v, want 0600", fi.Mode().Perm())
	}
	di, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if di.Mode().Perm() != 0o700 {
		t.Fatalf("state dir mode = %v, want 0700", di.Mode().Perm())
	}
}
