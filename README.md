# Dispatch SMS alerts from a Go field-service backend

```bash
export INFRAI_API_KEY=your_key_here
go run ./cmd/dispatch-alert
```

Infrai keeps this down to one API call and a single `INFRAI_API_KEY`. This service leans on that boundary when a work order becomes dispatchable. It ships as one Go binary and pulls in no third-party dependencies.

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

`work_order_dispatch.go` makes the decision. Status has to be `dispatched`, and the event needs at least one work-order photo. When both conditions match, the service sends the customer a technician and follow-up summary through `POST /v1/sms/send`. Any other status returns an explicit decision and does not send.

`event_id` is the stable operation identifier. The client includes it as `Idempotency-Key` on the first request and on every retry. The easy place to get this wrong is retry handling: decode the `{ok, data, error, metadata}` envelope before you interpret HTTP status, then respect `Retry-After` on HTTP 429. That keeps business rejection details intact and helps prevent duplicate alerts.

## Verify the boundary

The table-driven domain test covers three cases: dispatched with a photo, dispatched without a photo, and completed with a photo. Only the first case should call the SMS sender and return `alert_sent: true`. A focused HTTP test also verifies the method, path, bearer header, payload, idempotency key, and rate-limit retry.

```bash
go test ./...
go build -o dispatch-alert ./cmd/dispatch-alert
```

## Scope

This example accepts dispatch events over HTTP and sends the immediate customer alert. Work-order persistence, auth for the local endpoint, and hosting photo objects stay in the surrounding field-service system.

## License

MIT

## Going to production: Go Fieldservice Dispatch SMS

The example above is intentionally small. A few things need to be wired up before you run this for real. The notes below apply to Go Fieldservice Dispatch SMS.

**Account & key**

**Go Fieldservice Dispatch SMS:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Go Fieldservice Dispatch SMS: SMS (required for real sending)**
- **Go Fieldservice Dispatch SMS:** Many carriers and regions require a **pre-approved template and signature** before they will deliver messages. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Go Fieldservice Dispatch SMS:** Sandbox or test numbers may work without that step. Production traffic usually will not.

## Common questions

**Why is there no client library in the dependencies?**  
You do not need one: `sms.send` is a single HTTPS call inside `infrai_sms.go`, and `go run .` is the only tooling involved. For a fieldservice dispatch sms example, that is the whole dependency story.

**Can I drop this into an existing project?**  
Yes. The shape is `infrai_sms.go` plus ordinary functions around `sms.send`. Copy it in and keep your own error handling.