package errors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/A-pen-app/logging"
	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	if err := logging.Initialize(nil); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

var errTestPaidNotRecorded = Define("TEST_PAID_NOT_RECORDED", http.StatusInternalServerError, "paid but not recorded")

func serve(ctx context.Context, err error) (int, HttpError) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)

	Handle(func(*gin.Context) error { return err })(c)

	var body HttpError
	json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body
}

func TestHandleDefinedError(t *testing.T) {
	status, body := serve(context.Background(), Wrap(errTestPaidNotRecorded, "intent_id", "i1", "transaction_id", "t1"))

	if status != http.StatusInternalServerError || body.Code != "TEST_PAID_NOT_RECORDED" {
		t.Errorf("got %d %s, want 500 TEST_PAID_NOT_RECORDED", status, body.Code)
	}
	if body.Message != "paid but not recorded" {
		t.Errorf("message = %q", body.Message)
	}
	if len(body.Details) != 2 || body.Details["intent_id"] != "i1" || body.Details["transaction_id"] != "t1" {
		t.Errorf("details = %v", body.Details)
	}
}

func TestHandleClientDisconnect(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"cancellation", fmt.Errorf("verify: %w", context.Canceled), clientClosedRequest, string(KeyClientClosed)},
		{"defined 5xx", errTestPaidNotRecorded, http.StatusInternalServerError, "TEST_PAID_NOT_RECORDED"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, body := serve(canceledCtx(), c.err)
			if status != c.wantStatus || body.Code != c.wantCode {
				t.Errorf("got %d %s, want %d %s", status, body.Code, c.wantStatus, c.wantCode)
			}
		})
	}
}
