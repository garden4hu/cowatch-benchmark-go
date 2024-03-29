package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// use "log.SetOutput(ioutil.Discard)" in main to disable log output

type roomManager struct {
	addr             string
	roomSize         int
	userSize         int
	messageLength    int
	frequency        int
	lockRoom         sync.Mutex
	Rooms            []*roomUnit
	start            bool
	httpTimeout      time.Duration
	websocketTimeout time.Duration
	appID            string
	singleClientMode int
	parallelRequest  bool
	sdkVersion       string
	notifyUsersAdd   <-chan int // chan 大小为 用户总数
	notifyRoomAdd    <-chan int

	notifyUserPingOK chan int

	messagePool *sync.Pool

	// Transport
	tr *http.Transport

	// playback assets

	playBackContent string
	// for internal usage
	notifyUserAdd            chan int
	creatingRoomsOK          bool
	creatingUsersOK          bool
	finishedReqRoomRoutines  int
	finishedReqUsersRoutines int
	createRoomExtraData      map[string]string
	httpHeaders              map[string]string
}

// newRoomManager will return a roomManager
func newRoomManager(conf *Config) *roomManager {
	rm := &roomManager{addr: conf.Address, roomSize: conf.Rooms, userSize: conf.UsersPerRoom, messageLength: conf.Len, frequency: conf.Freq, start: false, httpTimeout: time.Second * time.Duration(conf.HttpTimeOut), websocketTimeout: time.Second * time.Duration(conf.WSTimeOut), appID: conf.AppID, singleClientMode: conf.SingleClientMode, parallelRequest: conf.ParallelMode == 1, sdkVersion: conf.SDKVersion}
	rm.creatingRoomsOK = false
	rm.creatingUsersOK = false
	rm.notifyUserAdd = make(chan int, conf.Rooms*conf.UsersPerRoom)
	rm.notifyUsersAdd = rm.notifyUserAdd
	rm.finishedReqRoomRoutines = 0
	rm.finishedReqUsersRoutines = 0
	var playBackAsset ContentInfo
	playBackAsset.Videos = conf.Video
	if len(playBackAsset.Videos.Links) != 0 {
		if asset, err := json.Marshal(playBackAsset); err == nil {
			rm.playBackContent = string(asset)
		}
	}
	tr := &http.Transport{}
	rm.tr = tr
	pingChannelSize := func() int {
		if conf.Rooms*conf.UsersPerRoom > 8 {
			return conf.Rooms * conf.UsersPerRoom / 4
		}
		return 4
	}()
	rm.notifyUserPingOK = make(chan int, pingChannelSize)
	return rm
}

func (p *roomManager) Close() {
	close(p.notifyUserAdd)
}

func (p *roomManager) CheckCreatingRoomsOK() bool {
	return p.creatingRoomsOK
}

func (p *roomManager) CheckCreatingUsersOK() bool {
	return p.creatingUsersOK
}

// requestAllRooms will request all the roomSize from the server.
// param when is the start time for request room from server concurrently [Only useful when parallel is true]
// param mode is the mode for request room. true means parallel and false means serial
func (p *roomManager) requestAllRooms(ctx context.Context, when time.Time) error {
	var wg sync.WaitGroup
	start := make(chan struct{})

	// for serial request
	var leftGoroutine int32
	atomic.StoreInt32(&leftGoroutine, int32(p.roomSize))

	for i := 0; i < p.roomSize; {
		// all goroutines will send request in the same time
		if p.parallelRequest == true {
			wg.Add(1)
			go p.requestRoom(ctx, &wg, start)
			i++
		} else {
			//  线程创建，为了提高速度，一次创建 8 个
			for j := i; j < i+8 && j < p.roomSize; j++ {
				// go p.RequestRoom()
				go func() {
					r := newRoom(p.addr, p.httpTimeout, p.websocketTimeout, p.userSize, p.messageLength, p.frequency, p.appID, p)
					_ = r.requestCreateRoom(ctx)
					//mtx.Lock()
					//leftGoroutine -= 1
					//mtx.Unlock()
					atomic.AddInt32(&leftGoroutine, -1)
				}()
			}
			i += 8
			time.Sleep(20 * time.Millisecond)
		}
	}
	if p.parallelRequest == true && p.singleClientMode == 0 {
		if p.singleClientMode == 0 { // 多台测试主机并发测试，需要等待特定时刻并发请求
			now := time.Now()
			if now.UnixNano() > when.UnixNano() {
				return errors.New("current time is newer than the schedule time. Operation of creating roomSize will not be executed")
			}
			time.Sleep(time.Nanosecond * time.Duration(when.UnixNano()-now.UnixNano()))
		}
	}

	close(start) // 开始并发创建请求

	if p.parallelRequest == true {
		wg.Wait()
	} else {
		for leftGoroutine != 0 {
			time.Sleep(1 * time.Second)
		}
	}
	p.creatingRoomsOK = true
	return nil
}

type transprotPool struct {
	*http.Transport
}

func (p *roomManager) requestRoom(ctx context.Context, wg *sync.WaitGroup, start chan struct{}) {
	r := newRoom(p.addr, p.httpTimeout, p.websocketTimeout, p.userSize, p.messageLength, p.frequency, p.appID, p)
	if wg != nil {
		defer wg.Done()
	}
	if p.parallelRequest {
		<-start // 需要等待
	}
	_ = r.requestCreateRoom(ctx)
}
