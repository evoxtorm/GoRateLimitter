package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gorilla/mux"
)

const maxTokens = 10

type RequestBody struct {
	UserId   string                 `json:"userId"`
	Endpoint string                 `json:"endpoint`
	Limit    int                    `json: limit`
	Body     map[string]interface{} `json: body`
}

// type rateLimitterStruct struct {
// 	tokens int
// 	// fillRate    float64
// 	lastUpdated time.Time
// 	// requests    int
// }

type rateLimitterMap struct {
	mc *memcache.Client
}

var rlm = newRateLimitterMap()

// func newRateLimitter(tokens int, fillRate float64) *rateLimitterStruct {
func newRateLimitterMap() *rateLimitterMap {
	memcacheClient := memcache.New("localhost:11211")
	err := memcacheClient.FlushAll()
	if err != nil {
		fmt.Printf("Error flushing memcache: %v", err)
	}
	return &rateLimitterMap{memcacheClient}
}

func (rlm *rateLimitterMap) get(userId string, tokens int) (*memcache.Item, error) {
	key := "ratelimit_" + userId
	item, err := rlm.mc.Get(key)
	if err != nil {
		if err == memcache.ErrCacheMiss {
			// Construct a new item with the desired values
			item = &memcache.Item{
				Key:        key,
				Value:      []byte(fmt.Sprintf("%d", tokens)),
				Expiration: 60,
			}
			if err := rlm.mc.Set(item); err != nil {
				return nil, fmt.Errorf("error setting rate limiter for user %s: %v", userId, err)
			}
			return item, nil
		}
		return nil, fmt.Errorf("error getting rate limiter for user %s: %v", userId, err)
	}
	return item, nil
}

func (r *rateLimitterMap) allowRequest(item *memcache.Item) bool {
	if item != nil {
		tokenStr := string(item.Value)
		tokenInt, err := strconv.Atoi(tokenStr)
		if err != nil {
			fmt.Println("Error converting token to int:", err)
			return false
		}
		if tokenInt > 0 {
			tokenInt -= 1
			item.Value = []byte(fmt.Sprintf("%d", tokenInt))
			if err := r.mc.Set(item); err != nil {
				fmt.Println(err, "Error while setting the token")
				return false
			}
			return true
		}
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
	item, err := rlm.get(requestBody.UserId, requestBody.Limit)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Some error in getting the value", http.StatusTooManyRequests)
		return
	}
	if rlm.allowRequest(item) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Request processed successfully"))
	} else {
		http.Error(w, "Rate limit exceeded, Please try after some time", http.StatusTooManyRequests)
		return
	}
}
