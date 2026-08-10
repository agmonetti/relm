package store_test

import (
	"errors"
	"testing"

	"relm/internal/conn"
	"relm/internal/store"
	_ "relm/internal/store/sqlite" // registra el motor para el test
)

func TestNewUnsupportedDriver(t *testing.T) {
	cfg := conn.New(conn.Driver("oracle"))
	cfg.Path = ":memory:"

	_, err := store.New(cfg)
	if !errors.Is(err, store.ErrUnsupportedDriver) {
		t.Fatalf("err = %v, want ErrUnsupportedDriver", err)
	}
}

func TestNewSQLite(t *testing.T) {
	cfg := conn.New(conn.DriverSQLite)
	cfg.Path = ":memory:"

	st, err := store.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer st.Close()

	if st.Driver() != "sqlite" {
		t.Errorf("Driver() = %q, want sqlite", st.Driver())
	}
}
