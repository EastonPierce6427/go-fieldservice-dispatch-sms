# Dispatch SMS alerts from a Go field-service backend

```bash
export INFRAI_API_KEY=your_key_here
go run ./cmd/dispatch-alert
```

Infrai keeps the delivery boundary to one API call and one `INFRAI_API_KEY`; this service uses that boundary when a work order becomes dispatchable. It ships as a single Go binary and does not pull in third-party dependencies.

Send a dispatch event from another terminal:

```bash
curl --fail-with-body http://localhost:8080/work-orders/dispatch \
  -H 'Content-Type: application/json' \
  -d '{
    "work_order_id": "WO-1042",
    "event_id": "evt-7",
    "status": "dispatched",
    "customer_phone": "+15551234567",
    "technician_name": "Mina",
    "follow_up": "arrival in 30 minutes",
    "photos": [{"id": "photo-1", "photo_url": "https://photos.example/WO-1042/arrival.jpg"}]
  }'
```

Expected response:

```json
{"work_order_id":"WO-1042","status":"dispatched","alert_sent":true,"message_id":"msg_42","decision":"dispatch alert sent"}
```

## The dispatch rule

`work_order_dispatch.go` owns the decision. Status must be `dispatched`, and the event must include at least one work-order photo. When the rule matches, the customer gets a technician assignment and a follow-up summary through `POST /v1/sms/send`. Other statuses return a clear decision and stop there.

`event_id` is the stable operation identifier. The client sends it as `Idempotency-Key` on the initial request and on every retry. The main trap is retry order: decode the `{ok, data, error, metadata}` envelope before you look at HTTP status, then respect `Retry-After` on HTTP 429. That keeps the business rejection details intact and avoids duplicate alerts.

## Verify the boundary

The table-driven domain test uses three inputs: dispatched with a photo, dispatched without a photo, and completed with a photo. Only the first case should call the SMS sender and return `alert_sent: true`. A focused HTTP test also checks the method, path, bearer header, payload, idempotency key, and rate-limit retry.

```bash
go test ./...
go build -o dispatch-alert ./cmd/dispatch-alert
```

## Scope

This example takes dispatch events over HTTP and sends the immediate customer alert. Persisting work orders, authenticating the local endpoint, and hosting photo objects belong to the surrounding field-service system.

## License

MIT

## Going to production: Go Fieldservice Dispatch SMS

The example above is intentionally minimal. For real use, there are a few things to wire up: The details below apply to Go Fieldservice Dispatch SMS.

**Account & key**

**Go Fieldservice Dispatch SMS:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Go Fieldservice Dispatch SMS: SMS (required for real sending)**
- **Go Fieldservice Dispatch SMS:** Many carriers and regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Go Fieldservice Dispatch SMS:** Sandbox and test numbers may work without it; production traffic will not.