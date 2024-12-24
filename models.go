package main

type Auction struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	Images          string `json:"images"`
	StartTimestamp  int64  `json:"startTimestamp"`
	FinishTimestamp int64  `json:"finishTimestamp"`
	BidStepSats     int64  `json:"bidStepSats"`
	Status          string `json:"status"`
}
