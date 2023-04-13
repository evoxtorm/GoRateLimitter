package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

const maxTokens = 10

type RequestBody struct {
	UserId   string                 `json:"user_id"`
	Endpoint string                 `json:"endpoint`
	Limit    int                    `json: limit`
	Body     map[string]interface{} `json: body`
}

type rateLimitterStruct struct {
	// tokens      int
	// fillRate    float64
	lastUpdated time.Time
	// resetTicker *time.Ticker
	requests int
}

var rl = newRateLimitter()

// func newRateLimitter(tokens int, fillRate float64) *rateLimitterStruct {
func newRateLimitter() *rateLimitterStruct {
	rateLimitterVal := &rateLimitterStruct{
		// tokens:      tokens,
		// fillRate:    fillRate,
		lastUpdated: time.Now(),
		requests:    0,
		// resetTicker: time.NewTicker(time.Minute),
	}
	ticker := time.NewTicker(time.Minute)
	go func() {
		for range ticker.C {
			rateLimitterVal.requests = 0
			rateLimitterVal.lastUpdated = time.Now()
		}
	}()
	return rateLimitterVal
}

func (r *rateLimitterStruct) allowRequest() bool {
	// now := time.Now()
	// timeElapsed := now.Sub(r.lastUpdated)
	// if timeElapsed.Minutes() >= 1 {
	// 	r.requests = 0
	//     r.lastUpdated = now
	// }
	// r.tokens += int(r.fillRate * now.Sub(r.lastUpdated).Seconds())
	// if r.tokens > maxTokens {
	// 	r.tokens = maxTokens
	// 	return false
	// }
	if r.requests < maxTokens {
		r.requests++
		return true
	}
	// r.lastUpdated = now
	// if r.tokens > 0 {
	// 	r.tokens--
	// 	return true
	// }
	return false
}

func main() {

	router := mux.NewRouter()

	router.HandleFunc("/", homeHandler)
	router.HandleFunc("/limitrequest", rateLimitter).Methods("POST")
	fmt.Println("Server started on port 8080")
	http.ListenAndServe(":8080", router)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Welcome to the homepage!")
}

func rateLimitter(w http.ResponseWriter, r *http.Request) {
	var requestBody RequestBody
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if rl.allowRequest() {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Request processed successfully"))
	} else {
		http.Error(w, "Rate limit exceeded, Please try after some time", http.StatusTooManyRequests)
		return
	}
}
