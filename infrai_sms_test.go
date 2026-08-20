package dispatchalerts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSMSClientRetriesRateLimitWithSameRequest(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost || r.URL.Path != "/v1/sms/send" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" || r.Header.Get("Idempotency-Key") != "work-order-event:evt-7" {
			t.Errorf("request headers missing")
		}
		var body SendSMSInput
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.To != "+15551234567" || body.Message != "Technician dispatched" {
			t.Errorf("body = %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"ok":false,"data":{},"error":{"code":"RATE_LIMITED","message":"retry later"},"metadata":{}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"data":{"message_id":"msg_42"},"error":null,"metadata":{}}`))
	}))
	defer server.Close()

	client := SMSClient{
		BaseURL: server.URL, APIKey: "test-key", MaxRetries: 1,
		Sleep: func(_ context.Context, delay time.Duration) error {
			if delay != 0 {
				t.Errorf("retry delay = %s", delay)
			}
			return nil
		},
	}
	got, err := client.Send(context.Background(), SendSMSInput{To: "+15551234567", Message: "Technician dispatched"}, "work-order-event:evt-7")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if requests != 2 || got.MessageID != "msg_42" {
		t.Fatalf("requests=%d message_id=%q", requests, got.MessageID)
	}
}
