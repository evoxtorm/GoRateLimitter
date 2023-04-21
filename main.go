package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
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
	tokens      int
	fillRate    float64
	lastUpdated time.Time
	// requests    int
}

type rateLimitterMap struct {
	sync.Mutex
	m map[string]*rateLimitterStruct
}

var rlm = newRateLimitterMap()

// func newRateLimitter(tokens int, fillRate float64) *rateLimitterStruct {
func newRateLimitterMap() *rateLimitterMap {
	rateLM := rateLimitterMap{
		m: make(map[string]*rateLimitterStruct),
	}
	ticker := time.NewTicker(time.Minute)
	go func() {
		for range ticker.C {
			rateLM.cleanupExpired()
		}
	}()
	return &rateLM
}

func (rlm *rateLimitterMap) get(userId string, tokens int, fillrate float64) *rateLimitterStruct {
	rlm.Lock()
	defer rlm.Unlock()
	rl, ok := rlm.m[userId]
	if !ok {
		rl = &rateLimitterStruct{
			tokens:      tokens,
			fillRate:    fillrate,
			lastUpdated: time.Now(),
		}
		rlm.m[userId] = rl
	}
	return rl
}

func (rlm *rateLimitterMap) cleanupExpired() {
	rlm.Lock()
	defer rlm.Unlock()
	now := time.Now()
	for userId, rl := range rlm.m {
		timeElapsed := now.Sub(rl.lastUpdated)
		if timeElapsed.Minutes() >= 1 {
			delete(rlm.m, userId)
		}
	}
}

func (r *rateLimitterStruct) allowRequest() bool {
	now := time.Now()
	r.tokens += int(r.fillRate * now.Sub(r.lastUpdated).Seconds())
	if r.tokens < maxTokens {
		return false
	}
	r.lastUpdated = now
	if r.tokens > 0 {
		r.tokens--
		return true
	}
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
	rl := rlm.get(requestBody.UserId, 0, 10)
	if rl.allowRequest() {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Request processed successfully"))
	} else {
		http.Error(w, "Rate limit exceeded, Please try after some time", http.StatusTooManyRequests)
		return
	}
}
