package database


import (
	"context"
	"github.com/go-redis/redis/v8"
	"os"
)


var Ctx = context.Background()


func CreateClient(dbNo int) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
        Addr:     os.Getenv("REDIS_ADDR"),
        Password: os.Getenv("REDIS_PASSWORD"),
        DB:       dbNo,
    })

    return rdb
}