package redis

import (
	"context"
	"testing"
	"time"

	"github.com/agmonetti/relm/internal/conn"
	"github.com/agmonetti/relm/internal/store"
)

func TestRedis_Registered(t *testing.T) {
	cfg := conn.New(conn.DriverRedis)
	cfg.Host = "127.0.0.1"
	cfg.Port = 65533 // unopened port

	_, err := store.New(cfg)
	if err == nil {
		t.Fatal("expected connection error on unopened port")
	}
	if err.Error() == "unsupported driver: redis" {
		t.Fatal("DriverRedis not registered in store registry")
	}
}

func TestRedis_ParseCommandArgs(t *testing.T) {
	args, err := parseRedisCommandArgs(`SET "my key" 'my value with spaces'`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(args) != 3 {
		t.Fatalf("args len = %d, want 3", len(args))
	}
	if args[0] != "SET" || args[1] != "my key" || args[2] != "my value with spaces" {
		t.Errorf("args = %v", args)
	}

	args2, err := parseRedisCommandArgs(`HGETALL   users:1001 `)
	if err != nil || len(args2) != 2 || args2[0] != "HGETALL" || args2[1] != "users:1001" {
		t.Errorf("args2 = %v, err = %v", args2, err)
	}
}

func TestRedis_FormatTTL(t *testing.T) {
	if s := formatTTL(-1 * time.Nanosecond); s != "-1 (no expiry)" {
		t.Errorf("formatTTL(-1) = %q", s)
	}
	if s := formatTTL(45 * time.Second); s != "45s" {
		t.Errorf("formatTTL(45s) = %q", s)
	}
	if s := formatTTL(125 * time.Second); s != "2m5s" {
		t.Errorf("formatTTL(125s) = %q", s)
	}
	if s := formatTTL(3700 * time.Second); s != "1h1m" {
		t.Errorf("formatTTL(3700s) = %q", s)
	}
}

func TestRedis_ExecutorIsMutationAndReadOnly(t *testing.T) {
	source := &RedisSource{readOnly: true}
	exec := &RedisExecutor{source: source}

	if !exec.IsMutation("SET foo bar") {
		t.Error("SET must be detected as mutation")
	}
	if !exec.IsMutation("HSET user name Alice") {
		t.Error("HSET must be detected as mutation")
	}
	if !exec.IsMutation("DEL foo") {
		t.Error("DEL must be detected as mutation")
	}
	if exec.IsMutation("GET foo") {
		t.Error("GET must not be detected as mutation")
	}
	if exec.IsMutation("HGETALL user") {
		t.Error("HGETALL must not be detected as mutation")
	}

	_, err := exec.Execute(context.Background(), "SET foo bar", 0, 10)
	if err == nil {
		t.Fatal("expected mutation to be blocked in read-only mode")
	}
}

func TestRedis_FormatResult(t *testing.T) {
	strRes := formatRedisResult("OK")
	tab, ok := strRes.(*store.TabularData)
	if !ok || len(tab.Rows) != 1 || tab.Rows[0][0] != "OK" {
		t.Errorf("string result = %v", strRes)
	}

	intRes := formatRedisResult(int64(42))
	tab2, ok2 := intRes.(*store.TabularData)
	if !ok2 || len(tab2.Rows) != 1 || tab2.Rows[0][0] != "42" {
		t.Errorf("int result = %v", intRes)
	}

	sliceRes := formatRedisResult([]any{"item1", "item2"})
	tab3, ok3 := sliceRes.(*store.TabularData)
	if !ok3 || len(tab3.Rows) != 2 {
		t.Errorf("slice result = %v", sliceRes)
	}
}
