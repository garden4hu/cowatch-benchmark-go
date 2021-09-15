package main

import (
	"errors"
	"log"
	"sync"
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
	notifyUsersAdd   <-chan int // chan 大小为 用户总数

	// for internal usage
	notifyUserAdd            chan int
	creatingRoomsOK          bool
	creatingUsersOK          bool
	finishedReqRoomRoutines  int
	finishedReqUsersRoutines int
	createRoomExtraField     map[string]string
}

// newRoomManager will return a roomManager
func newRoomManager(addr string, room, user, msgLen, frequency, httpTimeout, webSocketTimeout int, appID string, singleClientMode int, parallel int) *roomManager {
	if room < 0 || user < 0 || frequency <= 0 {
		log.Fatalln("Invalid param")
		return nil
	}
	if httpTimeout > 60 || httpTimeout < 0 {
		httpTimeout = 60
	}
	if webSocketTimeout > 60 || webSocketTimeout < 0 {
		webSocketTimeout = 45
	}
	if frequency <= 0 {
		frequency = 1
	}
	rm := &roomManager{addr: addr, roomSize: room, userSize: user, messageLength: msgLen, frequency: frequency, start: false, httpTimeout: time.Second * time.Duration(httpTimeout), websocketTimeout: time.Second * time.Duration(webSocketTimeout), appID: appID, singleClientMode: singleClientMode, parallelRequest: parallel == 1}
	rm.creatingRoomsOK = false
	rm.creatingUsersOK = false
	rm.notifyUserAdd = make(chan int, room*user)
	rm.notifyUsersAdd = rm.notifyUserAdd
	rm.finishedReqRoomRoutines = 0
	rm.finishedReqUsersRoutines = 0
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
func (p *roomManager) requestAllRooms(when time.Time) error {
	var wg sync.WaitGroup
	start := make(chan struct{})

	// for serial request
	mtx := sync.Mutex{}
	leftGoroutine := p.roomSize

	for i := 0; i < p.roomSize; {
		// all goroutines will send request in the same time
		if p.parallelRequest == true {
			wg.Add(1)
			go p.requestRoom(&wg, start)
			i++
		} else {
			//  线程创建，为了提高速度，一次创建 8 个
			for j := i; j < i+8 && j < p.roomSize; j++ {
				// go p.RequestRoom()
				go func() {
					r := newRoom(p.addr, p.httpTimeout, p.websocketTimeout, p.userSize, p.messageLength, p.frequency, p.appID, p)
					_ = r.request()
					mtx.Lock()
					leftGoroutine -= 1
					mtx.Unlock()
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

func (p *roomManager) requestRoom(wg *sync.WaitGroup, start chan struct{}) {
	r := newRoom(p.addr, p.httpTimeout, p.websocketTimeout, p.userSize, p.messageLength, p.frequency, p.appID, p)
	if wg != nil {
		defer wg.Done()
	}
	if p.parallelRequest {
		<-start // 需要等待
	}
	_ = r.request()
}
