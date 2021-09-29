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
	if conf.Rooms == 0 {
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
			go rm.Rooms[i].usersConnection(ch, ctx, nil)
			time.Sleep(15 * time.Millisecond)
		}
	} else if conf.ParallelMode == 1 {
		var wg sync.WaitGroup // Wait for the websocket handles of all users in the same room to be constructed
		for i := 0; i < len(rm.Rooms); i++ {
			wg.Add(1)
			go rm.Rooms[i].usersConnection(ch, ctx, &wg)
			wg.Wait()
		}
	} else if conf.ParallelMode == 2 {
		// 以秒为单位创建ws
		if conf.UsersPerRoom > 0 && conf.WsReqConcurrency >= 0 {
			for i := 0; i < len(rm.Rooms); {
				now := time.Now()
				for j := i; j < i+conf.WsReqConcurrency && j < len(rm.Rooms); j++ {
					go rm.Rooms[j].usersConnection(ch, ctx, nil)
				}
				i += conf.WsReqConcurrency
				if 1*time.Second > time.Since(now) {
					time.Sleep(1*time.Second - time.Since(now))
				}
			}
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
