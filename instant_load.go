package main

import (
	"context"
	"time"
)

func getRoomsParallel(conf *Config, ctx context.Context) {
	if err := getRooms(ctx, conf); err != nil {
		log.Errorln(err.Error())
		return
	}
	time.Sleep(2 * time.Second)
	getUsersParallel(conf, ctx)
}

func getUsersParallel(conf *Config, ctx context.Context) {
	if err := getUsers(conf, ctx); err != nil {
		log.Infoln(err)
		return
	}
}
