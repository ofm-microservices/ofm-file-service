package scylla

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gocql/gocql"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeQuery struct {
	execErr error
	execCnt  int
}

func (q *fakeQuery) WithContext(context.Context) Query { return q }
func (q *fakeQuery) Consistency(gocql.Consistency) Query { return q }
func (q *fakeQuery) Exec() error {
	q.execCnt++
	return q.execErr
}

type fakeSession struct {
	queries []string
	queryFn func(string) *fakeQuery
	closeCnt int
	mu      sync.Mutex
}

func (s *fakeSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeCnt++
}

func (s *fakeSession) Query(stmt string, _ ...any) Query {
	s.mu.Lock()
	s.queries = append(s.queries, stmt)
	s.mu.Unlock()
	if s.queryFn != nil {
		return s.queryFn(stmt)
	}
	return &fakeQuery{}
}

func (s *fakeSession) Raw() *gocql.Session { return nil }

var _ = Describe("wrappers", func() {
	It("wraps errors", func() {
		Expect(WrapCreateClusterSessionError(errors.New("boom")).Error()).To(ContainSubstring("create cluster session"))
		Expect(WrapEnsureSchemaError(errors.New("boom")).Error()).To(ContainSubstring("ensure schema"))
		Expect(WrapOpenSessionError(errors.New("boom")).Error()).To(ContainSubstring("open scylla session"))
	})
})

var _ = DescribeTable("parseConsistency",
	func(input string, expected gocql.Consistency) {
		Expect(parseConsistency(input)).To(Equal(expected))
	},
	Entry("default", "", gocql.Quorum),
	Entry("quorum fallback", "whatever", gocql.Quorum),
	Entry("one", "one", gocql.One),
	Entry("local quorum", " localquorum ", gocql.LocalQuorum),
	Entry("all", "ALL", gocql.All),
)

var _ = Describe("ConnectAndEnsureSchema", func() {
	origOpenSession := openSession

	BeforeEach(func() {
		openSession = func(*gocql.ClusterConfig) (Session, error) {
			return &fakeSession{}, nil
		}
	})

	AfterEach(func() {
		openSession = origOpenSession
	})

	It("wraps session creation failures quickly", func() {
		openSession = func(*gocql.ClusterConfig) (Session, error) {
			return nil, errors.New("boom")
		}

		sess, err := ConnectAndEnsureSchema(Options{
			Hosts:          []string{"127.0.0.1"},
			Port:           1,
			Keyspace:       "file_service",
			ConnectTimeout: time.Millisecond,
			RetryAttempts:  1,
			RetryBackoff:   0,
		})
		Expect(sess).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("create cluster session"))
	})

	It("wraps keyspace creation failures", func() {
		sys := &fakeSession{
			queryFn: func(stmt string) *fakeQuery {
				return &fakeQuery{execErr: errors.New("keyspace failed")}
			},
		}
		openCalls := 0
		openSession = func(*gocql.ClusterConfig) (Session, error) {
			openCalls++
			if openCalls == 1 {
				return sys, nil
			}
			return &fakeSession{}, nil
		}

		sess, err := ConnectAndEnsureSchema(Options{
			Hosts:          []string{"127.0.0.1"},
			Port:           1,
			Keyspace:       "file_service",
			ConnectTimeout: time.Millisecond,
			RetryAttempts:  1,
			RetryBackoff:   0,
		})
		Expect(sess).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("ensure schema"))
		Expect(sys.closeCnt).To(Equal(1))
	})

	It("wraps table creation failures and closes the app session", func() {
		sys := &fakeSession{
			queryFn: func(stmt string) *fakeQuery {
				return &fakeQuery{}
			},
		}
		app := &fakeSession{
			queryFn: func(stmt string) *fakeQuery {
				return &fakeQuery{execErr: errors.New("table failed")}
			},
		}
		openCalls := 0
		openSession = func(*gocql.ClusterConfig) (Session, error) {
			openCalls++
			if openCalls == 1 {
				return sys, nil
			}
			return app, nil
		}

		sess, err := ConnectAndEnsureSchema(Options{
			Hosts:          []string{"127.0.0.1"},
			Port:           1,
			Keyspace:       "file_service",
			ConnectTimeout: time.Millisecond,
			RetryAttempts:  1,
			RetryBackoff:   0,
		})
		Expect(sess).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("ensure schema"))
		Expect(sys.closeCnt).To(Equal(1))
		Expect(app.closeCnt).To(Equal(1))
	})

	It("returns the app session on success", func() {
		sys := &fakeSession{queryFn: func(stmt string) *fakeQuery { return &fakeQuery{} }}
		app := &fakeSession{queryFn: func(stmt string) *fakeQuery { return &fakeQuery{} }}
		openCalls := 0
		openSession = func(*gocql.ClusterConfig) (Session, error) {
			openCalls++
			if openCalls == 1 {
				return sys, nil
			}
			return app, nil
		}

		sess, err := ConnectAndEnsureSchema(Options{
			Hosts:          []string{"127.0.0.1"},
			Port:           1,
			Keyspace:       "file_service",
			ConnectTimeout: time.Millisecond,
			RetryAttempts:  1,
			RetryBackoff:   0,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(sess).To(Equal(app))
		Expect(sys.closeCnt).To(Equal(1))
		Expect(app.closeCnt).To(Equal(0))
	})
})

var _ = Describe("adapters", func() {
	It("wraps the gocql session and query chain", func() {
		adapter := sessionAdapter{db: &gocql.Session{}}
		adapter.Close()
		Expect(adapter.Raw()).To(Equal(adapter.db))

		q := adapter.Query("SELECT 1")
		Expect(q).NotTo(BeNil())
		Expect(q.WithContext(context.Background())).NotTo(BeNil())
		Expect(q.Consistency(gocql.One)).NotTo(BeNil())
	})
})
