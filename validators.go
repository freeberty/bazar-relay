package main

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"github.com/fiatjaf/eventstore/postgresql"
	"github.com/fiatjaf/khatru"
	"github.com/nbd-wtf/go-nostr"
)

func validateAuctionEvent(evt *nostr.Event) (bool, string) {
	var auction Auction
	if err := json.NewDecoder(strings.NewReader(evt.Content)).Decode(&auction); err != nil {
		return true, "invalid auction format"
	}

	return false, ""
}

func validateBidEvent(db *postgresql.PostgresBackend, evt *nostr.Event) (bool, string) {
	tag := evt.Tags.GetFirst([]string{"e", ""})

	auction, err := getAuction(db, (*tag)[1])
	if err != nil {
		log.Print("error on auction fetch")
		return true, "error on auction fetch"
	}

	if evt.CreatedAt > nostr.Timestamp(auction.FinishTimestamp) {
		log.Print("debug: auction closed")
		return true, "auction closed"
	}

	// Check bid size
	topBid, err := getTopBid(db, (*tag)[1])
	if err != nil {
		log.Printf("error on top bid check %s", err)
		return true, "error on top bid check"
	}
	newBid, newBidErr := strconv.ParseInt(evt.Content, 10, 64)
	if newBidErr != nil {
		log.Printf("error on bid content conversion %s", newBidErr)
		return true, "error on bid content conversion"
	}

	if newBid < (topBid + auction.BidStepSats) {
		log.Print("debug: new bid should be higher")
		return true, "new bid should be higher"
	}
	return false, ""
}

func validateEventPaid(ctx context.Context, evt *nostr.Event, relay *khatru.Relay) (bool, string) {
	var paidEvent int64 = 0
	if len(relay.CountEvents) > 0 {
		paidEvent, _ = relay.CountEvents[0](ctx, nostr.Filter{
			Kinds: []int{1888},
			Tags:  nostr.TagMap{"e": []string{evt.ID}},
		})
	}
	if paidEvent == 0 {
		log.Print("debug: event not paid")
		return true, "event not paid"
	}
	return false, ""
}
