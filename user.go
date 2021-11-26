package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// newUser construct the user
func newUser(p *roomUnit) *userInfo {
	u := &userInfo{uid: getHostId(), hostCoWatch: false, connected: false, readyForMsg: false, expireTimer: time.NewTicker(24 * time.Hour), msgPool: p.msgPool}
	u.id = int(atomic.AddInt32(&gID, 1))
	u.name = strconv.Itoa(u.id)
	u.room = p
	u.lock = &sync.Mutex{}
	return u
}

// usersConnection try to connect to the server and exchange message.
// param when is the time for requesting of websocket concurrently
// param mode is the mode for requesting. 0 means parallel and 1 means serial
func usersConnection(p *roomUnit, start chan struct{}, ctx context.Context, wg *sync.WaitGroup) {
	if p.rm.parallelRequest {
		defer wg.Done()
	}

	// create users
	var wg2 sync.WaitGroup // 用于并发请求，确保所有的 goroutine 同时发起请求，而不会出现开始并发请求时，有的 goroutine 还没有构造好 ws 句柄
	for i := 0; i < p.usersCap; i++ {
		if p.rm.parallelRequest {
			wg2.Add(1)
		}
		user := p.users[i]
		if p.rm.parallelRequest {
			// go userJoinRoom(ctx, u, p.rm.parallelRequest, start, &wg2)
			go func() {
				if wg != nil {
					wg.Done() // create goroutine done
				}
				if start != nil {
					<-start // waiting for starting
				}
				joinRoom(ctx, user)
			}()
		} else {
			joinRoom(ctx, user)
		}
	}

	if p.rm.parallelRequest {
		wg2.Wait()
	}
}

// userJoinRoom join the room on the websocket server
func userJoinRoom(ctx context.Context, user *userInfo, parallel bool, start chan struct{}, wg *sync.WaitGroup) {
	// if request users concurrently, goroutine should be waited
	if wg != nil {
		wg.Done() // create goroutine done
	}
	if parallel {
		if start != nil {
			<-start // waiting for starting
		}
	}

}

func joinRoom(ctx context.Context, user *userInfo) {
	r := user.room
	startJoin := time.Now()
	conn, err := createWsConn(ctx, user)
	if err == nil {
		user.wsReqTimeOH = time.Now().Sub(startJoin)
		logIn.Debugln("goID:", user.id, " ws_req_time(ms):", user.wsReqTimeOH.Milliseconds())
		go processMessageWorker(user, conn)
		user.connectionDuration = time.Since(startJoin)
	}
	f := func() {
		if user.connected {
			if conn != nil {
				user.lock.Lock()
				e := conn.WriteMessage(websocket.TextMessage, generateMessage(r))
				user.lock.Unlock()
				if e == nil {
					user.messageTimer.Reset(r.msgSendingInternal)
				}
			}
		}
	}
	user.messageTimer = time.AfterFunc(r.msgSendingInternal, f) // start a schedule message timer
}

func createWsConn(ctx context.Context, p *userInfo) (*websocket.Conn, error) {
	r := p.room
	// set ws/wss url param
	v := url.Values{}
	v.Add("uid", strconv.Itoa(p.uid))
	v.Add("name", p.name)
	v.Add("version", r.sdkVersion)
	v.Add("roomId", r.ns)
	v.Add("EIO", "4") // using socket.io V4
	v.Add("transport", "websocket")
	v.Add("hostname", "qa.visualon.com") // God knows why the hostname is set like this
	u := url.URL{Host: r.address, Path: "/socket.io/", ForceQuery: true, RawQuery: v.Encode()}

	switch r.schema {
	case "http":
		u.Scheme = "ws"
		break
	case "https":
		u.Scheme = "wss"
		break
	default:
		u.Scheme = "wss"
		break
	}

	//pool := &sync.Pool{New: func() interface{} {
	//	s := new(poolData)
	//	return &s
	//}}
	dialer := &websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  r.wsTimeout,
		EnableCompression: true,
		WriteBufferSize:   128, // because the longest size of message is 250, 128 is half of the largest size of message according to the guide of gorilla
		WriteBufferPool:   &sync.Pool{},
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
	}
	// set http->websocket header
	rq := http.Header{}
	rq.Add("Accept-Encoding", "gzip, deflate, br")
	rq.Add("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6")
	rq.Add("Cache-Control", "no-cache")
	// rq.Add("Connection","Upgrade")
	rq.Add("Pragma", "no-cache")
	for key, val := range r.rm.httpHeaders {
		rq.Add(key, val)
	}
	//rq.Add("Sec-WebSocket-Extensions","permessage-deflate; client_max_window_bits") // enable compress
	var conn *websocket.Conn
	var err error = nil
	for i := 0; i < 3; i++ {
		conn, _, err = dialer.DialContext(ctx, u.String(), rq)
		if err != nil {
			log.Errorln("websocket dialer error: ", err)
			if i < 3 {
				time.Sleep(50 * time.Millisecond)
				continue
			}
		} else {
			break
		}
	}
	if err != nil {
		log.Errorln("failed to dial websocket:", err)
		return nil, err
	}
	return conn, nil
}
