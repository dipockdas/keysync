package store

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestFallbackStore_SetAndGet(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	fp := filepath.Join(dir, "store.json")

	s, err := newFallbackStore(fp)
	if err != nil {
		t.Fatalf("newFallbackStore failed: %v", err)
	}

	if err := s.Set(ctx, ScopeGlobal, "", "", "TEST_KEY", "test-val"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := s.Get(ctx, ScopeGlobal, "", "", "TEST_KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "test-val" {
		t.Errorf("got %q, want %q", val, "test-val")
	}
}

func TestFallbackStore_Persistence(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	fp := filepath.Join(dir, "store.json")

	// Write
	s1, err := newFallbackStore(fp)
	if err != nil {
		t.Fatalf("newFallbackStore failed: %v", err)
	}
	if err := s1.Set(ctx, ScopeGlobal, "", "", "PERSIST_KEY", "persist-val"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		t.Fatal("store file was not created")
	}

	// Re-open from same path
	s2, err := newFallbackStore(fp)
	if err != nil {
		t.Fatalf("newFallbackStore failed: %v", err)
	}

	val, err := s2.Get(ctx, ScopeGlobal, "", "", "PERSIST_KEY")
	if err != nil {
		t.Fatalf("Get from reopened store failed: %v", err)
	}
	if val != "persist-val" {
		t.Errorf("got %q, want %q", val, "persist-val")
	}
}

func TestFallbackStore_DeleteNotFound(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	fp := filepath.Join(dir, "store.json")

	s, err := newFallbackStore(fp)
	if err != nil {
		t.Fatalf("newFallbackStore failed: %v", err)
	}

	err = s.Delete(ctx, ScopeGlobal, "", "", "NONEXISTENT")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFallbackStore_GetNotFound(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	fp := filepath.Join(dir, "store.json")

	s, err := newFallbackStore(fp)
	if err != nil {
		t.Fatalf("newFallbackStore failed: %v", err)
	}

	_, err = s.Get(ctx, ScopeGlobal, "", "", "NONEXISTENT")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFallbackStore_List(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	fp := filepath.Join(dir, "store.json")

	s, err := newFallbackStore(fp)
	if err != nil {
		t.Fatalf("newFallbackStore failed: %v", err)
	}

	s.Set(ctx, ScopeGlobal, "", "", "G_KEY", "gv")
	s.Set(ctx, ScopeProject, "app", "", "P_KEY", "pv")

	entries, err := s.List(ctx, "", "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}

	// Filter by project
	entries, err = s.List(ctx, ScopeProject, "app", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestFallbackStore_Concurrency(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	fp := filepath.Join(dir, "store.json")

	s, err := newFallbackStore(fp)
	if err != nil {
		t.Fatalf("newFallbackStore failed: %v", err)
	}

	// Concurrent writes
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		s.Set(ctx, ScopeGlobal, "", "", "CONCUR_A", "a")
	}()
	go func() {
		defer wg.Done()
		s.Set(ctx, ScopeGlobal, "", "", "CONCUR_B", "b")
	}()
	go func() {
		defer wg.Done()
		s.Get(ctx, ScopeGlobal, "", "", "CONCUR_A")
	}()
	wg.Wait()

	val, err := s.Get(ctx, ScopeGlobal, "", "", "CONCUR_A")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "a" {
		t.Errorf("got %q, want %q", val, "a")
	}
}

func TestFallbackStore_Delete(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	fp := filepath.Join(dir, "store.json")

	s, err := newFallbackStore(fp)
	if err != nil {
		t.Fatalf("newFallbackStore failed: %v", err)
	}

	s.Set(ctx, ScopeGlobal, "", "", "DEL_ME", "val")
	if err := s.Delete(ctx, ScopeGlobal, "", "", "DEL_ME"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify file still exists (but key is gone)
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		t.Fatal("store file should still exist after delete")
	}

	_, err = s.Get(ctx, ScopeGlobal, "", "", "DEL_ME")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
