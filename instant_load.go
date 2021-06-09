package main

import (
	"context"
	"fmt"
	"time"
)

func getRoomsParallel(conf *Config, ctx context.Context) {
	if err := getRooms(conf); err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(2 * time.Second)
	getUsersParallel(conf, ctx)
}

func getUsersParallel(conf *Config, ctx context.Context) {
	if err := getUsers(conf, ctx); err != nil {
		fmt.Println(err)
		return
	}
}
