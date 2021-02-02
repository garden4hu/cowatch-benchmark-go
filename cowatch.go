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

	when, err := func() (time.Time, error) {
		// multi-point clients should check the start time
		if conf.Mode == 1 {
			getTime, err := time.Parse(time.RFC3339, conf.StartTimeRoom)
			if err != nil {
				return time.Time{}, errors.New("failed to parse the start Time for request room. check the configure file please")
			}
			_, err = time.Parse(time.RFC3339, conf.StartTimeUser)
			if err != nil {
				return time.Time{}, errors.New("failed to parse the start Time for request user. check the configure file please")
			}
			if getTime.UnixNano() < time.Now().UnixNano() {
				return time.Time{}, errors.New("the start time(" + getTime.Local().String() + ") for creating room is expired, exit now")
			}
			return getTime, err
		}
		return time.Now().Add(2 * time.Second), nil
	}()
	if err != nil {
		return err
	}
	if err := rm.RequestAllRooms(when, 0); err != nil {
		return err
	}
	roomManager = rm
	return nil
}

// GetUsers try to request users and communicate with coWatch server
func getUsers(conf *Config) error {
	when, err := func() (time.Time, error) {
		if conf.Mode == 1 {
			when, _ := time.Parse(time.RFC3339, conf.StartTimeUser)
			if when.UnixNano() < time.Now().UnixNano() {
				return time.Time{}, errors.New("the start time" + when.Local().String() + "for creating users is expired, exit now")
			}
		}
		return time.Now().Add(3 * time.Second), nil
	}()
	if err != nil {
		return err
	}
	if roomManager == nil {
		return errors.New("create room manager firstly please, exit now")
	}
	for i := 0; i < len(roomManager.Rooms); i++ {
		go roomManager.Rooms[i].UsersConnection(when, 0)
	}
	return nil
}
