package errors

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// sqlStateErr stands in for the driver errors that carry a SQLSTATE.
type sqlStateErr struct {
	state string
}

func (e *sqlStateErr) Error() string { return "pq: " + e.state }

func (e *sqlStateErr) SQLState() string { return e.state }

func canceledCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func expiredCtx() context.Context {
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(0, 0))
	cancel()
	return ctx
}

func TestIsClientCancellation(t *testing.T) {
	cases := []struct {
		name string
		ctx  context.Context
		err  error
		want bool
	}{
		{"live request", context.Background(), context.Canceled, false},
		{"disconnect", canceledCtx(), context.Canceled, true},
		{"disconnect, wrapped", canceledCtx(), fmt.Errorf("list users: %w", context.Canceled), true},
		{"disconnect, behind AppError", canceledCtx(), Wrap(context.Canceled, "user_id", "u1"), true},
		{"disconnect, query canceled", canceledCtx(), &sqlStateErr{state: sqlStateQueryCanceled}, true},
		{"disconnect, server fault alongside", canceledCtx(), errors.New("boom"), false},
		{"disconnect, unique violation", canceledCtx(), &sqlStateErr{state: "23505"}, false},
		{"disconnect, nil error", canceledCtx(), nil, false},
		{"server deadline", expiredCtx(), context.Canceled, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsClientCancellation(c.ctx, c.err); got != c.want {
				t.Errorf("IsClientCancellation(%v, %v) = %v, want %v", c.ctx.Err(), c.err, got, c.want)
			}
		})
	}
}

var errTestDefined = Define("TEST_DEFINED", http.StatusServiceUnavailable, "test defined")

func TestGetErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want ErrorMapping
	}{
		{"defined", errTestDefined, ErrorMapping{"TEST_DEFINED", http.StatusServiceUnavailable}},
		{"defined, wrapped", fmt.Errorf("verify: %w", errTestDefined), ErrorMapping{"TEST_DEFINED", http.StatusServiceUnavailable}},
		{"defined, behind AppError", Wrap(errTestDefined, "intent_id", "i1"), ErrorMapping{"TEST_DEFINED", http.StatusServiceUnavailable}},
		{"defined, behind nested AppError", Wrap(Wrap(errTestDefined, "intent_id", "i1"), "transaction_id", "t1"), ErrorMapping{"TEST_DEFINED", http.StatusServiceUnavailable}},
		{"built-in", ErrorNotFound, ErrorMapping{KeyNotFound, http.StatusNotFound}},
		{"built-in, wrapped", fmt.Errorf("load: %w", ErrorNotFound), ErrorMapping{KeyInternalError, http.StatusInternalServerError}},
		{"unknown", errors.New("boom"), ErrorMapping{KeyInternalError, http.StatusInternalServerError}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := getErrorMapping(c.err); got != c.want {
				t.Errorf("getErrorMapping(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}

func TestDefinePanics(t *testing.T) {
	Define("TEST_TAKEN", http.StatusBadRequest, "taken")

	cases := []struct {
		name   string
		code   ErrorCode
		status int
	}{
		{"empty code", "", http.StatusBadRequest},
		{"blank code", "  ", http.StatusBadRequest},
		{"built-in code", KeyNotFound, http.StatusNotFound},
		{"client closed code", KeyClientClosed, http.StatusBadRequest},
		{"already defined code", "TEST_TAKEN", http.StatusBadRequest},
		{"2xx status", "TEST_OK", http.StatusOK},
		{"status above 5xx", "TEST_600", 600},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("Define(%q, %d) did not panic", c.code, c.status)
				}
			}()
			Define(c.code, c.status, "msg")
		})
	}
}
