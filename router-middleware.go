package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/datasparq-ai/houston/model"
	"golang.org/x/time/rate"
)

// loggingMiddleware sets the logging output file to the relevant file for the request. There is one file per key per
// day. This runs for all requests. If there is no API key then
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		key := r.Header.Get("x-access-key") // key hasn't been checked yet, but if key doesn't exist then it doesn't matter
		SetLoggingFile(key)

		next.ServeHTTP(w, r)
	})
}

// checkKey runs before requests that require a key to check that the key exists in the API database.
// This also sets the logging output file to the relevant file for the request. There is one file per key per day.
func (a *API) checkKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("x-access-key")
		if key == "" {
			err := &model.KeyNotProvidedError{}
			handleError(err, w)
			return
		}
		// check that key exists
		_, ok := a.db.Get(key, "u")
		if !ok {
			err := &model.KeyNotFoundError{}
			handleError(err, w)
			return
		}
		SetLoggingFile(key)
		next.ServeHTTP(w, r)
	})
}

// checkAdminPassword runs before all admin routes
func (a *API) checkAdminPassword(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.config.Password == "" {
			next.ServeHTTP(w, r) // if API is not password protected - do nothing
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok {
			handleError(&model.BadCredentialsError{}, w)
			return
		}

		if username != "admin" {
			handleError(&model.BadCredentialsError{}, w)
			return
		}
		// check that password matches hash of password stored in config
		if a.config.Password == hashPassword(password, a.config.Salt) {
			next.ServeHTTP(w, r)
			return
		} else {
			handleError(&model.BadCredentialsError{}, w)
			return
		}
	})
}

//
// Rate Limiter
//

type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	lst map[string]time.Time // last event time for each ip
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	i := &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		lst: make(map[string]time.Time),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}

	return i
}

// AddIP creates a new rate limiter and adds it to the ips map,
// using the IP address as the key
func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter := rate.NewLimiter(i.r, i.b)

	i.ips[ip] = limiter

	return limiter
}

// CleanUpIPs will delete any rate limiters belonging to IPs that haven't been seen in over a minute
func (i *IPRateLimiter) CleanUpIPs() {
	for {
		time.Sleep(time.Minute)
		i.mu.Lock()
		for ip, t := range i.lst {
			if time.Since(t) > 1*time.Minute {
				delete(i.ips, ip)
				delete(i.lst, ip)
			}
		}
		i.mu.Unlock()
	}
}

// GetLimiter returns the rate limiter for the provided IP address if it exists.
// Otherwise, calls AddIP to add IP address to the map
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	limiter, exists := i.ips[ip]

	i.lst[ip] = time.Now()

	if !exists {
		i.mu.Unlock()
		return i.AddIP(ip)
	}

	i.mu.Unlock()

	return limiter
}

// There is one rate limiter object shared by all API instances for simplicity. No individual IP address is allowed
// to make more than 100 requests per second or 500 requests in a burst
var limiter = NewIPRateLimiter(100, 500)

// rateLimit middleware checks the rate of requests for each IP seen and returns 429 if the rate limit is exceeded
func rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := limiter.GetLimiter(strings.Split(r.RemoteAddr, ":")[0])
		if !limiter.Allow() {
			fmt.Println("Client at", strings.Split(r.RemoteAddr, ":")[0], "has made too many requests! Request rate is being limited.")

			var err model.TooManyRequestsError
			handleError(&err, w)
			return
		}

		next.ServeHTTP(w, r)
	})
}
