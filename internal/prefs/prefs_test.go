package prefs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	p := Default()
	if p.QueryTimeout() != QueryTimeoutDefault {
		t.Errorf("QueryTimeout() = %d, want %d", p.QueryTimeout(), QueryTimeoutDefault)
	}
}

func TestQueryTimeoutNormalizesInvalid(t *testing.T) {
	p := Prefs{QueryTimeoutSeconds: 0}
	if p.QueryTimeout() != QueryTimeoutDefault {
		t.Errorf("QueryTimeout() = %d, want default for 0", p.QueryTimeout())
	}
	p = Prefs{QueryTimeoutSeconds: -5}
	if p.QueryTimeout() != QueryTimeoutDefault {
		t.Errorf("QueryTimeout() = %d, want default for negative", p.QueryTimeout())
	}
}

func TestLoadMissingReturnsDefault(t *testing.T) {
	t.Setenv("RELM_CONFIG_DIR", t.TempDir())
	p, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if p.QueryTimeout() != QueryTimeoutDefault {
		t.Errorf("QueryTimeout = %d, want default", p.QueryTimeout())
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	t.Setenv("RELM_CONFIG_DIR", t.TempDir())

	p := Default()
	p.QueryTimeoutSeconds = 120
	if err := p.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.QueryTimeoutSeconds != 120 {
		t.Errorf("QueryTimeoutSeconds = %d, want 120", got.QueryTimeoutSeconds)
	}
}

func TestSaveWrites0600(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RELM_CONFIG_DIR", dir)

	p := Default()
	p.QueryTimeoutSeconds = 30
	if err := p.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "relm", "prefs.json"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("perm = %o, want 0600", perm)
	}
}
