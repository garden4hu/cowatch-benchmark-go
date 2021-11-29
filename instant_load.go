package main

import (
	"context"
	"time"
)

func getRoomsParallel(conf *Config, ctx context.Context) {
	if err := getRooms(ctx, conf); err != nil {
		logA.Errorln(err.Error())
		return
	}
	if len(rm.Rooms) == 0 {
		logA.Errorln("No room created, program will be stopped")
		exitFlag <- true
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
