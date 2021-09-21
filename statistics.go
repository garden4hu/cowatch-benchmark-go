package main

import (
	"fmt"
	"time"
)

type statistic struct {
	lastOnlineUsers int
	lastSecondRooms int
	lastAddPercent  int
}

func printLogMessage(roomManager *roomManager) {
	if roomManager == nil {
		return
	}
	if len(roomManager.Rooms) == 0 {
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")

	if roomManager.CheckCreatingRoomsOK() == false {
		roomSize := len(roomManager.Rooms)
		info := fmt.Sprintf("\r%s \t[房间信息] \t创建房间:%d  \t期望数:%d  \t完成比例:%d%% \t \t平均创建耗时:%s", now, roomSize, roomManager.roomSize, roomSize*100/roomManager.roomSize, roomManager.GetCreatingRoomAvgDuration().String())
		fmt.Println(info)
		analytics.lastSecondRooms = roomSize
	}
	if roomManager.CheckCreatingRoomsOK() == true {
		if onlineUser != 0 {
			addedUsers := onlineUser - analytics.lastOnlineUsers

			if addedUsers == 0 {
				return
			}
			total := len(roomManager.Rooms) * roomManager.userSize
			if onlineUser*100/(total)-analytics.lastAddPercent < 1 {
				return
			}
			analytics.lastAddPercent = onlineUser * 100 / (total)
			info := fmt.Sprintf("%s \t[用户信息] \t在线数:%d  \t期望数:%d  \t在线比例(百分比):%d \t \t耗时:%s \t[房间总数=%d/每房间人数=%d]", now, onlineUser, total, onlineUser*100/(total), roomManager.GetCreatingRoomAvgDuration().String(), len(roomManager.Rooms), roomManager.userSize)
			fmt.Println(info)
			analytics.lastOnlineUsers = onlineUser
		}
	}
}
