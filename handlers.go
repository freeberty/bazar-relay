package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/fiatjaf/khatru"
	"github.com/nbd-wtf/go-nostr"
	"github.com/zebedeeio/go-sdk"
)

func handleEventPayment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if config.TicketPriceSats < 1 {
			json.NewEncoder(w).Encode(struct {
				Error string `json:"error"`
			}{"events are free"})
			return
		}
		params := r.URL.Query()
		eventHash := params.Get("id")
		invoice, err := generateInvoice(eventHash)
		log.Printf("invoice %s", invoice)
		if err != nil {
			json.NewEncoder(w).Encode(struct {
				Error string `json:"error"`
			}{err.Error()})
		} else {
			json.NewEncoder(w).Encode(struct {
				Invoice string `json:"bolt11"`
			}{invoice})
		}
	}
}

func handlePaymentUpdate(relay *khatru.Relay) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Content-Type", "application/json")
		hash := r.URL.Path[len("/payment-update/"):]
		var webhook zebedee.Charge
		err := json.NewDecoder(r.Body).Decode(&webhook)
		log.Printf("hash %s", hash)
		if err != nil {
			log.Printf("got invalid JSON webhook from zbd %s", err)
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(struct {
				Error string `json:"error"`
			}{err.Error()})
		}

		if webhook.Status != "completed" {
			w.WriteHeader(204)
			return
		}

		evt := &nostr.Event{
			CreatedAt: nostr.Now(),
			Kind:      1888, // event kind to store paid event id
			Tags: nostr.Tags{
				nostr.Tag{"e", hash},
				nostr.Tag{"expiration", strconv.FormatInt(86400, 10)},
			},
			Content: "",
		}
		evt.Sign(config.NostrPrivateKey)

		if err := relay.StoreEvent[0](ctx, evt); err != nil {
			log.Printf("error on attempt to store event %s", err)
		}

		w.WriteHeader(200)
	}
}

// Helper function for generating invoices (you'll need to implement this)
func generateInvoice(hash string) (string, error) {
	charge, err := zbd.Charge(&zebedee.Charge{
		ExpiresIn:   3600,
		Amount:      strconv.FormatInt(config.TicketPriceSats*1000, 10),
		CallbackURL: config.RelayUrl + "/payment-update/" + hash,
	})
	log.Printf("%s/payment-update/%s", config.RelayUrl, hash)
	if err != nil {
		return "", err
	}

	return charge.Invoice.Request, nil
}

func writeError(w http.ResponseWriter, err string) {
	json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{err})
}
