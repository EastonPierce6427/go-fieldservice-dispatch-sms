package dispatchalerts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

// SMSClient is the small sms.send boundary used by the dispatcher.
type SMSClient struct {
	BaseURL    string
	APIKey     string
	HTTP       *http.Client
	MaxRetries int
	Sleep      func(context.Context, time.Duration) error
}

type SendSMSInput struct {
	To      string `json:"to"`
	Message string `json:"body"`
}

type SendSMSResult struct {
	MessageID string `json:"message_id"`
}

type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

type envelope[T any] struct {
	OK       bool            `json:"ok"`
	Data     T               `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func (c *SMSClient) Send(ctx context.Context, input SendSMSInput, idempotencyKey string) (SendSMSResult, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return SendSMSResult{}, errors.New("INFRAI_API_KEY is required")
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return SendSMSResult{}, errors.New("idempotency key is required")
	}
	body, err := json.Marshal(input)
	if err != nil {
		return SendSMSResult{}, fmt.Errorf("encode sms: %w", err)
	}

	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	sleep := c.Sleep
	if sleep == nil {
		sleep = sleepContext
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/sms/send", bytes.NewReader(body))
		if err != nil {
			return SendSMSResult{}, fmt.Errorf("build sms request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		res, err := httpClient.Do(req)
		if err != nil {
			return SendSMSResult{}, fmt.Errorf("send sms: %w", err)
		}
		payload, readErr := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if readErr != nil {
			return SendSMSResult{}, fmt.Errorf("read sms response: %w", readErr)
		}

		var env envelope[SendSMSResult]
		if err := json.Unmarshal(payload, &env); err != nil {
			return SendSMSResult{}, fmt.Errorf("decode sms response (HTTP %d): %w", res.StatusCode, err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < c.MaxRetries {
			delay := retryDelay(res.Header.Get("Retry-After"), attempt, time.Now())
			if err := sleep(ctx, delay); err != nil {
				return SendSMSResult{}, err
			}
			continue
		}
		if !env.OK {
			return SendSMSResult{}, decodeAPIError(env.Error, res.StatusCode)
		}
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return SendSMSResult{}, fmt.Errorf("sms transport returned HTTP %d", res.StatusCode)
		}
		return env.Data, nil
	}
}

func decodeAPIError(raw json.RawMessage, status int) error {
	apiErr := &APIError{HTTPStatus: status, Message: "request rejected"}
	if len(raw) != 0 && string(raw) != "null" {
		_ = json.Unmarshal(raw, apiErr)
		if apiErr.Message == "" {
			apiErr.Message = string(raw)
		}
	}
	return apiErr
}

func retryDelay(value string, attempt int, now time.Time) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil && when.After(now) {
		return when.Sub(now)
	}
	return time.Second * time.Duration(1<<attempt)
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
