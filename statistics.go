package main

import (
	"fmt"
	"github.com/fatih/color"
	cb "github.com/garden4hu/cowatchbenchmark"
	"time"
)

func printLogMessage(roomManager *cb.RoomManager, conf *Config) {
	if roomManager == nil || conf == nil {
		return
	}
	if len(roomManager.Rooms) == 0 {
		color.Set(color.FgYellow)
		fmt.Println("no room was created")
		color.Unset()
		return
	}
	if conf.TestMode == 0 {
		color.Set(color.FgYellow)
		defer color.Unset()
		h, m, s := time.Now().Clock()
		info := fmt.Sprintf("%d:%d:%d [room information] created:%d  wanted:%d  percent:%d time_consumption:%s", h, m, s, len(roomManager.Rooms), conf.Room, len(roomManager.Rooms)*100/conf.Room, roomManager.GetCreatingRoomAvgDuration().String())
		fmt.Println(info)

		var avgUsersConsume time.Duration
		var userSize int
		for i := 0; i < len(roomManager.Rooms); i++ {
			avgUsersConsume += roomManager.Rooms[i].GetUsersAvgConnectionDuration()
			userSize += len(roomManager.Rooms[i].Users)
		}
		avgUsersConsume /= time.Duration(len(roomManager.Rooms))
		h, m, s = time.Now().Clock()
		info = fmt.Sprintf("%d:%d:%d [user information] created:%d  wanted:%d percent:%d time_consumption:%s", h, m, s, userSize, conf.Room*conf.User, userSize*100/(conf.Room*conf.User), avgUsersConsume.String())
		fmt.Println(info)
	} else {

	}
}

func PrintSerialInfo() {

}
