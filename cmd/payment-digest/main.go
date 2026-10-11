package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	digest "example.com/fintech-risk-digest"
)

type service struct {
	infrai *digest.InfraiClient
	store  *digest.EventStore
}

func main() {
	client, err := digest.NewInfraiClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if command == "schedule" {
		taskURL := os.Getenv("DIGEST_TASK_URL")
		if taskURL == "" {
			log.Fatal("DIGEST_TASK_URL is required")
		}
		job, err := client.CreateDigestCron(context.Background(), "0 9 * * 1-5", taskURL, "fintech-weekday-digest-v1")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("scheduled job:", job.JobID)
		return
	}
	if command != "serve" {
		log.Fatalf("unknown command %q (use serve or schedule)", command)
	}

	svc := &service{infrai: client, store: &digest.EventStore{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /payments", svc.recordPayment)
	mux.HandleFunc("POST /digest", svc.sendDigest)
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("payment digest listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func (s *service) recordPayment(w http.ResponseWriter, r *http.Request) {
	var event digest.PaymentEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "invalid payment event", http.StatusBadRequest)
		return
	}
	if event.EventID == "" || event.Subscriber == "" || event.AmountCents <= 0 || event.RiskScore < 0 || event.RiskScore > 100 {
		http.Error(w, "event_id, subscriber, positive amount_cents, and risk_score 0-100 are required", http.StatusBadRequest)
		return
	}
	s.store.Add(event)
	w.WriteHeader(http.StatusAccepted)
}

func (s *service) sendDigest(w http.ResponseWriter, r *http.Request) {
	events := s.store.Snapshot()
	batch := digest.BuildPaymentDigest(events)
	if len(batch.Messages) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	result, err := s.infrai.SendDigestBatch(ctx, batch.Messages, batch.IdempotencyKey)
	if err != nil {
		status := http.StatusBadGateway
		var apiErr *digest.InfraiError
		if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
			status = apiErr.HTTPStatus
		}
		http.Error(w, err.Error(), status)
		return
	}
	s.store.Commit(len(events))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"message_id": result.MessageID, "recipients": len(batch.Messages)})
}
