package neo4j

import (
	"context"
	"testing"

	"github.com/agmonetti/relm/internal/conn"
	"github.com/agmonetti/relm/internal/store"
)

func TestNeo4j_Registered(t *testing.T) {
	cfg := conn.New(conn.DriverNeo4j)
	cfg.Host = "127.0.0.1"
	cfg.Port = 65531 // unopened port

	_, err := store.New(cfg)
	if err == nil {
		t.Fatal("expected connection error on unopened port")
	}
	if err.Error() == "unsupported driver: neo4j" {
		t.Fatal("DriverNeo4j not registered in store registry")
	}
}

func TestNeo4j_ExecutorIsMutationAndReadOnly(t *testing.T) {
	source := &Neo4jSource{readOnly: true}
	exec := &Neo4jExecutor{source: source}

	if !exec.IsMutation("CREATE (n:Person {name: 'Alice'})") {
		t.Error("CREATE must be detected as mutation")
	}
	if !exec.IsMutation("MATCH (n:Person) DETACH DELETE n") {
		t.Error("DETACH DELETE must be detected as mutation")
	}
	if !exec.IsMutation("MERGE (n:Person {id: 1})") {
		t.Error("MERGE must be detected as mutation")
	}
	if exec.IsMutation("MATCH (n:Person) RETURN n") {
		t.Error("MATCH RETURN must not be detected as mutation")
	}

	_, err := exec.Execute(context.Background(), "CREATE (n:Person)", 0, 10)
	if err == nil {
		t.Fatal("expected mutation to be blocked in read-only mode")
	}
}

func TestNeo4j_LanguageAndTitle(t *testing.T) {
	exec := &Neo4jExecutor{}
	if exec.Language() != "Cypher" {
		t.Errorf("Language = %q, want Cypher", exec.Language())
	}
	if exec.PromptTitle() != "CYPHER QUERY" {
		t.Errorf("PromptTitle = %q, want CYPHER QUERY", exec.PromptTitle())
	}
}
