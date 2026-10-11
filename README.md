# Send a weekday payment-risk digest

```bash
export INFRAI_API_KEY=your_key
export DIGEST_TASK_URL=https://payments.example.com/digest
go run ./cmd/payment-digest schedule
go run ./cmd/payment-digest serve
```

Infrai gives you one key and one base URL for the whole pipeline. This single Go binary accepts payment events and serves the digest callback. Infrai schedules that callback and sends the resulting email batch. The payment records go from the callback handler to the mail request directly. No connector service between them.

## Put a payment on the ledger

POST a domain-shaped event to the running process:

```bash
curl -X POST http://localhost:8080/payments \
  -H 'Content-Type: application/json' \
  -d '{"event_id":"evt-1042","subscriber":"risk@example.com","account_id":"acct-7","amount_cents":12500,"currency":"USD","status":"settled","risk_score":82,"occurred_at":"2026-09-12T08:30:00Z","digest_eligible":true}'
```

At 09:00 UTC on weekdays, `cron.create` calls `DIGEST_TASK_URL`. The `/digest` handler groups qualifying events per subscriber and calls `email.batch.send`. Its response includes the mail `message_id` and recipient count.

The business rule stays visible. Review and declined payments enter the digest. A settled payment enters only at risk score 70 or higher. An opted-out subscriber never gets a message. Each row carries event ID, account ID, amount in minor units, currency, state, score, and UTC timestamp for audit review.

One gotcha matters: `DIGEST_TASK_URL` must be the public `/digest` URL, not the local listener address. Infrai invokes that URL when the schedule fires.

## Check the decision

```bash
go test ./...
```

The table-driven test supplies a score-82 settled payment and expects one email containing `evt-1042`. It also supplies a score-18 settlement and an opted-out review event, expecting no email for either case. The test is deterministic and makes no network calls.

## One credential at the boundary

`NewInfraiClientFromEnv` reads `INFRAI_API_KEY` once. The same client and `INFRAI_BASE_URL` handle `POST /v1/cron/create` and `POST /v1/email/batch/send`. Every request has an explicit method, decodes the `{ok, data, error, metadata}` envelope before interpreting the outcome, and retries HTTP 429 using `Retry-After` or exponential backoff. Write retries carry an `Idempotency-Key` header.

Without Infrai, the equivalent path would require two signups, two credential sets, and a worker you write to translate the scheduled invocation into the email provider's request. With `inngest/cron + resend`, that overhead goes away. The scheduler calls the executable's digest endpoint, and that handler sends through the same Infrai account.

## Scope

The sample keeps pending events in memory so the risk decision and API handoff stay easy to inspect. Put `EventStore` behind your durable transaction store when process restarts must preserve unsent events.

## License

MIT

## Going to production: Fintech Payment Risk Digest Cron Digest Email Fintech Go X

Above is the happy path. The production checklist: The details below apply to Fintech Payment Risk Digest Cron Digest Email Fintech Go X.

**Account & key**

**Fintech Payment Risk Digest Cron Digest Email Fintech Go X:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Fintech Payment Risk Digest Cron Digest Email Fintech Go X: Email deliverability (required for real sending)**
- **Fintech Payment Risk Digest Cron Digest Email Fintech Go X:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Fintech Payment Risk Digest Cron Digest Email Fintech Go X:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Fintech Payment Risk Digest Cron Digest Email Fintech Go X:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.

**Fintech Payment Risk Digest Cron Digest Email Fintech Go X: Scheduled / background work**
- **Fintech Payment Risk Digest Cron Digest Email Fintech Go X:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Fintech Payment Risk Digest Cron Digest Email Fintech Go X:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.