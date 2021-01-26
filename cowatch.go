package main

import (
	"errors"
	"time"

	cb "github.com/garden4hu/cowatchbenchmark"
)

func getRooms(conf *Config) error {
	if 0 == conf.Room {
		return nil
	}
	rm := cb.NewRoomManager(conf.Host, conf.Room, conf.User, conf.Len, conf.Freq, 25, 45)

	when, err := time.Parse(time.RFC3339, conf.StartTimeRoom)
	if err != nil {
		return errors.New("failed to parse the start Time for request room. check the configure file please")
	}
	_, err = time.Parse(time.RFC3339, conf.StartTimeUser)
	if err != nil {
		return errors.New("failed to parse the start Time for request user. check the configure file please")
	}
	if when.UnixNano() < time.Now().UnixNano() {
		return errors.New("the start time(" + when.Local().String() + ") for creating room is expired, exit now")
	}

	if err = rm.RequestRoomsFromServer(when); err != nil {
		return err
	}
	roomManager = rm
	return nil
}

func getUsers(conf *Config) error {
	when, _ := time.Parse(time.RFC3339, conf.StartTimeUser)
	if when.UnixNano() < time.Now().UnixNano() {
		return errors.New("the start time" + when.Local().String() + "for creating users is expired, exit now")
	}
	if roomManager == nil {
		return errors.New("create room manager firstly please, exit now")
	}
	for i := 0; i < len(roomManager.Rooms); i++ {
		go roomManager.Rooms[i].CreateUsers(when)
	}
	return nil
}
