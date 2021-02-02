package main

import (
	"fmt"
	cb "github.com/garden4hu/cowatchbenchmark"
	"time"
)

// NonInstanceLoading  will test the top capacity of the server.
// It will create room and users continually util the server cannot allocate new space for room/users.
// The size of room and user are not necessary for this testing mode.
func NonInstanceLoading(conf *Config) {
	roomManager = cb.NewRoomManager(conf.Host, 0, conf.User, conf.Len, conf.Freq, conf.HttpTimeOut, conf.WSTimeOut)

	for {
		var room *cb.RoomUnit
		var err error = nil
		for i := 0; i < 3; i++ {
			room, err = roomManager.RequestRoom()
			if err != nil && i < 3 {
				continue
			} else {
				break
			}
		}
		if room == nil {
			break
		}
		// request user serially
		room.UsersConnection(time.Now(), 1)
		fmt.Println("create a room, user=", len(room.Users), "  total rooms=", len(roomManager.Rooms))
		if len(room.Users) == 0 {
			fmt.Print("failed to request users(0 indeed), maybe beyond the capacity of server. Program will not request new room")
			roomManager.Rooms = roomManager.Rooms[:len(roomManager.Rooms)-1]
			return
		}
	}
}
