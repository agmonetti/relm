package cassandra

import (
	"context"
	"testing"

	"github.com/agmonetti/relm/internal/conn"
	"github.com/agmonetti/relm/internal/store"
)

func TestCassandra_Registered(t *testing.T) {
	cfg := conn.New(conn.DriverCassandra)
	cfg.Host = "127.0.0.1"
	cfg.Port = 65532 // unopened port

	_, err := store.New(cfg)
	if err == nil {
		t.Fatal("expected connection error on unopened port")
	}
	if err.Error() == "unsupported driver: cassandra" {
		t.Fatal("DriverCassandra not registered in store registry")
	}
}

func TestCassandra_ExecutorIsMutationAndReadOnly(t *testing.T) {
	source := &CassandraSource{readOnly: true}
	exec := &CassandraExecutor{source: source}

	if !exec.IsMutation("INSERT INTO users (id, name) VALUES (1, 'Alice')") {
		t.Error("INSERT must be detected as mutation")
	}
	if !exec.IsMutation("UPDATE users SET name = 'Bob' WHERE id = 1") {
		t.Error("UPDATE must be detected as mutation")
	}
	if !exec.IsMutation("DELETE FROM users WHERE id = 1") {
		t.Error("DELETE must be detected as mutation")
	}
	if !exec.IsMutation("DROP TABLE users") {
		t.Error("DROP must be detected as mutation")
	}
	if !exec.IsMutation("TRUNCATE users") {
		t.Error("TRUNCATE must be detected as mutation")
	}
	if exec.IsMutation("SELECT * FROM users") {
		t.Error("SELECT must not be detected as mutation")
	}

	_, err := exec.Execute(context.Background(), "INSERT INTO users (id) VALUES (1)", 0, 10)
	if err == nil {
		t.Fatal("expected mutation to be blocked in read-only mode")
	}
}

func TestCassandra_LanguageAndTitle(t *testing.T) {
	exec := &CassandraExecutor{}
	if exec.Language() != "CQL" {
		t.Errorf("Language = %q, want CQL", exec.Language())
	}
	if exec.PromptTitle() != "CQL QUERY" {
		t.Errorf("PromptTitle = %q, want CQL QUERY", exec.PromptTitle())
	}
}
