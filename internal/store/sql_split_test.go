package store

import (
	"testing"
)

func TestSplitStatements_Basic(t *testing.T) {
	sql := "SELECT 1; SELECT 2;\nSELECT 3"
	stmts := SplitStatements(sql)
	if len(stmts) != 3 {
		t.Fatalf("expected 3 statements, got %d: %+v", len(stmts), stmts)
	}
	if stmts[0].Text != "SELECT 1" || stmts[0].Line != 0 {
		t.Errorf("stmt 0: %+v", stmts[0])
	}
	if stmts[1].Text != "SELECT 2" || stmts[1].Line != 0 {
		t.Errorf("stmt 1: %+v", stmts[1])
	}
	if stmts[2].Text != "SELECT 3" || stmts[2].Line != 1 {
		t.Errorf("stmt 2: %+v", stmts[2])
	}
}

func TestSplitStatements_Comments(t *testing.T) {
	sql := `// Double slash comment
SELECT 'hello;world';
-- Dash dash comment
SELECT 2; /* block comment; */
# Hash comment
SELECT 3`
	stmts := SplitStatements(sql)
	if len(stmts) != 3 {
		t.Fatalf("expected 3 statements, got %d: %+v", len(stmts), stmts)
	}
	if stmts[0].Text != "SELECT 'hello;world'" || stmts[0].Line != 1 {
		t.Errorf("stmt 0: %+v", stmts[0])
	}
	if stmts[1].Text != "SELECT 2" || stmts[1].Line != 3 {
		t.Errorf("stmt 1: %+v", stmts[1])
	}
	if stmts[2].Text != "SELECT 3" || stmts[2].Line != 5 {
		t.Errorf("stmt 2: %+v", stmts[2])
	}
}
