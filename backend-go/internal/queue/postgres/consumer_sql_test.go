package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func newQueueMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, mock
}

func TestConsumerPollReturnsMessagesAndMarksRunning(t *testing.T) {
	db, mock := newQueueMockDB(t)
	consumer := NewConsumer(db)
	mock.ExpectExec("UPDATE assistant_requests\\s+SET status = 'pending'").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectBegin()
	mock.ExpectQuery("FROM assistant_requests\\s+WHERE status = 'pending'").
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "request_type", "related_type", "related_id", "payload"}).
			AddRow("req-1", "assistant.ask", "document", "doc-1", `{"question":"hello"}`).
			AddRow("req-2", "document.summarize", "document", "doc-2", `{}`))
	mock.ExpectExec("UPDATE assistant_requests\\s+SET status = 'running'").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	got := consumer.Poll(context.Background(), 2)
	if len(got) != 2 || got[0].RequestID != "req-1" || got[0].Payload["question"] != "hello" {
		t.Fatalf("messages = %#v, want two decoded messages", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestConsumerPollDefaultsLimitAndHandlesEmptyQueue(t *testing.T) {
	db, mock := newQueueMockDB(t)
	consumer := NewConsumer(db)
	mock.ExpectExec("UPDATE assistant_requests\\s+SET status = 'pending'").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectBegin()
	mock.ExpectQuery("FROM assistant_requests\\s+WHERE status = 'pending'").
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "request_type", "related_type", "related_id", "payload"}))

	got := consumer.Poll(context.Background(), 0)
	if got != nil {
		t.Fatalf("messages = %#v, want nil for empty queue", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestConsumerPollReturnsNilOnDecodeError(t *testing.T) {
	db, mock := newQueueMockDB(t)
	consumer := NewConsumer(db)
	mock.ExpectExec("UPDATE assistant_requests\\s+SET status = 'pending'").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectBegin()
	mock.ExpectQuery("FROM assistant_requests\\s+WHERE status = 'pending'").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "request_type", "related_type", "related_id", "payload"}).
			AddRow("req-1", "assistant.ask", "document", "doc-1", `{bad`))

	got := consumer.Poll(context.Background(), 1)
	if got != nil {
		t.Fatalf("messages = %#v, want nil on bad payload", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestConsumerPollReturnsNilOnTransactionFailures(t *testing.T) {
	cases := []struct {
		name  string
		setup func(sqlmock.Sqlmock)
	}{
		{
			name: "begin",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE assistant_requests\\s+SET status = 'pending'").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectBegin().WillReturnError(errors.New("begin failed"))
			},
		},
		{
			name: "query",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE assistant_requests\\s+SET status = 'pending'").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectBegin()
				mock.ExpectQuery("FROM assistant_requests\\s+WHERE status = 'pending'").
					WithArgs(1).
					WillReturnError(errors.New("query failed"))
			},
		},
		{
			name: "mark running",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE assistant_requests\\s+SET status = 'pending'").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectBegin()
				mock.ExpectQuery("FROM assistant_requests\\s+WHERE status = 'pending'").
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "request_type", "related_type", "related_id", "payload"}).
						AddRow("req-1", "assistant.ask", "document", "doc-1", `{}`))
				mock.ExpectExec("UPDATE assistant_requests\\s+SET status = 'running'").
					WithArgs(sqlmock.AnyArg()).
					WillReturnError(errors.New("update failed"))
			},
		},
		{
			name: "commit",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE assistant_requests\\s+SET status = 'pending'").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectBegin()
				mock.ExpectQuery("FROM assistant_requests\\s+WHERE status = 'pending'").
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "request_type", "related_type", "related_id", "payload"}).
						AddRow("req-1", "assistant.ask", "document", "doc-1", `{}`))
				mock.ExpectExec("UPDATE assistant_requests\\s+SET status = 'running'").
					WithArgs(sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newQueueMockDB(t)
			consumer := NewConsumer(db)
			tc.setup(mock)

			got := consumer.Poll(context.Background(), 1)
			if got != nil {
				t.Fatalf("messages = %#v, want nil on %s failure", got, tc.name)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet sql expectations: %v", err)
			}
		})
	}
}
