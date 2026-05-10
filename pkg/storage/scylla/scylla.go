package scylla

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
)

// Session abstracts the Scylla session behavior used by the bootstrap logic.
type Session interface {
	Close()
	Query(stmt string, values ...any) Query
	Raw() *gocql.Session
}

// Query abstracts the Scylla query execution path used by the bootstrap logic.
type Query interface {
	WithContext(ctx context.Context) Query
	Consistency(consistency gocql.Consistency) Query
	Exec() error
}

type sessionAdapter struct {
	db *gocql.Session
}

func (s sessionAdapter) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s sessionAdapter) Query(stmt string, values ...any) Query {
	return queryAdapter{q: s.db.Query(stmt, values...)}
}

func (s sessionAdapter) Raw() *gocql.Session {
	return s.db
}

type queryAdapter struct {
	q *gocql.Query
}

func (q queryAdapter) Exec() error {
	return q.q.Exec()
}

func (q queryAdapter) WithContext(ctx context.Context) Query {
	q.q = q.q.WithContext(ctx)
	return q
}

func (q queryAdapter) Consistency(consistency gocql.Consistency) Query {
	q.q = q.q.Consistency(consistency)
	return q
}

var openSession = func(cluster *gocql.ClusterConfig) (Session, error) {
	db, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}
	return sessionAdapter{db: db}, nil
}

// Options groups the Scylla connection and schema settings used by
// file-service.
type Options struct {
	Hosts                  []string
	Port                   int
	Keyspace               string
	Username               string
	Password               string
	Consistency            string
	ConnectTimeout         time.Duration
	MaxWaitSchemaAgreement time.Duration
	RetryAttempts          int
	RetryBackoff           time.Duration
}

// ConnectAndEnsureSchema connects to ScyllaDB, creates the keyspace/table
// owned by file-service, and returns the application session.
func ConnectAndEnsureSchema(cfg Options) (Session, error) {
	cluster := gocql.NewCluster(cfg.Hosts...)
	cluster.Port = cfg.Port
	cluster.Timeout = cfg.ConnectTimeout
	cluster.ConnectTimeout = cfg.ConnectTimeout
	cluster.MaxWaitSchemaAgreement = cfg.MaxWaitSchemaAgreement
	cluster.Consistency = parseConsistency(cfg.Consistency)
	if cfg.Username != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: cfg.Username,
			Password: cfg.Password,
		}
	}

	var sys Session
	var err error
	for i := 0; i < cfg.RetryAttempts; i++ {
		sys, err = openSession(cluster)
		if err == nil {
			break
		}
		time.Sleep(cfg.RetryBackoff)
	}
	if err != nil {
		return nil, WrapCreateClusterSessionError(err)
	}

	defer sys.Close()

	if err := sys.Query(fmt.Sprintf(`
		CREATE KEYSPACE IF NOT EXISTS %s
		WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': 1}
	`, cfg.Keyspace)).Exec(); err != nil {
		return nil, WrapEnsureSchemaError(err)
	}

	cluster.Keyspace = cfg.Keyspace

	var app Session
	for i := 0; i < cfg.RetryAttempts; i++ {
		app, err = openSession(cluster)
		if err == nil {
			break
		}
		time.Sleep(cfg.RetryBackoff)
	}
	if err != nil {
		return nil, WrapCreateClusterSessionError(err)
	}

	if err := app.Query(`
		CREATE TABLE IF NOT EXISTS files (
			file_id TEXT PRIMARY KEY,
			owner_id TEXT,
			filename TEXT,
			extension TEXT,
			content_type TEXT,
			bucket TEXT,
			storage_path TEXT,
			size_bytes BIGINT,
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		)
	`).Exec(); err != nil {
		app.Close()
		return nil, WrapEnsureSchemaError(err)
	}

	return app, nil
}

func parseConsistency(level string) gocql.Consistency {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "one":
		return gocql.One
	case "localquorum":
		return gocql.LocalQuorum
	case "all":
		return gocql.All
	default:
		return gocql.Quorum
	}
}
