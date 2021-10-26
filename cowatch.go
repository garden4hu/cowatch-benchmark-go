package main

import (
	"context"
	"errors"
	"sync"
	"time"
)

func getRooms(ctx context.Context, conf *Config) error {
	if rm == nil {

		return errors.New("create roomManager please")
	}
	if 0 == conf.Rooms {
		return nil
	}

	when, err := func() (time.Time, error) {
		// 定时并发请求创建room
		if conf.ParallelMode == 1 && conf.SingleClientMode == 0 {
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
	if err := rm.requestAllRooms(ctx, when); err != nil {
		return err
	}
	return nil
}

// GetUsers try to request users and communicate with coWatch server
func getUsers(conf *Config, ctx context.Context) error {
	// 解析字符串获得 utc 标准时间结构
	when, err := func() (time.Time, error) {
		if conf.ParallelMode == 1 && conf.SingleClientMode == 0 {
			when, _ := time.Parse(time.RFC3339, conf.StartTimeUser)
			if when.UnixNano() < time.Now().UnixNano() {
				return time.Time{}, errors.New("the start time" + when.Local().String() + "for creating users is expired, exit now")
			}
		}
		return time.Now(), nil
	}()
	if err != nil {
		return err
	}

	if rm == nil {
		return errors.New("create room manager firstly please, exit now")
	}
	ch := make(chan struct{})
	if conf.ParallelMode == 0 {
		for i := 0; i < len(rm.Rooms); i++ {
			go usersConnection(rm.Rooms[i], ch, ctx, nil)
			time.Sleep(15 * time.Millisecond)
		}
	} else if conf.ParallelMode == 1 {
		var wg sync.WaitGroup // Wait for the websocket handles of all users in the same room to be constructed
		for i := 0; i < len(rm.Rooms); i++ {
			wg.Add(1)
			go usersConnection(rm.Rooms[i], ch, ctx, &wg)
			wg.Wait()
		}
	} else if conf.ParallelMode == 2 {
		// 以秒为单位创建 ws, 采取动态的方式增加/降低创建 websocket 的速度
		// 策略: 在 1s 内实现当前设置的目标连接数 n ，则 n x 1.35;
		// 如果在 1s 内没有实现目标链接，则 n / 1.15。（经验值）
		// 从 n == len(room.users) 开始
		roomIndex := 0
		userIndex := 0
		fetchUser := func() *userInfo {
			if userIndex == conf.UsersPerRoom {
				userIndex = 0
				roomIndex++
			}
			if roomIndex == conf.Rooms {
				return nil
			}
			u := rm.Rooms[roomIndex].users[userIndex]
			userIndex++
			return u
		}

		joinRoomBatch := func(ctx context.Context, n int) (d time.Duration, e error) {
			now := time.Now()
			defer func() {
				d = time.Since(now)
			}()
			var wg sync.WaitGroup
			for i := 0; i < n; i++ {
				user := fetchUser()
				if user == nil {
					return 0, errors.New("no user left")
				}
				wg.Add(1)
				go func() {
					joinRoom(ctx, user)
					wg.Done()
				}()
			}
			wg.Wait()
			return
		}

		base := conf.UsersPerRoom
		lastDuration := 0 * time.Millisecond
		for {
			d, e := joinRoomBatch(ctx, base)
			if e != nil { // means that there is no more user to create connection
				break
			}
			if d.Milliseconds() != 0 && base != 0 {
				log.Infof("join room in batch, size: %d, duration(ms):%d\n", base, d.Milliseconds())
			}
			if d < lastDuration {
				base = int(float32(base)*1.4) + 1
			} else {
				base = int(float32(base) / 1.15)
			}
			lastDuration = d

		}
	}

	// 并发请求
	if conf.ParallelMode == 1 && conf.SingleClientMode == 0 {
		// 分布式客户端，需要等待设置的同一时刻启动
		now := time.Now()
		if now.UnixNano() > when.UnixNano() {
			return errors.New("current time is newer than the schedule time. Operation of creating users will not be executed")
		}
		time.Sleep(time.Nanosecond * time.Duration(when.UnixNano()-now.UnixNano()))
	}
	close(ch) // 开始请求
	return nil
}
