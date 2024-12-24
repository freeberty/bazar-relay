package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fiatjaf/eventstore/postgresql"
)

func getAuction(db *postgresql.PostgresBackend, auctionId string) (Auction, error) {
	var auction Auction
	result := db.DB.QueryRow(`SELECT content FROM public.event WHERE id = $1 AND kind = 33222`, auctionId)
	if result == nil {
		return auction, errors.New(fmt.Sprintf("failed to fetch auction %s", auctionId))
	}

	var auctionJson string
	err := result.Scan(&auctionJson)
	if (err != nil) || (auctionJson == "") {
		return auction, errors.New(fmt.Sprintf("failed to find auction %s", auctionId))
	}

	if err := json.NewDecoder(strings.NewReader(auctionJson)).Decode(&auction); err != nil {
		return auction, errors.New(err.Error())
	}
	return auction, nil
}

func getTopBid(db *postgresql.PostgresBackend, auctionId string) (int64, error) {
	var topBid int64
	m := `SELECT CAST (content as integer) as topBid FROM public.event WHERE exists (
  select 1 from jsonb_array_elements(tags) tag where tag = '["e","%s","root"]'::jsonb AND kind = 1077
  ) ORDER BY topBid DESC LIMIT 1`
	q := fmt.Sprintf(m, auctionId)
	err := db.DB.QueryRow(q).Scan(&topBid)
	if err != nil {
		if err == sql.ErrNoRows {
			// No bids found - return 0 without error
			return 0, nil
		}
		return topBid, errors.New(fmt.Sprintf("failed to fetch topBid for auction %s", auctionId))
	}

	return topBid, nil
}
