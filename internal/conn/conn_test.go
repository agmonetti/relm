package conn

import "testing"

func TestDefaultPort(t *testing.T) {
	cases := map[Driver]int{
		DriverSQLite:   0,
		DriverPostgres: 5432,
		DriverMySQL:    3306,
		DriverMariaDB:  3306,
		DriverMSSQL:    1433,
	}
	for d, want := range cases {
		if got := DefaultPort(d); got != want {
			t.Errorf("DefaultPort(%s) = %d, want %d", d, got, want)
		}
	}
}

func TestDriversList(t *testing.T) {
	if len(Drivers) != 5 {
		t.Fatalf("Drivers = %d, want 5", len(Drivers))
	}
}

func TestNeedsNetwork(t *testing.T) {
	if NeedsNetwork(DriverSQLite) {
		t.Error("sqlite no requiere red")
	}
	for _, d := range Drivers {
		if d != DriverSQLite && !NeedsNetwork(d) {
			t.Errorf("%s debería requerir red", d)
		}
	}
}
