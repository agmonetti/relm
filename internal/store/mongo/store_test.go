package mongo

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/agmonetti/relm/internal/conn"
	"github.com/agmonetti/relm/internal/store"
)

func TestMongo_Registered(t *testing.T) {
	// Check that DriverMongo is registered in the factory registry
	// by attempting to create an invalid host connection (should fail on ping, not 'unknown driver')
	cfg := conn.New(conn.DriverMongo)
	cfg.Host = "127.0.0.1"
	cfg.Port = 65534 // unopened port

	_, err := store.New(cfg)
	if err == nil {
		t.Fatal("expected connection error on unopened port")
	}
	if err.Error() == "unsupported driver: mongo" {
		t.Fatal("DriverMongo not registered in store registry")
	}
}

func TestMongo_ExtractDocID(t *testing.T) {
	oid := primitive.NewObjectID()
	doc := bson.M{"_id": oid, "name": "Alice"}
	if id := extractDocID(doc); id != oid.Hex() {
		t.Errorf("extractDocID = %q, want %q", id, oid.Hex())
	}

	doc2 := bson.M{"_id": "custom-string-id", "num": 42}
	if id := extractDocID(doc2); id != "custom-string-id" {
		t.Errorf("extractDocID = %q, want custom-string-id", id)
	}

	doc3 := bson.M{"no_id": true}
	if id := extractDocID(doc3); id != "unknown" {
		t.Errorf("extractDocID = %q, want unknown", id)
	}
}

func TestMongo_BuildDocSummary(t *testing.T) {
	doc := bson.M{
		"_id":   primitive.NewObjectID(),
		"title": "Clean Code",
		"price": 29.99,
	}
	summary := buildDocSummary(doc)
	if summary == "" || summary == "{}" {
		t.Errorf("summary was empty for doc: %s", summary)
	}
}

func TestMongo_InferBSONType(t *testing.T) {
	tests := []struct {
		val  any
		want string
	}{
		{primitive.NewObjectID(), "objectId"},
		{"hello", "string"},
		{int32(10), "int"},
		{int64(100), "int"},
		{12.34, "double"},
		{true, "bool"},
		{bson.A{1, 2, 3}, "array"},
		{bson.M{"a": 1}, "object"},
		{nil, "null"},
	}

	for _, tt := range tests {
		got := inferBSONType(tt.val)
		if got != tt.want {
			t.Errorf("inferBSONType(%v) = %q, want %q", tt.val, got, tt.want)
		}
	}
}

func TestMongo_MQLPattern(t *testing.T) {
	matches := mqlCallPattern.FindStringSubmatch("db.users.find({ age: { $gt: 18 } })")
	if len(matches) != 4 {
		t.Fatalf("expected 4 match elements, got %d", len(matches))
	}
	if matches[1] != "users" {
		t.Errorf("collection = %q, want users", matches[1])
	}
	if matches[2] != "find" {
		t.Errorf("method = %q, want find", matches[2])
	}

	// Semicolon stripping
	matches2 := mqlCallPattern.FindStringSubmatch("db.orders.countDocuments();")
	if len(matches2) != 4 || matches2[1] != "orders" || matches2[2] != "countDocuments" {
		t.Fatalf("match with semicolon failed: %v", matches2)
	}
}

func TestMongo_ExecutorLanguageAndPrompt(t *testing.T) {
	exec := &MongoExecutor{}
	if exec.Language() != "MQL (BSON / JSON)" {
		t.Errorf("Language = %q", exec.Language())
	}
	if exec.PromptTitle() != "MONGO QUERY" {
		t.Errorf("PromptTitle = %q", exec.PromptTitle())
	}
	if !exec.IsMutation("db.users.insertOne({name: 'Alice'})") {
		t.Error("insertOne must be detected as mutation")
	}
	if !exec.IsMutation("db.users.deleteOne({id: 1})") {
		t.Error("deleteOne must be detected as mutation")
	}
	if exec.IsMutation("db.users.find({})") {
		t.Error("find must not be detected as mutation")
	}
}

func TestMongo_ReadOnlyBlocksMutations(t *testing.T) {
	source := &MongoSource{readOnly: true}
	exec := &MongoExecutor{source: source}

	_, err := exec.Execute(context.Background(), "db.users.insertOne({name: 'Blocked'})", 0, 10)
	if err == nil {
		t.Fatal("expected mutation to be blocked in read-only mode")
	}
}
