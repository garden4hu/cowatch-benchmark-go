package main

import (
	"context"
	"time"
)

// NonInstanceLoading  will test the top capacity of the server.
// It will create room and users continually util the server cannot allocate new space for room/users.
// The size of room and user are not necessary for this testing mode.
func NonInstanceLoading(conf *Config, ctx context.Context) {
	if rm == nil {
		return
	}

	err := rm.requestAllRooms(ctx, time.Now())
	if err != nil {
		log.Errorln("failed to create room")
		return
	}
	// request user serially
	//room.usersConnection(time.Now(), false)
	//fmt.Println("create a room, user=", len(room.users), "  total roomSize=", len(rm.Rooms))
	//if len(room.users) == 0 {
	//	fmt.Print("failed to request users(0 indeed), maybe beyond the capacity of server. Program will not request new room")
	//	rm.Rooms = rm.Rooms[:len(rm.Rooms)-1]
	//	return
	//}
	for rm.CheckCreatingRoomsOK() == false {
		log.Errorln("not yet create room ok")
		time.Sleep(1 * time.Second)
	}
	webSocketRunningDuration.Reset(time.Duration(conf.OnlineTime) * time.Second)
	getUsersSerial(conf, ctx)
}

func getUsersSerial(conf *Config, ctx context.Context) {
	if err := getUsers(conf, ctx); err != nil {
		log.Errorln(err)
		return
	}
}
