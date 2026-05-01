package platforms

import (
	"testing"
)

func TestRegisterAndGet(t *testing.T) {
	// Save and restore registry
	saved := registry
	registry = map[string]func(configJSON string) (Platform, error){}
	defer func() { registry = saved }()

	Register("test", func(configJSON string) (Platform, error) {
		return &testPlatform{name: configJSON}, nil
	})

	p, err := Get("test", "my-config")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if p.Name() != "my-config" {
		t.Errorf("Name() = %q, want %q", p.Name(), "my-config")
	}
}

func TestGet_Unknown(t *testing.T) {
	_, err := Get("nonexistent", "")
	if err == nil {
		t.Fatal("expected error for unknown platform, got nil")
	}
}

func TestList(t *testing.T) {
	// Save and restore registry
	saved := registry
	registry = map[string]func(configJSON string) (Platform, error){}
	defer func() { registry = saved }()

	Register("a", func(string) (Platform, error) { return &testPlatform{}, nil })
	Register("b", func(string) (Platform, error) { return &testPlatform{}, nil })

	names := List()
	if len(names) != 2 {
		t.Errorf("List() returned %d names, want 2", len(names))
	}
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["a"] || !found["b"] {
		t.Errorf("List() = %v, missing expected entries", names)
	}
}

type testPlatform struct {
	name string
}

func (t *testPlatform) Name() string {
	if t.name == "" {
		return "test"
	}
	return t.name
}

func (t *testPlatform) Upsert(key, value string) error {
	return nil
}
