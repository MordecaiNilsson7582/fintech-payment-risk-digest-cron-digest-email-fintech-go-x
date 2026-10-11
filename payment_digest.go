package digest

import (
	"crypto/sha256"
	"fmt"
	"html"
	"sort"
	"strings"
	"sync"
	"time"
)

type PaymentStatus string

const (
	PaymentSettled  PaymentStatus = "settled"
	PaymentDeclined PaymentStatus = "declined"
	PaymentReview   PaymentStatus = "review"
)

type PaymentEvent struct {
	EventID        string        `json:"event_id"`
	Subscriber     string        `json:"subscriber"`
	AccountID      string        `json:"account_id"`
	AmountCents    int64         `json:"amount_cents"`
	Currency       string        `json:"currency"`
	Status         PaymentStatus `json:"status"`
	RiskScore      int           `json:"risk_score"`
	OccurredAt     time.Time     `json:"occurred_at"`
	DigestEligible bool          `json:"digest_eligible"`
}

type Digest struct {
	Messages       []EmailMessage
	IdempotencyKey string
}

func IncludeInDigest(event PaymentEvent) bool {
	if !event.DigestEligible || event.EventID == "" || event.Subscriber == "" || event.AmountCents <= 0 {
		return false
	}
	if event.Status == PaymentReview || event.Status == PaymentDeclined {
		return true
	}
	return event.Status == PaymentSettled && event.RiskScore >= 70
}

func BuildPaymentDigest(events []PaymentEvent) Digest {
	grouped := make(map[string][]PaymentEvent)
	for _, event := range events {
		if IncludeInDigest(event) {
			grouped[event.Subscriber] = append(grouped[event.Subscriber], event)
		}
	}

	recipients := make([]string, 0, len(grouped))
	for recipient := range grouped {
		recipients = append(recipients, recipient)
	}
	sort.Strings(recipients)

	messages := make([]EmailMessage, 0, len(recipients))
	fingerprint := sha256.New()
	for _, recipient := range recipients {
		rows := grouped[recipient]
		sort.Slice(rows, func(i, j int) bool { return rows[i].EventID < rows[j].EventID })
		var body strings.Builder
		body.WriteString("<h1>Payment risk digest</h1><table><thead><tr><th>Event</th><th>Account</th><th>Amount</th><th>Status</th><th>Risk</th><th>Time</th></tr></thead><tbody>")
		for _, event := range rows {
			fmt.Fprintf(&body, "<tr><td>%s</td><td>%s</td><td>%d %s</td><td>%s</td><td>%d</td><td>%s</td></tr>",
				html.EscapeString(event.EventID), html.EscapeString(event.AccountID), event.AmountCents,
				html.EscapeString(event.Currency), html.EscapeString(string(event.Status)), event.RiskScore,
				event.OccurredAt.UTC().Format(time.RFC3339))
			fmt.Fprintf(fingerprint, "%s\x00%s\x00%d\x00%s\n", recipient, event.EventID, event.AmountCents, event.Status)
		}
		body.WriteString("</tbody></table>")
		messages = append(messages, EmailMessage{To: recipient, Subject: "Payment risk digest", HTML: body.String()})
	}
	return Digest{Messages: messages, IdempotencyKey: fmt.Sprintf("payment-digest-%x", fingerprint.Sum(nil))}
}

type EventStore struct {
	mu      sync.Mutex
	pending []PaymentEvent
}

func (s *EventStore) Add(event PaymentEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = append(s.pending, event)
}

func (s *EventStore) Snapshot() []PaymentEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]PaymentEvent(nil), s.pending...)
}

func (s *EventStore) Commit(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if count > len(s.pending) {
		count = len(s.pending)
	}
	s.pending = append([]PaymentEvent(nil), s.pending[count:]...)
}
