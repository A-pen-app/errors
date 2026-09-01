package errors

import (
	"context"
	"errors"
	"fmt"
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
