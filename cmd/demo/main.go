// Command demo creates a sample SQLite database (demo.db) to try relm without
// needing the sqlite3 CLI. Cross-platform replacement for `make demo`.
//
// It seeds 20 tables with a few thousand rows each (users, orders, products,
// payments, logs, ...) so pagination and the browser can be exercised with a
// real amount of data.
package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	path := "demo.db"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	if err := create(path); err != nil {
		fmt.Fprintf(os.Stderr, "demo: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Example database created: %s - open it with relm (engine SQLite, path %s)\n", path, path)
}

func create(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	rng := rand.New(rand.NewSource(42)) // reproducible dataset

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	exec := func(q string, args ...any) error {
		_, err := tx.Exec(q, args...)
		return err
	}

	// ---------------------------------------------------------------------
	// Schema: 20 tables with realistic columns.
	// ---------------------------------------------------------------------
	schema := []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT UNIQUE,
			password_hash TEXT, role TEXT, active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE customers (
			id INTEGER PRIMARY KEY, first_name TEXT, last_name TEXT, email TEXT,
			phone TEXT, country TEXT, city TEXT, vip INTEGER NOT NULL DEFAULT 0,
			loyalty_points INTEGER NOT NULL DEFAULT 0, created_at TEXT)`,
		`CREATE TABLE addresses (
			id INTEGER PRIMARY KEY, customer_id INTEGER, street TEXT, city TEXT,
			state TEXT, zip TEXT, country TEXT, is_primary INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE categories (
			id INTEGER PRIMARY KEY, name TEXT, description TEXT, parent_id INTEGER)`,
		`CREATE TABLE products (
			id INTEGER PRIMARY KEY, name TEXT, sku TEXT, category_id INTEGER,
			price REAL, cost REAL, stock INTEGER NOT NULL DEFAULT 0,
			discontinued INTEGER NOT NULL DEFAULT 0, description TEXT, weight REAL)`,
		`CREATE TABLE suppliers (
			id INTEGER PRIMARY KEY, name TEXT, contact TEXT, email TEXT,
			phone TEXT, country TEXT)`,
		`CREATE TABLE orders (
			id INTEGER PRIMARY KEY, customer_id INTEGER, status TEXT,
			total REAL, shipping REAL, tax REAL, currency TEXT,
			created_at TEXT, shipped_at TEXT)`,
		`CREATE TABLE order_items (
			id INTEGER PRIMARY KEY, order_id INTEGER, product_id INTEGER,
			quantity INTEGER, unit_price REAL, discount REAL)`,
		`CREATE TABLE invoices (
			id INTEGER PRIMARY KEY, order_id INTEGER, number TEXT,
			issued_at TEXT, due_at TEXT, paid_at TEXT, amount REAL, status TEXT)`,
		`CREATE TABLE payments (
			id INTEGER PRIMARY KEY, invoice_id INTEGER, method TEXT,
			amount REAL, reference TEXT, paid_at TEXT)`,
		`CREATE TABLE departments (
			id INTEGER PRIMARY KEY, name TEXT, location TEXT)`,
		`CREATE TABLE employees (
			id INTEGER PRIMARY KEY, name TEXT, email TEXT, department_id INTEGER,
			salary REAL, hired_at TEXT, manager_id INTEGER)`,
		`CREATE TABLE projects (
			id INTEGER PRIMARY KEY, name TEXT, department_id INTEGER,
			budget REAL, start_at TEXT, end_at TEXT, status TEXT)`,
		`CREATE TABLE tasks (
			id INTEGER PRIMARY KEY, project_id INTEGER, assignee_id INTEGER,
			title TEXT, status TEXT, priority INTEGER, estimate_hours REAL, due_at TEXT)`,
		`CREATE TABLE comments (
			id INTEGER PRIMARY KEY, task_id INTEGER, author_id INTEGER,
			body TEXT, created_at TEXT)`,
		`CREATE TABLE events (
			id INTEGER PRIMARY KEY, name TEXT, payload TEXT, occurred_at TEXT)`,
		`CREATE TABLE sessions (
			id INTEGER PRIMARY KEY, user_id INTEGER, token TEXT, ip TEXT,
			user_agent TEXT, started_at TEXT, expires_at TEXT)`,
		`CREATE TABLE notifications (
			id INTEGER PRIMARY KEY, user_id INTEGER, kind TEXT, message TEXT,
			read INTEGER NOT NULL DEFAULT 0, created_at TEXT)`,
		`CREATE TABLE logs (
			id INTEGER PRIMARY KEY, level TEXT, source TEXT, message TEXT, created_at TEXT)`,
		`CREATE TABLE configs (
			id INTEGER PRIMARY KEY, key TEXT UNIQUE, value TEXT)`,
	}
	for _, s := range schema {
		if _, err := tx.Exec(s); err != nil {
			tx.Rollback()
			return fmt.Errorf("%s: %w", s, err)
		}
	}

	// ---------------------------------------------------------------------
	// Data
	// ---------------------------------------------------------------------
	ts := time.Now().Add(-365 * 24 * time.Hour)
	rfc := func(d time.Time) string { return d.Format(time.RFC3339) }
	date := func() time.Time { return ts.Add(time.Duration(rng.Intn(365*24)) * time.Hour) }
	randStr := func(prefix string, n int) string {
		b := make([]byte, n)
		const letters = "abcdefghijklmnopqrstuvwxyz"
		for i := range b {
			b[i] = letters[rng.Intn(len(letters))]
		}
		return prefix + string(b)
	}
	email := func(name string) string {
		return fmt.Sprintf("%s%d@example.com", name, rng.Intn(9999))
	}

	names := make([]string, 4000)
	for i := range names {
		names[i] = randStr("user", 6)
	}

	// users (2k)
	for i := 1; i <= 2000; i++ {
		n := names[i%len(names)]
		if err := exec(`INSERT INTO users (name,email,password_hash,role,active,created_at,updated_at)
			VALUES (?,?,?,?,?,?,?)`,
			n, email(n), randStr("$2a$", 12), []string{"admin", "staff", "viewer"}[rng.Intn(3)],
			rng.Intn(2), rfc(date()), rfc(date())); err != nil {
			return err
		}
	}

	// customers (2.5k)
	for i := 1; i <= 2500; i++ {
		fn := randStr("f", 5)
		ln := randStr("l", 7)
		if err := exec(`INSERT INTO customers (first_name,last_name,email,phone,country,city,vip,loyalty_points,created_at)
			VALUES (?,?,?,?,?,?,?,?,?)`,
			fn, ln, email(fn), fmt.Sprintf("+1%09d", rng.Intn(1e9)),
			[]string{"US", "UK", "ES", "AR", "BR", "DE"}[rng.Intn(6)],
			randStr("C", 6), rng.Intn(2), rng.Intn(5000), rfc(date())); err != nil {
			return err
		}
	}

	// addresses (3k)
	for i := 1; i <= 3000; i++ {
		if err := exec(`INSERT INTO addresses (customer_id,street,city,state,zip,country,is_primary)
			VALUES (?,?,?,?,?,?,?)`,
			rng.Intn(2500)+1, fmt.Sprintf("%d %s St", rng.Intn(9999), randStr("S", 6)),
			randStr("City", 5), randStr("ST", 2), fmt.Sprintf("%05d", rng.Intn(99999)),
			[]string{"US", "UK", "ES", "AR"}[rng.Intn(4)], rng.Intn(2)); err != nil {
			return err
		}
	}

	// categories (60)
	for i := 1; i <= 60; i++ {
		parent := 0
		if i > 10 {
			parent = rng.Intn(10) + 1
		}
		if err := exec(`INSERT INTO categories (name,description,parent_id) VALUES (?,?,?)`,
			randStr("cat_", 6), randStr("desc ", 20), parent); err != nil {
			return err
		}
	}

	// products (1.5k)
	for i := 1; i <= 1500; i++ {
		if err := exec(`INSERT INTO products (name,sku,category_id,price,cost,stock,discontinued,description,weight)
			VALUES (?,?,?,?,?,?,?,?,?)`,
			randStr("prod_", 8), randStr("SKU-", 8), rng.Intn(60)+1,
			float64(rng.Intn(20000))/100, float64(rng.Intn(15000))/100,
			rng.Intn(500), rng.Intn(2), randStr("desc ", 30), float64(rng.Intn(5000))/100); err != nil {
			return err
		}
	}

	// suppliers (300)
	for i := 1; i <= 300; i++ {
		if err := exec(`INSERT INTO suppliers (name,contact,email,phone,country) VALUES (?,?,?,?,?)`,
			randStr("supp_", 7), randStr("contact_", 8), email(randStr("s", 5)),
			fmt.Sprintf("+1%09d", rng.Intn(1e9)), []string{"CN", "US", "DE", "BR"}[rng.Intn(4)]); err != nil {
			return err
		}
	}

	// orders (6k) + order_items (20k)
	statuses := []string{"pending", "paid", "shipped", "delivered", "cancelled", "refunded"}
	for i := 1; i <= 6000; i++ {
		shipping := float64(rng.Intn(3000)) / 100
		tax := float64(rng.Intn(1500)) / 100
		if err := exec(`INSERT INTO orders (customer_id,status,total,shipping,tax,currency,created_at,shipped_at)
			VALUES (?,?,?,?,?,?,?,?)`,
			rng.Intn(2500)+1, statuses[rng.Intn(len(statuses))],
			float64(rng.Intn(50000))/100, shipping, tax,
			[]string{"USD", "EUR", "GBP", "ARS"}[rng.Intn(4)],
			rfc(date()), rfc(date())); err != nil {
			return err
		}
		items := rng.Intn(6) + 1
		for j := 0; j < items; j++ {
			qty := rng.Intn(10) + 1
			unit := float64(rng.Intn(20000)) / 100
			if err := exec(`INSERT INTO order_items (order_id,product_id,quantity,unit_price,discount)
				VALUES (?,?,?,?,?)`,
				i, rng.Intn(1500)+1, qty, unit, float64(rng.Intn(50))/100); err != nil {
				return err
			}
		}
	}

	// invoices (6k) + payments (8k)
	for i := 1; i <= 6000; i++ {
		paid := rng.Intn(2)
		paidAt := "NULL"
		if paid == 1 {
			paidAt = "'" + rfc(date()) + "'"
		}
		if err := exec(fmt.Sprintf(`INSERT INTO invoices (order_id,number,issued_at,due_at,paid_at,amount,status)
			VALUES (?,?,?,?,%s,?,?)`, paidAt),
			rng.Intn(6000)+1, fmt.Sprintf("INV-%05d", i), rfc(date()), rfc(date()),
			float64(rng.Intn(50000))/100, statuses[rng.Intn(4)]); err != nil {
			return err
		}
	}
	for i := 1; i <= 8000; i++ {
		if err := exec(`INSERT INTO payments (invoice_id,method,amount,reference,paid_at) VALUES (?,?,?,?,?)`,
			rng.Intn(6000)+1, []string{"card", "cash", "transfer", "paypal"}[rng.Intn(4)],
			float64(rng.Intn(20000))/100, randStr("PAY-", 10), rfc(date())); err != nil {
			return err
		}
	}

	// departments (25) + employees (800)
	for i := 1; i <= 25; i++ {
		if err := exec(`INSERT INTO departments (name,location) VALUES (?,?)`,
			randStr("dept_", 6), randStr("loc_", 5)); err != nil {
			return err
		}
	}
	for i := 1; i <= 800; i++ {
		if err := exec(`INSERT INTO employees (name,email,department_id,salary,hired_at,manager_id) VALUES (?,?,?,?,?,?)`,
			randStr("emp_", 7), email(randStr("e", 5)), rng.Intn(25)+1,
			float64(rng.Intn(150000)+30000)/100, rfc(date()), rng.Intn(800)+1); err != nil {
			return err
		}
	}

	// projects (400) + tasks (4k)
	for i := 1; i <= 400; i++ {
		if err := exec(`INSERT INTO projects (name,department_id,budget,start_at,end_at,status) VALUES (?,?,?,?,?,?)`,
			randStr("proj_", 7), rng.Intn(25)+1, float64(rng.Intn(1000000))/100,
			rfc(date()), rfc(date()), statuses[rng.Intn(3)]); err != nil {
			return err
		}
	}
	taskStatus := []string{"todo", "doing", "review", "done"}
	for i := 1; i <= 4000; i++ {
		if err := exec(`INSERT INTO tasks (project_id,assignee_id,title,status,priority,estimate_hours,due_at) VALUES (?,?,?,?,?,?,?)`,
			rng.Intn(400)+1, rng.Intn(800)+1, randStr("task_", 10),
			taskStatus[rng.Intn(len(taskStatus))], rng.Intn(5)+1,
			float64(rng.Intn(80))/10, rfc(date())); err != nil {
			return err
		}
	}

	// comments (8k)
	for i := 1; i <= 8000; i++ {
		if err := exec(`INSERT INTO comments (task_id,author_id,body,created_at) VALUES (?,?,?,?)`,
			rng.Intn(4000)+1, rng.Intn(800)+1, randStr("comment ", 60), rfc(date())); err != nil {
			return err
		}
	}

	// events (10k)
	for i := 1; i <= 10000; i++ {
		if err := exec(`INSERT INTO events (name,payload,occurred_at) VALUES (?,?,?)`,
			[]string{"login", "logout", "click", "purchase", "export"}[rng.Intn(5)],
			randStr("{} \"payload\": \"", 30), rfc(date())); err != nil {
			return err
		}
	}

	// sessions (12k)
	for i := 1; i <= 12000; i++ {
		if err := exec(`INSERT INTO sessions (user_id,token,ip,user_agent,started_at,expires_at) VALUES (?,?,?,?,?,?)`,
			rng.Intn(2000)+1, randStr("tok_", 24), fmt.Sprintf("%d.%d.%d.%d", rng.Intn(256), rng.Intn(256), rng.Intn(256), rng.Intn(256)),
			randStr("ua/", 20), rfc(date()), rfc(date())); err != nil {
			return err
		}
	}

	// notifications (15k)
	for i := 1; i <= 15000; i++ {
		if err := exec(`INSERT INTO notifications (user_id,kind,message,read,created_at) VALUES (?,?,?,?,?)`,
			rng.Intn(2000)+1, []string{"info", "alert", "billing", "security"}[rng.Intn(4)],
			randStr("msg ", 40), rng.Intn(2), rfc(date())); err != nil {
			return err
		}
	}

	// logs (30k)
	for i := 1; i <= 30000; i++ {
		if err := exec(`INSERT INTO logs (level,source,message,created_at) VALUES (?,?,?,?)`,
			[]string{"debug", "info", "warn", "error"}[rng.Intn(4)],
			[]string{"api", "db", "auth", "worker", "cron"}[rng.Intn(5)],
			randStr("log ", 50), rfc(date())); err != nil {
			return err
		}
	}

	// configs (200)
	for i := 1; i <= 200; i++ {
		if err := exec(`INSERT INTO configs (key,value) VALUES (?,?)`,
			randStr("cfg.", 12), randStr("v", 16)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
