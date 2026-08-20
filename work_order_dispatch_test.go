package dispatchalerts

import (
	"context"
	"testing"
)

type recordingSender struct {
	calls int
	input SendSMSInput
	key   string
}

func (s *recordingSender) Send(_ context.Context, input SendSMSInput, key string) (SendSMSResult, error) {
	s.calls++
	s.input = input
	s.key = key
	return SendSMSResult{MessageID: "msg_42"}, nil
}

func TestDispatchDecision(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		photos    []WorkOrderPhoto
		wantCalls int
		wantSent  bool
		decision  string
	}{
		{name: "dispatch with evidence sends", status: "dispatched", photos: []WorkOrderPhoto{{ID: "photo-1", PhotoURL: "https://photos.example/1.jpg"}}, wantCalls: 1, wantSent: true, decision: "dispatch alert sent"},
		{name: "dispatch waits for evidence", status: "dispatched", decision: "awaiting work-order photo"},
		{name: "completed does not notify", status: "completed", photos: []WorkOrderPhoto{{ID: "photo-1"}}, decision: "status does not notify"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &recordingSender{}
			dispatcher := Dispatcher{SMS: sender}
			got, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
				WorkOrderID: "WO-1042", EventID: "evt-7", Status: tt.status,
				CustomerPhone: "+15551234567", TechnicianName: "Mina", FollowUp: "arrival in 30 minutes", Photos: tt.photos,
			})
			if err != nil {
				t.Fatalf("Dispatch() error = %v", err)
			}
			if sender.calls != tt.wantCalls || got.AlertSent != tt.wantSent || got.Decision != tt.decision {
				t.Fatalf("calls=%d sent=%v decision=%q", sender.calls, got.AlertSent, got.Decision)
			}
			if tt.wantSent && (sender.key != "work-order-event:evt-7" || got.MessageID != "msg_42") {
				t.Fatalf("key=%q message_id=%q", sender.key, got.MessageID)
			}
		})
	}
}
