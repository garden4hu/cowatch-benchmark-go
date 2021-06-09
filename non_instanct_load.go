package main

import (
	"context"
	"fmt"
	"time"
)

// NonInstanceLoading  will test the top capacity of the server.
// It will create room and users continually util the server cannot allocate new space for room/users.
// The size of room and user are not necessary for this testing mode.
func NonInstanceLoading(conf *Config, ctx context.Context) {
	if roomManager == nil {
		return
	}

	err := roomManager.RequestAllRooms(time.Now())
	if err != nil {
		fmt.Println("failed to create room")
		return
	}
	// request user serially
	//room.UsersConnection(time.Now(), false)
	//fmt.Println("create a room, user=", len(room.Users), "  total rooms=", len(roomManager.Rooms))
	//if len(room.Users) == 0 {
	//	fmt.Print("failed to request users(0 indeed), maybe beyond the capacity of server. Program will not request new room")
	//	roomManager.Rooms = roomManager.Rooms[:len(roomManager.Rooms)-1]
	//	return
	//}
	for roomManager.CheckCreatingRoomsOK() == false {
		fmt.Println("not yet create room ok")
		time.Sleep(1 * time.Second)
	}
	fmt.Println("online time = ", conf.OnlineTime)
	webSocketRunningDuration.Reset(time.Duration(conf.OnlineTime) * time.Second)
	getUsersSerial(conf, ctx)
}

func getUsersSerial(conf *Config, ctx context.Context) {
	if err := getUsers(conf, ctx); err != nil {
		fmt.Println(err)
		return
	}
}
