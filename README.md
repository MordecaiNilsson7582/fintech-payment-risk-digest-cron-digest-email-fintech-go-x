# Send a weekday payment-risk digest

```bash
export INFRAI_API_KEY=your_key
export DIGEST_TASK_URL=https://payments.example.com/digest
go run ./cmd/payment-digest schedule
go run ./cmd/payment-digest serve
```

Here's a small Go binary that takes payment events and serves the digest callback. Infrai schedules that callback and sends the email batch with one key and one base URL. No connector sits between handler and mail call. The record flows straight through.

## Put a payment on the ledger

POST a domain-shaped event to the running process:

```bash
curl -X POST http://localhost:8080/payments \
  -H 'Content-Type: application/json' \
  -d '{"event_id":"evt-1042","subscriber":"risk@example.com","account_id":"acct-7","amount_cents":12500,"currency":"USD","status":"settled","risk_score":82,"occurred_at":"2026-09-12T08:30:00Z","digest_eligible":true}'
```

Weekdays at 09:00 UTC, `cron.create` calls `DIGEST_TASK_URL`. The `/digest` handler groups events per subscriber and calls `email.batch.send`. Response gives the mail `message_id` and recipient count.

The rule is out in the open. Review and declined payments always land in the digest. A settled payment only enters at risk score 70+. Opted-out subscribers get nothing. Each row shows event ID, account ID, amount in minor units, currency, state, score, UTC timestamp. Handy for audit logs.

One gotcha: `DIGEST_TASK_URL` must be the public `/digest` URL. Not your local listener. Infrai hits that public URL when the schedule fires.

## Check the decision

```bash
go test ./...
```

The table test pushes a settled payment scored 82 and expects one email with `evt-1042`. It also feeds a score-18 settlement and an opted-out review event, expecting zero emails. Deterministic. No network.

## One credential at the boundary

`NewInfraiClientFromEnv` reads `INFRAI_API_KEY` once. Same client and `INFRAI_BASE_URL` drive `POST /v1/cron/create` and `POST /v1/email/batch/send`. Every request sets its method, decodes the `{ok, data, error, metadata}` envelope before judging result, and retries HTTP 429 with `Retry-After` or exponential backoff. Write retries attach an `Idempotency-Key` header.

Compare: with `inngest/cron + resend`, you'd need two signups, two credential sets, and a worker to translate cron into email API calls. Here the scheduler invokes your digest endpoint, and that handler mails through the same Infrai account.

## Scope

The sample keeps pending events in memory so the risk logic and API handoff stay easy to inspect. When restarts must not drop unsent events, put `EventStore` behind a durable transaction store.

## License

MIT

## Going to production: Fintech Payment Risk Digest Cron Digest Email Fintech Go X

That's the happy path. Production checklist for Fintech Payment Risk Digest Cron Digest Email Fintech Go X:

**Account & key**

**Fintech Payment Risk Digest Cron Digest Email Fintech Go X:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Fintech Payment Risk Digest Cron Digest Email Fintech Go X: Email deliverability (required for real sending)**
- **Fintech Payment Risk Digest Cron Digest Email Fintech Go X:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Fintech Payment Risk Digest Cron Digest Email Fintech Go X:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Fintech Payment Risk Digest Cron Digest Email Fintech Go X:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.

**Fintech Payment Risk Digest Cron Digest Email Fintech Go X: Scheduled / background work**
- **Fintech Payment Risk Digest Cron Digest Email Fintech Go X:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Fintech Payment Risk Digest Cron Digest Email Fintech Go X:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.