package main

import "time"

// ---------------------------------------- for statistics --------------------------------------------

func (p *roomManager) GetCreatedRooms() int {
	return len(p.Rooms)
}

func (p *roomManager) GetTotalUsers() int {
	total := 0
	for i := 0; i < len(p.Rooms); i++ {
		total += len(p.Rooms[i].users)
	}
	return total
}

// GetCreatingRoomAvgDuration return the average time consumption of all roomSize which are created successfully
func (p *roomManager) GetCreatingRoomAvgDuration() time.Duration {
	if len(p.Rooms) == 0 {
		return time.Duration(0)
	}
	var totalDuration time.Duration = 0
	for i := 0; i < len(p.Rooms); i++ {
		totalDuration += p.Rooms[i].connectionDuration
	}
	return time.Duration(int64(totalDuration) / int64(len(p.Rooms)))
}

func (p *roomManager) GetCreatingUsersAvgDuration() time.Duration {
	if len(p.Rooms) == 0 {
		return time.Duration(0)
	}
	var totalDuration time.Duration = 0
	for i := 0; i < len(p.Rooms); i++ {
		totalDuration += p.Rooms[i].usersAvgConnectionDuration()
	}
	return time.Duration(int64(totalDuration) / int64(len(p.Rooms)))
}

// usersAvgConnectionDuration return the average time consumption of all roomSize which are created successfully
func (p *roomUnit) usersAvgConnectionDuration() time.Duration {
	var totalDuration time.Duration = 0
	usersSize := len(p.users)
	if usersSize == 0 {
		return totalDuration
	}
	for i := 0; i < usersSize; i++ {
		totalDuration += p.users[i].connectionDuration
	}
	return time.Duration(int64(totalDuration) / int64(usersSize))
}
