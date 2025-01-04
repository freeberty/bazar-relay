package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fiatjaf/eventstore/postgresql"
	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/policies"
	"github.com/kelseyhightower/envconfig"
	"github.com/nbd-wtf/go-nostr"
	"github.com/zebedeeio/go-sdk"
)

var (
	zbd    *zebedee.Client
	config RelayConfig
)

type RelayConfig struct {
	PostgresDatabase string `envconfig:"POSTGRESQL_DATABASE_URL"`
	TicketPriceSats  int64  `envconfig:"TICKET_PRICE_SATS"`
	ZbdApiKey        string `envconfig:"ZBD_API_KEY" required:"true"`
	RelayUrl         string `envconfig:"RELAY_URL"`
	NostrPrivateKey  string `envconfig:"NOSTR_PRIVATE_KEY"`
}

func main() {
	if err := envconfig.Process("", &config); err != nil {
		log.Fatalf("failed to read from env: %v", err)
		os.Exit(1)
		return
	}

	// Create relay instance
	relay := khatru.NewRelay()

	// Set basic properties
	relay.Info.Name = "JustBazar Relay"
	relay.Info.Description = "Relay to store paid auction and bid events not nip-15"

	// Initialize database
	db := postgresql.PostgresBackend{DatabaseURL: config.PostgresDatabase}
	if err := db.Init(); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
		return
	}

	// Add justbazar client ip to whitelist
	AddIpToWhitelist("66.241.124.167")
	AddIpToWhitelist("2a09:8280:1::58:c4c5:0")
	AddIpToWhitelist("2605:4c40:197:d7f9:0:e031:74d:1")

	// Apply database handlers
	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)
	relay.CountEvents = append(relay.CountEvents, db.CountEvents)
	relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent)

	// Apply reject policies
	relay.RejectEvent = append(relay.RejectEvent,
		policies.RejectEventsWithBase64Media,
		policies.EventIPRateLimiter(2, time.Minute*3, 10),
	)

	relay.RejectFilter = append(relay.RejectFilter,
		policies.NoComplexFilters,
		policies.FilterIPRateLimiter(20, time.Minute, 100),
	)

	relay.RejectConnection = append(relay.RejectConnection,
		WithIpWhitelisted(policies.ConnectionRateLimiter(1, time.Minute*5, 100)),
	)

	// Custom event validation
	relay.RejectEvent = append(relay.RejectEvent,
		func(ctx context.Context, evt *nostr.Event) (reject bool, msg string) {
			if config.TicketPriceSats > 0 {
				return validateEventPaid(ctx, evt, relay)
			}
			switch evt.Kind {
			case 33222: // Create auction
				return validateAuctionEvent(evt)
			case 1077: // Make bid (should be 1021)
				return validateBidEvent(&db, evt)
			}
			return true, ""
		},
	)

	// Used to accept payment for events
	zbd = zebedee.New(config.ZbdApiKey)

	// Set up HTTP handlers
	mux := relay.Router()
	mux.HandleFunc("/pay-for-event", handleEventPayment())
	mux.HandleFunc("/payment-update/{hash}", handlePaymentUpdate(relay))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html")
		fmt.Fprintf(w, `<b>welcome</b> to bazar relay!`)
	})

	// Start server
	fmt.Println("Running auction relay on :3334")
	if err := http.ListenAndServe(":3334", relay); err != nil {
		log.Fatalf("server terminated: %v", err)
	}
}
