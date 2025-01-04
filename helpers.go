package main

import (
	"log"
	"net/http"

	"github.com/fiatjaf/khatru"
)

var WhitelistedIPs = make(map[string]bool)

func AddIpToWhitelist(ip string) {
	WhitelistedIPs[ip] = true
}

func RemoveIpFromWhitelist(ip string) {
	delete(WhitelistedIPs, ip)
}

func WithIpWhitelisted(rateLimiter func(r *http.Request) bool) func(r *http.Request) bool {
	return func(r *http.Request) bool {
		ip := khatru.GetIPFromRequest(r)
		log.Printf("connected ip %s", ip)
		if WhitelistedIPs[ip] {
			return true
		}
		return rateLimiter(r)
	}
}
