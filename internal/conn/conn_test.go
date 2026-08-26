package conn

import "testing"

func TestDefaultPort(t *testing.T) {
	cases := map[Driver]int{
		DriverSQLite:    0,
		DriverPostgres:  5432,
		DriverMySQL:     3306,
		DriverMariaDB:   3306,
		DriverMSSQL:     1433,
		DriverMongo:     27017,
		DriverRedis:     6379,
		DriverCassandra: 9042,
		DriverNeo4j:     7687,
	}
	for d, want := range cases {
		if got := DefaultPort(d); got != want {
			t.Errorf("DefaultPort(%s) = %d, want %d", d, got, want)
		}
	}
}

func TestDriversList(t *testing.T) {
	if len(Drivers) != 9 {
		t.Fatalf("Drivers = %d, want 9", len(Drivers))
	}
}

func TestNeedsNetwork(t *testing.T) {
	if NeedsNetwork(DriverSQLite) {
		t.Error("sqlite should not require the network")
	}
	for _, d := range Drivers {
		if d != DriverSQLite && !NeedsNetwork(d) {
			t.Errorf("%s should require the network", d)
		}
	}
}

func TestLabel(t *testing.T) {
	cases := []struct {
		cfg  ConnectionConfig
		want string
	}{
		{ConnectionConfig{Driver: DriverSQLite, Path: "/data/app.db"}, "sqlite /data/app.db"},
		{ConnectionConfig{Driver: DriverRedis, Host: "localhost", Port: 6379, Database: "0"}, "redis localhost:6379/0"},
		{ConnectionConfig{Driver: DriverMongo, Host: "localhost", Port: 27017, Database: "test"}, "mongodb localhost:27017/test"},
		{ConnectionConfig{Driver: DriverCassandra, Host: "localhost", Port: 9042, Database: "keyspace1"}, "cassandra localhost:9042/keyspace1"},
		{ConnectionConfig{Driver: DriverNeo4j, User: "neo4j", Host: "localhost", Port: 7687, Database: "neo4j"}, "neo4j neo4j@localhost:7687/neo4j"},
	}
	for _, c := range cases {
		if got := c.cfg.Label(); got != c.want {
			t.Errorf("Label() = %q, want %q", got, c.want)
		}
	}
}
