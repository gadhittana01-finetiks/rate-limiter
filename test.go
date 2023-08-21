package main

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

func getTime() (map[string]interface{}, error) {

	var maxLimit int64

	var limitTime int64

	maxLimit = 5

	limitTime = 60

	systemUser := "john_doe"

	uniqueKey := systemUser

	ctx := context.Background()

	redisClient := redis.NewClient(&redis.Options{

		Addr: "localhost:6379",

		Password: "",

		DB: 0,
	})

	var counter int64

	counter, err := redisClient.Get(ctx, uniqueKey).Int64()

	if err == redis.Nil {

		err = redisClient.Set(ctx, uniqueKey, 1, time.Duration(limitTime)*time.Second).Err()

		if err != nil {

			return nil, err

		}

		counter = 1

	} else if err != nil {

		return nil, err

	} else {

		if counter >= maxLimit {

			return nil, errors.New("Limit reached.")

		}

		counter, err = redisClient.Incr(ctx, uniqueKey).Result()

		if err != nil {

			return nil, err

		}

	}

	dt := time.Now()

	res := map[string]interface{}{

		"data": dt.Format("2006-01-02 15:04:05"),
	}

	return res, nil

}
