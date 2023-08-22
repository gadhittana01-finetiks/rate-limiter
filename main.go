package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-redis/redis/v8"
)

func response(w http.ResponseWriter, message string, status int) {
	data := map[string]interface{}{
		"data": message,
	}

	respBytes, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(respBytes)
}

func generateOTP(max int) string {
	var table = [...]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}
	b := make([]byte, max)
	n, err := io.ReadAtLeast(rand.Reader, b, max)
	if n != max {
		panic(err)
	}

	for i := 0; i < len(b); i++ {
		b[i] = table[int(b[i])%len(table)]
	}
	return string(b)
}

func newRedisClient(host string, password string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     host,
		Password: password,
	})
	return client
}

func rateLimiter(cache *redis.Client, ctx context.Context, userID string) (bool, error) {
	var res bool
	expired := 20 * time.Second
	max := 3

	curr, err := cache.Get(ctx, userID).Int()
	if err != nil && err != redis.Nil {
		return res, err
	}

	if curr == max {
		return false, nil
	}

	if curr == 0 {
		err := cache.Incr(ctx, userID).Err()
		if err != nil {
			return res, err
		}

		err = cache.Expire(ctx, userID, expired).Err()
		if err != nil {
			return res, err
		}
		return true, nil
	}

	err = cache.Incr(ctx, userID).Err()
	if err != nil {
		return res, err
	}
	return true, nil
}

func getRemainTime(cache *redis.Client, ctx context.Context, userID string) time.Duration {
	return cache.TTL(ctx, userID).Val()
}

func main() {
	var redisHost = "localhost:6379"
	var redisPassword = ""

	cache := newRedisClient(redisHost, redisPassword)
	fmt.Println("redis client initialized")

	router := chi.NewRouter()

	router.Get("/request-otp", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			response(w, "user id is empty", http.StatusBadRequest)
			return
		}

		isAllowed, err := rateLimiter(cache, ctx, userID)
		if err != nil {
			response(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !isAllowed {
			remainTime := getRemainTime(cache, ctx, userID)
			response(w, fmt.Sprintf("Please wait until %s", remainTime), http.StatusBadRequest)
			return
		}

		otp := generateOTP(6)
		// TODO:
		// isSent: bool
		// ttl_left: if true 0 else ttl redis

		response(w, otp, http.StatusBadRequest)
	})

	port := 8000
	log.Println("Serving HTTP on ports :" + strconv.Itoa(port))
	http.ListenAndServe(fmt.Sprintf(":%d", port), router)
}
