//go:build ignore

package main

import (
	"context"
	"log"
	"time"

	"github.com/dotcommander/gokart/cache"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	ctx := context.Background()
	c, err := cache.Open(ctx, "localhost:6379")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	if err := c.Client().Set(ctx, c.Key("greeting"), "hello", time.Hour).Err(); err != nil {
		log.Fatal(err)
	}
	value, err := c.Client().Get(ctx, c.Key("greeting")).Result()
	if err != nil {
		log.Fatal(err)
	}
	log.Print(value)

	if err := c.SetJSON(ctx, "user:1", User{ID: 1, Name: "Alice"}, time.Hour); err != nil {
		log.Fatal(err)
	}
}
