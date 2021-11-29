package main

import (
	"github.com/sirupsen/logrus"
)

type statistic struct {
	printed         bool
	lastRoomSize    int
	lastOnlineUsers int
	lastUsersPingOk int
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

	roomOKRatio := roomSize * 100 / roomManager.roomSize
	roomTimeCostAvg := roomManager.GetCreatingRoomAvgDuration().Milliseconds()
	onlineUsersRatio := onlineUser * 100 / (total)
	usersPingOk := onlineUserPingOK
	usersWSCostAvg := roomManager.GetCreatingRoomAvgDuration().Milliseconds()
	usersNewAdd := addedUsers

	checkNoChanging := func() bool {
		if analytics.lastRoomSize == roomSize && analytics.lastOnlineUsers == onlineUser && analytics.lastUsersPingOk == onlineUserPingOK {
			return false
		}
		return true
	}

	if !checkNoChanging() {
		if analytics.printed {
			return
		}
	} else {
		analytics.printed = false
	}
	logA.WithFields(logrus.Fields{
		"created_room":          roomSize,
		"created_room_ratio(%)": roomOKRatio,
		"HTTP_time_cost":        roomTimeCostAvg,
		"online_users":          onlineUser,
		"online_users_ratio(%)": onlineUsersRatio,
		"users_ping":            usersPingOk,
		"WS_time_cost":          usersWSCostAvg,
		"users_added":           usersNewAdd,
		"transport_pool_size":   totalTransport,
	}).Println()

	analytics.lastRoomSize = roomSize
	analytics.lastOnlineUsers = onlineUser
	analytics.lastUsersPingOk = usersPingOk
	analytics.printed = true
}
