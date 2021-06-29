package main

import (
	"fmt"
	"testing"
	"time"
)

func TestRoomManager_RequestRoomsFromServer(t *testing.T) {
	rm := newRoomManager("https://cowatch_server", 3000, 1, 20, 1, 25000, 25000, "app_id", 1, 1)
	err := rm.requestAllRooms(time.Now())
	if err != nil {
		t.Errorf(err.Error())
	}
	fmt.Println("target room size :=", rm.roomSize, "real room size = : ", len(rm.Rooms))
}
