package cassandra

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"

	"github.com/agmonetti/relm/internal/conn"
	"github.com/agmonetti/relm/internal/store"
)

func init() {
	store.Register(conn.DriverCassandra, New)
}

// CassandraSource implements store.DataSource for Apache Cassandra / ScyllaDB.
type CassandraSource struct {
	session  *gocql.Session
	keyspace string
	readOnly bool
}

// New creates and connects a CassandraSource.
func New(cfg conn.ConnectionConfig) (store.DataSource, error) {
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port <= 0 {
		port = 9042
	}

	cluster := gocql.NewCluster(host)
	cluster.Port = port
	cluster.ConnectTimeout = 10 * time.Second
	cluster.Timeout = 10 * time.Second
	cluster.Consistency = gocql.LocalOne

	if cfg.Database != "" {
		cluster.Keyspace = cfg.Database
	} else {
		cluster.Keyspace = "system"
	}

	if cfg.User != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: cfg.User,
			Password: cfg.Password,
		}
	}

	if cfg.SSLMode != "" && cfg.SSLMode != "disable" {
		cluster.SslOpts = &gocql.SslOptions{
			Config: &tls.Config{
				InsecureSkipVerify: cfg.SSLMode == "skip-verify" || cfg.SSLMode == "insecure",
			},
			EnableHostVerification: false,
		}
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("cassandra connect: %w", err)
	}

	return &CassandraSource{
		session:  session,
		keyspace: cluster.Keyspace,
		readOnly: cfg.ReadOnly,
	}, nil
}

func (s *CassandraSource) Driver() conn.Driver {
	return conn.DriverCassandra
}

func (s *CassandraSource) Version(ctx context.Context) (string, error) {
	var version string
	iter := s.session.Query("SELECT release_version FROM system.local").WithContext(ctx).Iter()
	if iter.Scan(&version) {
		_ = iter.Close()
		return "Cassandra " + version, nil
	}
	_ = iter.Close()
	return "Cassandra", nil
}

func (s *CassandraSource) Close() error {
	s.session.Close()
	return nil
}

func (s *CassandraSource) ReadOnly() bool {
	return s.readOnly
}

func (s *CassandraSource) Catalog() store.CatalogDescriptor {
	return store.CatalogDescriptor{
		Title:    "TABLES",
		ItemNoun: "table",
		ListObjects: func(ctx context.Context) ([]store.CatalogItem, error) {
			query := "SELECT table_name FROM system_schema.tables WHERE keyspace_name = ?"
			iter := s.session.Query(query, s.keyspace).WithContext(ctx).Iter()

			var items []store.CatalogItem
			var tableName string
			for iter.Scan(&tableName) {
				items = append(items, store.CatalogItem{
					Name: tableName,
				})
			}
			if err := iter.Close(); err != nil {
				return nil, fmt.Errorf("list tables: %w", err)
			}
			return items, nil
		},
	}
}

func (s *CassandraSource) Browse(ctx context.Context, req store.BrowseRequest) (store.BrowseResponse, error) {
	if req.ObjectName == "" {
		return store.BrowseResponse{}, errors.New("no table specified")
	}

	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	cql := fmt.Sprintf("SELECT * FROM %s.%s", s.keyspace, req.ObjectName)
	query := s.session.Query(cql).WithContext(ctx).PageSize(pageSize)

	if req.Cursor != "" {
		pageBytes, err := hex.DecodeString(req.Cursor)
		if err == nil && len(pageBytes) > 0 {
			query = query.PageState(pageBytes)
		}
	}

	iter := query.Iter()
	cols := make([]string, len(iter.Columns()))
	for i, c := range iter.Columns() {
		cols[i] = c.Name
	}

	// Read exactly one page: gocql transparently fetches subsequent pages as
	// they are scanned, so stop after pageSize rows to keep the PageState
	// cursor meaningful for the browser's next-page request.
	var rows [][]string
	for len(rows) < pageSize {
		rowMap := make(map[string]interface{})
		if !iter.MapScan(rowMap) {
			break
		}
		row := make([]string, len(cols))
		for i, c := range cols {
			val := rowMap[c]
			if val != nil {
				row[i] = fmt.Sprint(val)
			} else {
				row[i] = ""
			}
		}
		rows = append(rows, row)
	}

	nextState := iter.PageState()
	var nextCursor string
	hasNext := len(rows) > 0 && len(nextState) > 0
	if hasNext {
		nextCursor = hex.EncodeToString(nextState)
	}

	if err := iter.Close(); err != nil {
		return store.BrowseResponse{}, err
	}

	tabData := &store.TabularData{
		Columns:   cols,
		Rows:      rows,
		TotalRows: -1, // unbounded in Cassandra
	}

	return store.BrowseResponse{
		Data:       tabData,
		HasNext:    hasNext,
		NextCursor: nextCursor,
		TotalCount: -1,
	}, nil
}

func (s *CassandraSource) Inspect(ctx context.Context, name string) (store.InspectionView, error) {
	// Query system_schema.columns for PK, Clustering, and Regular columns
	colQuery := "SELECT column_name, type, kind, position FROM system_schema.columns WHERE keyspace_name = ? AND table_name = ?"
	iter := s.session.Query(colQuery, s.keyspace, name).WithContext(ctx).Iter()

	var cols []store.Column
	var colName, colType, kind string
	var pos int

	for iter.Scan(&colName, &colType, &kind, &pos) {
		isPK := kind == "partition_key"
		isClustering := kind == "clustering"

		cols = append(cols, store.Column{
			Name:       colName,
			Type:       colType,
			PK:         isPK,
			Clustering: isClustering,
		})
	}
	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("inspect columns: %w", err)
	}

	// Query system_schema.indexes
	ixQuery := "SELECT index_name, options FROM system_schema.indexes WHERE keyspace_name = ? AND table_name = ?"
	ixIter := s.session.Query(ixQuery, s.keyspace, name).WithContext(ctx).Iter()

	var indexes []store.Index
	var ixName string
	var options map[string]string

	for ixIter.Scan(&ixName, &options) {
		target := options["target"]
		indexes = append(indexes, store.Index{
			Name:    ixName,
			Columns: []string{target},
		})
	}
	_ = ixIter.Close()

	return &store.RelationalStructure{
		Columns: cols,
		Indexes: indexes,
	}, nil
}

func (s *CassandraSource) Query() store.QueryExecutor {
	return &CassandraExecutor{source: s}
}

// CassandraExecutor executes CQL queries.
type CassandraExecutor struct {
	source *CassandraSource
}

func (e *CassandraExecutor) Language() string {
	return "CQL"
}

func (e *CassandraExecutor) PromptTitle() string {
	return "CQL QUERY"
}

func (e *CassandraExecutor) Placeholder() string {
	return "SELECT * FROM table LIMIT 10;"
}

var cqlMutations = []string{"INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "TRUNCATE", "BATCH"}

func (e *CassandraExecutor) IsMutation(stmt string) bool {
	upper := strings.ToUpper(strings.TrimSpace(stmt))
	for _, m := range cqlMutations {
		if strings.HasPrefix(upper, m) {
			return true
		}
	}
	return false
}

// Execute runs a CQL statement. The QueryExecutor contract passes
// (buffer, line, maxRows): line is the cursor line (unused) and maxRows caps
// how many rows a read query loads into memory (a hard breaker, since gocql
// iterates every server page otherwise).
func (e *CassandraExecutor) Execute(ctx context.Context, buffer string, _line int, maxRows int) (store.DataView, error) {
	trimmed := strings.TrimSpace(buffer)
	if trimmed == "" {
		return nil, errors.New("empty CQL query")
	}

	isMut := e.IsMutation(trimmed)
	if e.source.readOnly && isMut {
		return nil, errors.New("mutations are blocked in read-only mode")
	}

	if isMut {
		err := e.source.session.Query(trimmed).WithContext(ctx).Exec()
		if err != nil {
			return nil, err
		}
		return &store.TabularData{
			Affected: 0,
		}, nil
	}

	pageSize := maxRows
	if pageSize <= 0 {
		pageSize = 50
	}

	iter := e.source.session.Query(trimmed).WithContext(ctx).PageSize(pageSize).Iter()
	cols := make([]string, len(iter.Columns()))
	for i, c := range iter.Columns() {
		cols[i] = c.Name
	}

	var rows [][]string
	truncated := false
	for {
		if maxRows > 0 && len(rows) >= maxRows {
			truncated = true
			break
		}
		rowMap := make(map[string]interface{})
		if !iter.MapScan(rowMap) {
			break
		}
		row := make([]string, len(cols))
		for i, c := range cols {
			val := rowMap[c]
			if val != nil {
				row[i] = fmt.Sprint(val)
			} else {
				row[i] = ""
			}
		}
		rows = append(rows, row)
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}

	return &store.TabularData{
		Columns:   cols,
		Rows:      rows,
		Affected:  -1, // read query
		TotalRows: -1,
		Truncated: truncated,
	}, nil
}
