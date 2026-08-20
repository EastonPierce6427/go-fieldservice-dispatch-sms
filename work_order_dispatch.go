package dispatchalerts

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type WorkOrderPhoto struct {
	ID       string `json:"id"`
	PhotoURL string `json:"photo_url"`
}

type DispatchRequest struct {
	WorkOrderID    string           `json:"work_order_id"`
	EventID        string           `json:"event_id"`
	Status         string           `json:"status"`
	CustomerPhone  string           `json:"customer_phone"`
	TechnicianName string           `json:"technician_name"`
	FollowUp       string           `json:"follow_up"`
	Photos         []WorkOrderPhoto `json:"photos"`
}

type DispatchResult struct {
	WorkOrderID string `json:"work_order_id"`
	Status      string `json:"status"`
	AlertSent   bool   `json:"alert_sent"`
	MessageID   string `json:"message_id,omitempty"`
	Decision    string `json:"decision"`
}

type SMSSender interface {
	Send(context.Context, SendSMSInput, string) (SendSMSResult, error)
}

type Dispatcher struct {
	SMS SMSSender
}

func (d Dispatcher) Dispatch(ctx context.Context, request DispatchRequest) (DispatchResult, error) {
	result := DispatchResult{WorkOrderID: request.WorkOrderID, Status: request.Status}
	if strings.TrimSpace(request.WorkOrderID) == "" || strings.TrimSpace(request.EventID) == "" {
		return result, errors.New("work_order_id and event_id are required")
	}
	if request.Status != "dispatched" {
		result.Decision = "status does not notify"
		return result, nil
	}
	if len(request.Photos) == 0 {
		result.Decision = "awaiting work-order photo"
		return result, nil
	}
	if strings.TrimSpace(request.CustomerPhone) == "" || strings.TrimSpace(request.TechnicianName) == "" || strings.TrimSpace(request.FollowUp) == "" {
		return result, errors.New("customer_phone, technician_name, and follow_up are required for dispatch")
	}
	message := fmt.Sprintf("Work order %s is dispatched. Technician: %s. Follow-up: %s", request.WorkOrderID, request.TechnicianName, request.FollowUp)
	sent, err := d.SMS.Send(ctx, SendSMSInput{To: request.CustomerPhone, Message: message}, "work-order-event:"+request.EventID)
	if err != nil {
		return result, err
	}
	result.AlertSent = true
	result.MessageID = sent.MessageID
	result.Decision = "dispatch alert sent"
	return result, nil
}
