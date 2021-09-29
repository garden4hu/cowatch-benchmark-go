package main

import (
	"github.com/sirupsen/logrus"
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
	// now := time.Now().Format("2006-01-02 15:04:05")

	roomSize := len(roomManager.Rooms)
	addedUsers := onlineUser - analytics.lastOnlineUsers
	total := len(roomManager.Rooms) * roomManager.userSize
	logA.WithFields(logrus.Fields{
		"已创建房间":          roomSize,
		"room比例":         roomSize * 100 / roomManager.roomSize,
		"http请求平均耗时":     roomManager.GetCreatingRoomAvgDuration().Seconds(),
		"用户在线数":          onlineUser,
		"user在线比例":       onlineUser * 100 / (total),
		"user_ping正常数":   onlineUserPingOK,
		"ws创建平均耗时":       roomManager.GetCreatingRoomAvgDuration().Seconds(),
		"新增用户数":          addedUsers,
		"当前总的 transport": totalTransport,
	}).Println()

	analytics.lastSecondRooms = roomSize
	analytics.lastAddPercent = onlineUser * 100 / (total)
	analytics.lastOnlineUsers = onlineUser

}
