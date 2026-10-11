package digest

import (
	"strings"
	"testing"
	"time"
)

func TestBuildPaymentDigest(t *testing.T) {
	when := time.Date(2026, 9, 12, 8, 30, 0, 0, time.UTC)
	tests := []struct {
		name      string
		events    []PaymentEvent
		wantCount int
		wantText  string
	}{
		{
			name: "includes a high-risk settlement for an opted-in subscriber",
			events: []PaymentEvent{{
				EventID: "evt-1042", Subscriber: "risk@example.com", AccountID: "acct-7",
				AmountCents: 12500, Currency: "USD", Status: PaymentSettled, RiskScore: 82,
				OccurredAt: when, DigestEligible: true,
			}},
			wantCount: 1,
			wantText:  "evt-1042",
		},
		{
			name: "excludes a low-risk settlement",
			events: []PaymentEvent{{
				EventID: "evt-1043", Subscriber: "risk@example.com", AccountID: "acct-8",
				AmountCents: 4500, Currency: "USD", Status: PaymentSettled, RiskScore: 18,
				OccurredAt: when, DigestEligible: true,
			}},
			wantCount: 0,
		},
		{
			name: "excludes a subscriber who opted out",
			events: []PaymentEvent{{
				EventID: "evt-1044", Subscriber: "quiet@example.com", AccountID: "acct-9",
				AmountCents: 9900, Currency: "USD", Status: PaymentReview, RiskScore: 91,
				OccurredAt: when, DigestEligible: false,
			}},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildPaymentDigest(tt.events)
			if len(got.Messages) != tt.wantCount {
				t.Fatalf("got %d messages, want %d", len(got.Messages), tt.wantCount)
			}
			if tt.wantText != "" && !strings.Contains(got.Messages[0].HTML, tt.wantText) {
				t.Errorf("digest does not contain %q", tt.wantText)
			}
			if got.IdempotencyKey == "" {
				t.Fatal("idempotency key is empty")
			}
		})
	}
}
