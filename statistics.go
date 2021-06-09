package main

import (
	"fmt"
	"time"

	cb "github.com/garden4hu/cowatchbenchmark"
)

type statistic struct {
	lastOnlineUsers int
	lastSecondRooms int
}

func printLogMessage(roomManager *cb.RoomManager) {
	if roomManager == nil {
		return
	}
	if len(roomManager.Rooms) == 0 {
		return
	}
	now := time.Now().String()
	if roomManager.CheckCreatingRoomsOK() == false {
		roomSize := len(roomManager.Rooms)
		addedRooms := roomSize - analytics.lastSecondRooms
		info := fmt.Sprintf("%s [room information] created:%d  wanted:%d  percent:%d%% added:%d time_consumption:%s", now, roomSize, roomManager.RoomSize, roomSize*100/roomManager.RoomSize, addedRooms, roomManager.GetCreatingRoomAvgDuration().String())
		fmt.Println(info)
		analytics.lastSecondRooms = roomSize
	}
	if onlineUser != 0 {
		now := time.Now().String()
		addedUsers := onlineUser - analytics.lastOnlineUsers
		if addedUsers == 0 {
			return
		}
		total := len(roomManager.Rooms) * roomManager.UserSize
		info := fmt.Sprintf("%s [user information] created:%d  wanted:%d  percent:%d%% new added:%d time_consumption:%s [room=%d]", now, onlineUser, total, onlineUser*100/(total), addedUsers, roomManager.GetCreatingRoomAvgDuration().String(), len(roomManager.Rooms))
		fmt.Println(info)
		analytics.lastOnlineUsers = onlineUser
	}
}
